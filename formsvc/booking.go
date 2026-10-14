package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Appointment booking. The booking itself lives in EGroupware (app acbooking): services,
// bookable persons, their hours and calendars. formsvc only proxies, server to server, so the
// visitor's browser never talks to EGroupware (privacy rule 12), and sends the mails:
//
//	GET  /api/booking/catalog?lang=        services and persons (cached bookingCatalogTTL)
//	GET  /api/booking/slots?service=&person=&from=&to=   free slots (cached bookingSlotsTTL)
//	POST /api/booking/hold     form: anti-spam checks as the contact form, then EGroupware holds
//	                           the slot for some minutes and we mail the confirm link
//	POST /api/booking/confirm  token from the mailed link -> event in the calendar, mail with cancel link
//	POST /api/booking/cancel   token from the cancel link
//	POST /api/booking/booking  token -> the booking, for the confirm / cancel page
//
// The tokens travel in the fragment of the mailed links (#t=...) and only ever in POST bodies,
// so they never reach a server log; the pages act on them only by an explicit button press - a
// mail scanner that opens the link confirms nothing.
const (
	bookingCatalogTTL = 5 * time.Minute
	bookingSlotsTTL   = 30 * time.Second
)

type booking struct {
	baseURL string // .../groupdav.php/acbooking (empty = booking disabled)
	user    string
	token   string
	client  *http.Client
	now     func() time.Time

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	until  time.Time
	status int
	body   []byte
}

func newBooking(baseURL, user, token string) *booking {
	return &booking{
		baseURL: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		user:    user,
		token:   token,
		client:  &http.Client{Timeout: 15 * time.Second},
		now:     time.Now,
		cache:   map[string]cached{},
	}
}

func (b *booking) enabled() bool { return b != nil && b.baseURL != "" }

// egwError is what EGroupware answers on a refusal: {"error": 409, "code": "slot_taken", ...}
type egwError struct {
	Status int    `json:"error"`
	Code   string `json:"code"`
	Msg    string `json:"message"`
}

// call does one request to EGroupware and returns status and body.
func (b *booking) call(method, path string, payload any) (int, []byte, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, b.baseURL+path, body)
	if err != nil {
		return 0, nil, err
	}
	req.SetBasicAuth(b.user, b.token)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := b.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	return res.StatusCode, data, err
}

// cachedGet answers GETs from the cache; only 200s are cached.
func (b *booking) cachedGet(path string, ttl time.Duration) (int, []byte, error) {
	b.mu.Lock()
	if c, ok := b.cache[path]; ok && b.now().Before(c.until) {
		b.mu.Unlock()
		return c.status, c.body, nil
	}
	b.mu.Unlock()
	status, body, err := b.call("GET", path, nil)
	if err == nil && status == http.StatusOK {
		b.mu.Lock()
		if len(b.cache) > 2000 { // many different slot queries: start over rather than grow
			b.cache = map[string]cached{}
		}
		b.cache[path] = cached{until: b.now().Add(ttl), status: status, body: body}
		b.mu.Unlock()
	}
	return status, body, err
}

// forgetSlots drops the cached slots after a hold, confirm or cancel changed them.
func (b *booking) forgetSlots() {
	b.mu.Lock()
	for k := range b.cache {
		if strings.HasPrefix(k, "/slots") {
			delete(b.cache, k)
		}
	}
	b.mu.Unlock()
}

// --- the view of a booking EGroupware answers ------------------------------

type bookingView struct {
	ID      int    `json:"id"`
	Status  string `json:"status"`
	Service *struct {
		Title    string   `json:"title"`
		Duration int      `json:"durationMinutes"`
		Price    *float64 `json:"price"`
		Currency string   `json:"currency"`
		Video    bool     `json:"video"`
		Location *string  `json:"location"`
	} `json:"service"`
	Person *struct {
		Name string `json:"name"`
	} `json:"person"`
	Start       string `json:"start"`
	End         string `json:"end"`
	TimeZone    string `json:"timeZone"`
	Expires     string `json:"expires,omitempty"`
	CancelUntil string `json:"cancelUntil,omitempty"`
	Client      struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"client"`
	Lang         string `json:"lang"`
	ConfirmToken string `json:"confirmToken,omitempty"`
	CancelToken  string `json:"cancelToken,omitempty"`
	// AlreadyConfirmed is set by a repeated confirm (double click, reload, a mail scanner running JS)
	AlreadyConfirmed bool `json:"alreadyConfirmed,omitempty"`
}

// public is what the page gets to see: no tokens
func (v bookingView) public() map[string]any {
	m := map[string]any{"id": v.ID, "status": v.Status, "start": v.Start, "end": v.End, "timeZone": v.TimeZone}
	if v.Service != nil {
		m["service"] = v.Service.Title
		m["video"] = v.Service.Video
		if v.Service.Location != nil {
			m["location"] = *v.Service.Location
		}
	}
	if v.Person != nil {
		m["person"] = v.Person.Name
	}
	if v.Expires != "" {
		m["expires"] = v.Expires
	}
	if v.CancelUntil != "" {
		m["cancelUntil"] = v.CancelUntil
	}
	return m
}

// --- handlers --------------------------------------------------------------

var (
	reInt      = regexp.MustCompile(`^[0-9]{1,9}$`)
	reDate     = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	reTimeISO  = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}(:[0-9]{2})?([+-][0-9]{2}:[0-9]{2}|Z)$`)
	reBkToken  = regexp.MustCompile(`^[0-9]{1,10}\.[A-Za-z0-9_-]{43}$`)
	bookLangRe = regexp.MustCompile(`^(en|de|el)$`)
)

func bookLang(s string) string {
	if bookLangRe.MatchString(s) {
		return s
	}
	return "en"
}

// relay passes an EGroupware answer on: 200 bodies as they are, refusals as our formError.
func (s *server) relay(w http.ResponseWriter, status int, body []byte, err error) {
	if err != nil {
		log.Printf("booking: egroupware: %v", err)
		writeJSON(w, http.StatusBadGateway, &formError{Code: "server"})
		return
	}
	if status == http.StatusOK || status == http.StatusCreated {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
		return
	}
	writeJSON(w, s.refusal(status, body).Status, s.refusal(status, body))
}

// refusal turns an EGroupware error answer into the formError the page understands
func (s *server) refusal(status int, body []byte) *formError {
	var e egwError
	json.Unmarshal(body, &e)
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden || status >= 500 || e.Code == "":
		log.Printf("booking: egroupware answered %d: %.200s", status, body)
		return &formError{Status: http.StatusBadGateway, Code: "server"}
	case e.Code == "bad_name":
		return invalid("name")
	case e.Code == "bad_email":
		return invalid("email")
	}
	return &formError{Status: status, Code: e.Code}
}

func (s *server) bookingDisabled(w http.ResponseWriter) bool {
	if !s.booking.enabled() {
		writeJSON(w, http.StatusServiceUnavailable, &formError{Code: "unavailable"})
		return true
	}
	return false
}

func (s *server) handleBookingCatalog(w http.ResponseWriter, r *http.Request) {
	if s.bookingDisabled(w) {
		return
	}
	status, body, err := s.booking.cachedGet("/catalog?lang="+bookLang(r.URL.Query().Get("lang")), bookingCatalogTTL)
	s.relay(w, status, body, err)
}

func (s *server) handleBookingSlots(w http.ResponseWriter, r *http.Request) {
	if s.bookingDisabled(w) {
		return
	}
	q := r.URL.Query()
	service, person, from, to := q.Get("service"), q.Get("person"), q.Get("from"), q.Get("to")
	if person == "" {
		person = "any"
	}
	if !reInt.MatchString(service) || (person != "any" && !reInt.MatchString(person)) ||
		!reDate.MatchString(from) || !reDate.MatchString(to) {
		writeJSON(w, http.StatusBadRequest, &formError{Code: "invalid"})
		return
	}
	// whole days of the booking timezone: EGroupware reads a date without time in it
	path := "/slots?" + url.Values{"serviceId": {service}, "personId": {person}, "from": {from}, "to": {to}}.Encode()
	status, body, err := s.booking.cachedGet(path, bookingSlotsTTL)
	s.relay(w, status, body, err)
}

func (s *server) handleBookingHold(w http.ResponseWriter, r *http.Request) {
	if s.bookingDisabled(w) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseMultipartForm(64 << 10); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		s.fail(w, r, &formError{Status: http.StatusRequestEntityTooLarge, Code: "too_large"})
		return
	}
	if done, ferr := s.common(w, r); done || ferr != nil {
		if ferr != nil {
			s.fail(w, r, ferr)
		}
		return
	}
	lang := bookLang(r.FormValue("lang"))
	service := clean(r.FormValue("service"), 10)
	person := clean(r.FormValue("person"), 10)
	start := clean(r.FormValue("start"), 40)
	name := clean(r.FormValue("name"), 200)
	email := clean(r.FormValue("email"), 200)
	phone := clean(r.FormValue("phone"), 50)
	company := clean(r.FormValue("company"), 200)
	text := cleanText(r.FormValue("message"), 4000)
	confirmPage := localPath(r.FormValue("confirm_page"), "")

	var bad []string
	if !reInt.MatchString(service) {
		bad = append(bad, "service")
	}
	if person == "" {
		person = "any"
	}
	if person != "any" && !reInt.MatchString(person) {
		bad = append(bad, "person")
	}
	if !reTimeISO.MatchString(start) {
		bad = append(bad, "start")
	}
	if name == "" {
		bad = append(bad, "name")
	}
	if !validEmail(email) {
		bad = append(bad, "email")
	}
	if confirmPage == "" {
		bad = append(bad, "confirm_page")
	}
	if len(bad) > 0 {
		s.fail(w, r, invalid(bad...))
		return
	}
	serviceID, _ := strconv.Atoi(service)
	payload := map[string]any{
		"serviceId": serviceID, "personId": person, "start": start, "name": name, "email": email,
		"phone": phone, "company": company, "message": text, "lang": lang, "ip": clientIP(r),
	}
	status, body, err := s.booking.call("POST", "/holds", payload)
	if err != nil || status != http.StatusCreated {
		if err != nil {
			log.Printf("booking: hold: %v", err)
			s.fail(w, r, &formError{Status: http.StatusBadGateway, Code: "server"})
			return
		}
		s.fail(w, r, s.refusal(status, body))
		return
	}
	s.booking.forgetSlots()
	var v bookingView
	if err := json.Unmarshal(body, &v); err != nil || v.ConfirmToken == "" {
		log.Printf("booking: hold answer not understood: %v", err)
		s.fail(w, r, &formError{Status: http.StatusBadGateway, Code: "server"})
		return
	}
	link := s.cfg.SiteURL + confirmPage + "#t=" + v.ConfirmToken
	msg := message{
		From:    s.cfg.MailFrom,
		To:      (&mail.Address{Name: name, Address: email}).String(),
		ReplyTo: s.cfg.MailToContact,
		Subject: "[" + s.cfg.SiteName + "] " + bt(lang, "confirm_subject"),
		Body:    confirmMail(lang, s.cfg.SiteName, v, link),
	}
	if err := s.mailer.Send(msg); err != nil {
		// the hold expires by itself; the visitor gets an error and can try again
		log.Printf("booking: sending the confirm mail failed: %v", err)
		s.fail(w, r, &formError{Status: http.StatusBadGateway, Code: "server"})
		return
	}
	log.Printf("booking #%d held from %s", v.ID, clientIP(r))
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "booking": v.public()})
		return
	}
	http.Redirect(w, r, localPath(r.FormValue("next"), "/"), http.StatusSeeOther)
}

// tokenRequest reads token (and the optional fields) of a confirm / cancel POST: JSON or form
func tokenRequest(w http.ResponseWriter, r *http.Request) (map[string]string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	out := map[string]string{}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var in map[string]any
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			return nil, err
		}
		for k, v := range in {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
		return out, nil
	}
	if err := r.ParseMultipartForm(16 << 10); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		return nil, err
	}
	for k := range r.Form {
		out[k] = r.FormValue(k)
	}
	return out, nil
}

func (s *server) handleBookingConfirm(w http.ResponseWriter, r *http.Request) {
	if s.bookingDisabled(w) {
		return
	}
	if !s.tokenLimiter.allow(clientIP(r), s.now()) {
		writeJSON(w, http.StatusTooManyRequests, &formError{Code: "rate_limited"})
		return
	}
	in, err := tokenRequest(w, r)
	if err != nil || !reBkToken.MatchString(in["token"]) {
		writeJSON(w, http.StatusNotFound, &formError{Code: "bad_token"})
		return
	}
	status, body, err := s.booking.call("POST", "/confirm", map[string]string{"token": in["token"]})
	if err != nil || status != http.StatusOK {
		s.relay(w, status, body, err)
		return
	}
	s.booking.forgetSlots()
	var v bookingView
	if err := json.Unmarshal(body, &v); err != nil {
		s.relay(w, http.StatusBadGateway, nil, err)
		return
	}
	// a repeated confirm answers the same booking: mail only the first time
	if !v.AlreadyConfirmed {
		lang := bookLang(v.Lang)
		cancelPage := localPath(in["cancel_page"], "")
		link := ""
		if cancelPage != "" && v.CancelToken != "" {
			link = s.cfg.SiteURL + cancelPage + "#t=" + v.CancelToken
		}
		msg := message{
			From:    s.cfg.MailFrom,
			To:      (&mail.Address{Name: v.Client.Name, Address: v.Client.Email}).String(),
			ReplyTo: s.cfg.MailToContact,
			Subject: "[" + s.cfg.SiteName + "] " + bt(lang, "booked_subject"),
			Body:    bookedMail(lang, s.cfg.SiteName, v, link),
		}
		if err := s.mailer.Send(msg); err != nil {
			log.Printf("booking #%d: sending the booked mail failed: %v", v.ID, err) // the booking stands anyway
		}
		if s.cfg.MailToBookings != "" {
			office := message{
				From: s.cfg.MailFrom, To: s.cfg.MailToBookings,
				ReplyTo: (&mail.Address{Name: v.Client.Name, Address: v.Client.Email}).String(),
				Subject: fmt.Sprintf("[%s] Booking: %s – %s", s.cfg.SiteName, serviceTitle(v), v.Client.Name),
				Body:    officeMail(s.cfg.SiteName, v),
			}
			if err := s.mailer.Send(office); err != nil {
				log.Printf("booking #%d: office mail failed: %v", v.ID, err)
			}
		}
		log.Printf("booking #%d confirmed from %s", v.ID, clientIP(r))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "booking": v.public()})
}

func (s *server) handleBookingCancel(w http.ResponseWriter, r *http.Request) {
	if s.bookingDisabled(w) {
		return
	}
	if !s.tokenLimiter.allow(clientIP(r), s.now()) {
		writeJSON(w, http.StatusTooManyRequests, &formError{Code: "rate_limited"})
		return
	}
	in, err := tokenRequest(w, r)
	if err != nil || !reBkToken.MatchString(in["token"]) {
		writeJSON(w, http.StatusNotFound, &formError{Code: "bad_token"})
		return
	}
	status, body, err := s.booking.call("POST", "/cancel", map[string]string{"token": in["token"], "reason": clean(in["reason"], 200)})
	if err != nil || status != http.StatusOK {
		s.relay(w, status, body, err)
		return
	}
	s.booking.forgetSlots()
	var v bookingView
	json.Unmarshal(body, &v)
	log.Printf("booking #%d cancelled from %s", v.ID, clientIP(r))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "booking": v.public()})
}

func (s *server) handleBookingLookup(w http.ResponseWriter, r *http.Request) {
	if s.bookingDisabled(w) {
		return
	}
	if !s.tokenLimiter.allow(clientIP(r), s.now()) {
		writeJSON(w, http.StatusTooManyRequests, &formError{Code: "rate_limited"})
		return
	}
	in, err := tokenRequest(w, r)
	if err != nil || !reBkToken.MatchString(in["token"]) {
		writeJSON(w, http.StatusNotFound, &formError{Code: "bad_token"})
		return
	}
	status, body, err := s.booking.call("POST", "/lookup", map[string]string{"token": in["token"]})
	if err != nil || status != http.StatusOK {
		s.relay(w, status, body, err)
		return
	}
	var v bookingView
	json.Unmarshal(body, &v)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "booking": v.public()})
}

// --- mails -----------------------------------------------------------------

var bookingTexts = map[string]map[string]string{
	"en": {
		"confirm_subject": "Please confirm your appointment",
		"confirm_intro":   "Thank you for your booking request. Please confirm the appointment with the link below.",
		"confirm_expires": "The time is reserved for you until %s. Without confirmation it is released again.",
		"confirm_ignore":  "If you did not request this appointment, simply ignore this e-mail.",
		"booked_subject":  "Your appointment is confirmed",
		"booked_intro":    "Your appointment is confirmed. You will also receive a calendar invitation.",
		"cancel_link":     "If you cannot attend, you can cancel the appointment online until %s:",
		"cancel_call":     "If you cannot attend, please let us know.",
		"service":         "Service",
		"person":          "With",
		"when":            "When",
		"location":        "Location",
		"video":           "Video call, the link follows with the calendar invitation",
		"regards":         "Kind regards",
	},
	"de": {
		"confirm_subject": "Bitte bestätigen Sie Ihren Termin",
		"confirm_intro":   "Vielen Dank für Ihre Terminanfrage. Bitte bestätigen Sie den Termin über den folgenden Link.",
		"confirm_expires": "Der Termin ist bis %s für Sie reserviert. Ohne Bestätigung wird er wieder freigegeben.",
		"confirm_ignore":  "Falls Sie diesen Termin nicht angefragt haben, ignorieren Sie diese E-Mail bitte.",
		"booked_subject":  "Ihr Termin ist bestätigt",
		"booked_intro":    "Ihr Termin ist bestätigt. Sie erhalten zusätzlich eine Kalendereinladung.",
		"cancel_link":     "Falls Sie den Termin nicht wahrnehmen können, können Sie ihn bis %s online absagen:",
		"cancel_call":     "Falls Sie den Termin nicht wahrnehmen können, geben Sie uns bitte Bescheid.",
		"service":         "Leistung",
		"person":          "Mit",
		"when":            "Wann",
		"location":        "Ort",
		"video":           "Videogespräch, der Link folgt mit der Kalendereinladung",
		"regards":         "Mit freundlichen Grüßen",
	},
	"el": {
		"confirm_subject": "Επιβεβαιώστε το ραντεβού σας",
		"confirm_intro":   "Σας ευχαριστούμε για το αίτημα ραντεβού. Παρακαλούμε επιβεβαιώστε το ραντεβού με τον παρακάτω σύνδεσμο.",
		"confirm_expires": "Η ώρα είναι δεσμευμένη για εσάς έως %s. Χωρίς επιβεβαίωση αποδεσμεύεται.",
		"confirm_ignore":  "Αν δεν ζητήσατε εσείς αυτό το ραντεβού, αγνοήστε αυτό το μήνυμα.",
		"booked_subject":  "Το ραντεβού σας επιβεβαιώθηκε",
		"booked_intro":    "Το ραντεβού σας επιβεβαιώθηκε. Θα λάβετε επίσης πρόσκληση ημερολογίου.",
		"cancel_link":     "Αν δεν μπορείτε να έρθετε, μπορείτε να ακυρώσετε το ραντεβού online έως %s:",
		"cancel_call":     "Αν δεν μπορείτε να έρθετε, παρακαλούμε ενημερώστε μας.",
		"service":         "Υπηρεσία",
		"person":          "Με",
		"when":            "Πότε",
		"location":        "Τόπος",
		"video":           "Βιντεοκλήση, ο σύνδεσμος ακολουθεί με την πρόσκληση ημερολογίου",
		"regards":         "Με εκτίμηση",
	},
}

func bt(lang, key string) string {
	if t, ok := bookingTexts[lang][key]; ok {
		return t
	}
	return bookingTexts["en"][key]
}

var (
	weekdays = map[string][]string{
		"en": {"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
		"de": {"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"},
		"el": {"Κυριακή", "Δευτέρα", "Τρίτη", "Τετάρτη", "Πέμπτη", "Παρασκευή", "Σάββατο"},
	}
	months = map[string][]string{
		"en": {"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
		"de": {"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"},
		// genitive, as in a date
		"el": {"Ιανουαρίου", "Φεβρουαρίου", "Μαρτίου", "Απριλίου", "Μαΐου", "Ιουνίου", "Ιουλίου", "Αυγούστου", "Σεπτεμβρίου", "Οκτωβρίου", "Νοεμβρίου", "Δεκεμβρίου"},
	}
)

// longDate formats an ISO 8601 time in its own offset (the booking timezone), e.g.
// "Montag, 19. Oktober 2026, 10:00 Uhr"
func longDate(lang, iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	wd, mo := weekdays[lang], months[lang]
	if wd == nil {
		lang, wd, mo = "en", weekdays["en"], months["en"]
	}
	switch lang {
	case "de":
		return fmt.Sprintf("%s, %d. %s %d, %s Uhr", wd[t.Weekday()], t.Day(), mo[t.Month()-1], t.Year(), t.Format("15:04"))
	case "el":
		return fmt.Sprintf("%s %d %s %d, %s", wd[t.Weekday()], t.Day(), mo[t.Month()-1], t.Year(), t.Format("15:04"))
	}
	return fmt.Sprintf("%s, %d %s %d, %s", wd[t.Weekday()], t.Day(), mo[t.Month()-1], t.Year(), t.Format("15:04"))
}

func serviceTitle(v bookingView) string {
	if v.Service != nil {
		return v.Service.Title
	}
	return ""
}

func bookingDetails(lang string, v bookingView) string {
	var b strings.Builder
	row := func(label, value string) {
		if value != "" {
			fmt.Fprintf(&b, "%-10s %s\n", label+":", value)
		}
	}
	row(bt(lang, "service"), serviceTitle(v))
	if v.Person != nil {
		row(bt(lang, "person"), v.Person.Name)
	}
	row(bt(lang, "when"), longDate(lang, v.Start))
	if v.Service != nil && v.Service.Video {
		row(bt(lang, "location"), bt(lang, "video"))
	} else if v.Service != nil && v.Service.Location != nil {
		row(bt(lang, "location"), *v.Service.Location)
	}
	return b.String()
}

func greeting(lang, name string) string {
	switch lang {
	case "de":
		return "Guten Tag " + name + ","
	case "el":
		return "Αγαπητέ/ή " + name + ","
	}
	return "Dear " + name + ","
}

func confirmMail(lang, site string, v bookingView, link string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n%s\n\n%s\n%s\n\n", greeting(lang, v.Client.Name), bt(lang, "confirm_intro"), bookingDetails(lang, v), link)
	if v.Expires != "" {
		t, err := time.Parse(time.RFC3339, v.Expires)
		if err == nil {
			fmt.Fprintf(&b, bt(lang, "confirm_expires")+"\n", t.Format("15:04"))
		}
	}
	fmt.Fprintf(&b, "%s\n\n%s\n%s\n", bt(lang, "confirm_ignore"), bt(lang, "regards"), site)
	return b.String()
}

func bookedMail(lang, site string, v bookingView, cancelLink string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n%s\n\n%s\n", greeting(lang, v.Client.Name), bt(lang, "booked_intro"), bookingDetails(lang, v))
	if cancelLink != "" && v.CancelUntil != "" {
		fmt.Fprintf(&b, bt(lang, "cancel_link")+"\n%s\n\n", longDate(lang, v.CancelUntil), cancelLink)
	} else {
		fmt.Fprintf(&b, "%s\n\n", bt(lang, "cancel_call"))
	}
	fmt.Fprintf(&b, "%s\n%s\n", bt(lang, "regards"), site)
	return b.String()
}

func officeMail(site string, v bookingView) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Appointment booked and confirmed on %s\n\n%s", site, bookingDetails("en", v))
	fmt.Fprintf(&b, "%-10s %s <%s>\nLanguage:  %s\nBooking:   #%d\n\nThe event is in the calendar of the booked person.\n",
		"Client:", v.Client.Name, v.Client.Email, v.Lang, v.ID)
	return b.String()
}
