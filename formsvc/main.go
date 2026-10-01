// formsvc receives the contact and CV forms of the static avatax.eu site
// and forwards them by e-mail. It keeps no state on disk.
//
// Endpoints (all behind the web container's nginx at /api/):
//
//	GET  /api/token    signed timestamp for the anti-spam check (fetched by the form JS)
//	POST /api/contact  contact form
//	POST /api/cv       job application incl. CV upload
//	GET  /healthz      liveness probe
package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // distroless image has no zoneinfo; TZ comes from compose
	"unicode/utf8"
)

type config struct {
	SiteName       string // shown in mail subjects and bodies, e.g. "avatax.eu"
	Listen         string
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPass       string
	SMTPTLS        string // starttls | tls | none
	MailFrom       string
	MailToContact  string
	MailToCV       string
	TokenSecret    []byte
	MinFill        time.Duration
	MaxTokenAge    time.Duration
	RateLimit      int
	RateWindow     time.Duration
	MaxUploadBytes int64
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(env(key, "")); err == nil {
		return v
	}
	return def
}

func loadConfig() (config, error) {
	c := config{
		SiteName:       env("SITE_NAME", "website"),
		Listen:         env("LISTEN", ":8081"),
		SMTPHost:       env("SMTP_HOST", ""),
		SMTPPort:       envInt("SMTP_PORT", 587),
		SMTPUser:       env("SMTP_USER", ""),
		SMTPPass:       os.Getenv("SMTP_PASS"),
		SMTPTLS:        strings.ToLower(env("SMTP_TLS", "starttls")),
		MailFrom:       env("MAIL_FROM", ""),
		MailToContact:  env("MAIL_TO_CONTACT", ""),
		MinFill:        time.Duration(envInt("MIN_FILL_SECONDS", 3)) * time.Second,
		MaxTokenAge:    time.Duration(envInt("MAX_TOKEN_AGE_MINUTES", 180)) * time.Minute,
		RateLimit:      envInt("RATE_LIMIT", 5),
		RateWindow:     time.Duration(envInt("RATE_WINDOW_MINUTES", 10)) * time.Minute,
		MaxUploadBytes: int64(envInt("MAX_UPLOAD_MB", 5)) << 20,
	}
	c.MailToCV = env("MAIL_TO_CV", c.MailToContact)
	if s := os.Getenv("TOKEN_SECRET"); s != "" {
		c.TokenSecret = []byte(s)
	} else {
		// Tokens then only survive until the next restart, which is acceptable.
		c.TokenSecret = make([]byte, 32)
		if _, err := rand.Read(c.TokenSecret); err != nil {
			return c, err
		}
	}
	var missing []string
	for k, v := range map[string]string{"SMTP_HOST": c.SMTPHost, "MAIL_FROM": c.MailFrom, "MAIL_TO_CONTACT": c.MailToContact} {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return c, fmt.Errorf("missing required settings: %s", strings.Join(missing, ", "))
	}
	switch c.SMTPTLS {
	case "starttls", "tls", "none":
	default:
		return c, fmt.Errorf("SMTP_TLS must be starttls, tls or none, got %q", c.SMTPTLS)
	}
	return c, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	srv := newServer(cfg, &smtpMailer{cfg: cfg})
	log.Printf("formsvc listening on %s (smtp %s:%d, tls=%s)", cfg.Listen, cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPTLS)
	hs := &http.Server{
		Addr:              cfg.Listen,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	log.Fatal(hs.ListenAndServe())
}

// ---------------------------------------------------------------------------

type server struct {
	cfg     config
	mailer  mailer
	limiter *rateLimiter
	now     func() time.Time
}

func newServer(cfg config, m mailer) *server {
	return &server{cfg: cfg, mailer: m, limiter: newRateLimiter(cfg.RateLimit, cfg.RateWindow), now: time.Now}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok\n") })
	mux.HandleFunc("GET /api/token", s.handleToken)
	mux.HandleFunc("POST /api/contact", s.handleContact)
	mux.HandleFunc("POST /api/cv", s.handleCV)
	return mux
}

// --- anti-spam token -------------------------------------------------------

func (s *server) sign(ts string) string {
	m := hmac.New(sha256.New, s.cfg.TokenSecret)
	m.Write([]byte(ts))
	return hex.EncodeToString(m.Sum(nil))
}

func (s *server) newToken() string {
	ts := strconv.FormatInt(s.now().Unix(), 10)
	return ts + "." + s.sign(ts)
}

var errTooFast = errors.New("form submitted too fast")

func (s *server) checkToken(tok string) error {
	ts, sig, ok := strings.Cut(tok, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.sign(ts))) {
		return errors.New("invalid token")
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return errors.New("invalid token")
	}
	age := s.now().Sub(time.Unix(sec, 0))
	if age < s.cfg.MinFill {
		return errTooFast
	}
	if age > s.cfg.MaxTokenAge {
		return errors.New("token expired")
	}
	return nil
}

func (s *server) handleToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"token": s.newToken()})
}

// --- form handling ---------------------------------------------------------

// formError is reported back to the page; Code is mapped to a localized text by the form JS.
type formError struct {
	Status int      `json:"-"`
	Code   string   `json:"error"`
	Fields []string `json:"fields,omitempty"`
}

func (e *formError) Error() string { return e.Code }

func invalid(fields ...string) *formError {
	return &formError{Status: http.StatusUnprocessableEntity, Code: "invalid", Fields: fields}
}

// common runs the checks shared by both forms. It returns done=true when the
// request has already been answered (honeypot hit).
func (s *server) common(w http.ResponseWriter, r *http.Request) (done bool, ferr *formError) {
	if !s.limiter.allow(clientIP(r), s.now()) {
		return false, &formError{Status: http.StatusTooManyRequests, Code: "rate_limited"}
	}
	if r.FormValue("website") != "" { // honeypot: bots fill every field; pretend success
		log.Printf("honeypot hit from %s", clientIP(r))
		s.respondOK(w, r)
		return true, nil
	}
	if err := s.checkToken(r.FormValue("token")); err != nil {
		log.Printf("token rejected from %s: %v", clientIP(r), err)
		return false, &formError{Status: http.StatusBadRequest, Code: "spam"}
	}
	if r.FormValue("consent") == "" {
		return false, invalid("consent")
	}
	return false, nil
}

type field struct{ label, value string }

func (s *server) handleContact(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	// The form JS posts multipart (FormData), plain HTML posts are urlencoded.
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
	name := clean(r.FormValue("name"), 200)
	email := clean(r.FormValue("email"), 200)
	phone := clean(r.FormValue("phone"), 50)
	subject := clean(r.FormValue("subject"), 200)
	text := cleanText(r.FormValue("message"), 10000)

	var bad []string
	if name == "" {
		bad = append(bad, "name")
	}
	if !validEmail(email) {
		bad = append(bad, "email")
	}
	if phone == "" {
		bad = append(bad, "phone")
	}
	if subject == "" {
		bad = append(bad, "subject")
	}
	if text == "" {
		bad = append(bad, "message")
	}
	if len(bad) > 0 {
		s.fail(w, r, invalid(bad...))
		return
	}

	msg := message{
		From:    s.cfg.MailFrom,
		To:      s.cfg.MailToContact,
		ReplyTo: (&mail.Address{Name: name, Address: email}).String(),
		Subject: "[" + s.cfg.SiteName + "] " + subject,
		Body: body(s.cfg.SiteName, "Contact form", r, s.now(), []field{
			{"Name", name}, {"E-mail", email}, {"Phone", phone}, {"Subject", subject},
		}, text),
	}
	s.send(w, r, msg)
}

var cvTypes = map[string]string{
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".odt":  "application/vnd.oasis.opendocument.text",
}

// sniffCV checks the file's magic bytes match its extension.
func sniffCV(ext string, head []byte) bool {
	switch ext {
	case ".pdf":
		return strings.HasPrefix(string(head), "%PDF-")
	case ".docx", ".odt":
		return strings.HasPrefix(string(head), "PK\x03\x04")
	case ".doc":
		return strings.HasPrefix(string(head), "\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1")
	}
	return false
}

func (s *server) handleCV(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes+(256<<10))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			s.fail(w, r, &formError{Status: http.StatusRequestEntityTooLarge, Code: "too_large", Fields: []string{"cv"}})
		} else {
			s.fail(w, r, invalid())
		}
		return
	}
	defer r.MultipartForm.RemoveAll()
	if done, ferr := s.common(w, r); done || ferr != nil {
		if ferr != nil {
			s.fail(w, r, ferr)
		}
		return
	}

	position := clean(r.FormValue("position"), 200)
	name := clean(r.FormValue("name"), 200)
	email := clean(r.FormValue("email"), 200)
	phone := clean(r.FormValue("phone"), 50)
	start := clean(r.FormValue("start_date"), 20)
	commute := clean(r.FormValue("commute"), 20)
	salary := clean(r.FormValue("salary"), 20)
	text := cleanText(r.FormValue("message"), 10000)

	var bad []string
	if position == "" {
		bad = append(bad, "position")
	}
	if name == "" {
		bad = append(bad, "name")
	}
	if !validEmail(email) {
		bad = append(bad, "email")
	}
	if start != "" {
		if _, err := time.Parse("2006-01-02", start); err != nil {
			bad = append(bad, "start_date")
		}
	}

	var att *attachment
	file, hdr, err := r.FormFile("cv")
	if err != nil {
		bad = append(bad, "cv")
	} else {
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(hdr.Filename))
		data, rerr := io.ReadAll(io.LimitReader(file, s.cfg.MaxUploadBytes+1))
		switch {
		case rerr != nil:
			bad = append(bad, "cv")
		case int64(len(data)) > s.cfg.MaxUploadBytes:
			s.fail(w, r, &formError{Status: http.StatusRequestEntityTooLarge, Code: "too_large", Fields: []string{"cv"}})
			return
		case cvTypes[ext] == "" || !sniffCV(ext, data):
			s.fail(w, r, &formError{Status: http.StatusUnprocessableEntity, Code: "bad_file", Fields: []string{"cv"}})
			return
		default:
			att = &attachment{Name: safeFilename(name, ext), ContentType: cvTypes[ext], Data: data}
		}
	}
	if len(bad) > 0 {
		s.fail(w, r, invalid(bad...))
		return
	}

	msg := message{
		From:        s.cfg.MailFrom,
		To:          s.cfg.MailToCV,
		ReplyTo:     (&mail.Address{Name: name, Address: email}).String(),
		Subject:     fmt.Sprintf("[%s] CV: %s – %s", s.cfg.SiteName, position, name),
		Attachments: []attachment{*att},
		Body: body(s.cfg.SiteName, "Job application", r, s.now(), []field{
			{"Position", position}, {"Name", name}, {"E-mail", email}, {"Phone", phone},
			{"Earliest start", start}, {"Commute (minutes)", commute}, {"Desired gross monthly salary (EUR)", salary},
		}, text),
	}
	s.send(w, r, msg)
}

func (s *server) send(w http.ResponseWriter, r *http.Request, m message) {
	if err := s.mailer.Send(m); err != nil {
		log.Printf("sending mail failed: %v", err)
		s.fail(w, r, &formError{Status: http.StatusBadGateway, Code: "server"})
		return
	}
	log.Printf("mail sent: %q from %s", m.Subject, clientIP(r))
	s.respondOK(w, r)
}

func body(site, title string, r *http.Request, now time.Time, fields []field, text string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s submitted on %s\n\n", title, site)
	for _, f := range fields {
		if f.value != "" {
			fmt.Fprintf(&b, "%-36s %s\n", f.label+":", f.value)
		}
	}
	if text != "" {
		fmt.Fprintf(&b, "\nMessage:\n%s\n", text)
	}
	fmt.Fprintf(&b, "\n--\nLanguage: %s | Page: %s\nTime: %s | IP: %s\n",
		clean(r.FormValue("lang"), 5), clean(r.Referer(), 300), now.Format(time.RFC1123Z), clientIP(r))
	return b.String()
}

// --- responses -------------------------------------------------------------

// wantsJSON is true for the fetch() submissions of the form JS; plain HTML
// form posts (no JavaScript) get redirects to the localized result pages.
func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func (s *server) respondOK(w http.ResponseWriter, r *http.Request) {
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	http.Redirect(w, r, localPath(r.FormValue("next"), "/"), http.StatusSeeOther)
}

func (s *server) fail(w http.ResponseWriter, r *http.Request, e *formError) {
	if wantsJSON(r) {
		writeJSON(w, e.Status, e)
		return
	}
	http.Redirect(w, r, localPath(r.FormValue("error_page"), "/"), http.StatusSeeOther)
}

// localPath only accepts same-site absolute paths, to avoid open redirects.
func localPath(p, def string) string {
	if strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "//") && !strings.HasPrefix(p, "/\\") && !strings.ContainsAny(p, "\r\n") {
		return p
	}
	return def
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// --- helpers ---------------------------------------------------------------

// clean trims, removes control characters (incl. CR/LF, so values are safe
// in mail headers) and truncates to max runes.
func clean(s string, max int) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
	return truncate(strings.TrimSpace(s), max)
}

// cleanText is clean for multi-line text: keeps newlines and tabs.
func cleanText(s string, max int) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.Map(func(r rune) rune {
		if (r < 0x20 && r != '\n' && r != '\t') || r == 0x7f {
			return -1
		}
		return r
	}, s)
	return truncate(strings.TrimSpace(s), max)
}

func truncate(s string, max int) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

func validEmail(s string) bool {
	if s == "" || strings.ContainsAny(s, " <>,;\"") {
		return false
	}
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s && strings.Contains(s[strings.LastIndex(s, "@"):], ".")
}

func safeFilename(name, ext string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('_')
		}
	}
	base := strings.Trim(b.String(), "_")
	if base == "" {
		base = "applicant"
	}
	return "CV_" + truncate(base, 60) + ext
}

// clientIP uses X-Real-IP, which the web container's nginx sets from the
// real client address. formsvc is never exposed directly.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(ip) != nil {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// --- rate limiting ---------------------------------------------------------

type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
	calls  int
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, hits: map[string][]time.Time{}}
}

func (l *rateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++
	if l.calls%1000 == 0 { // occasional cleanup of stale entries
		for k, v := range l.hits {
			if len(v) == 0 || now.Sub(v[len(v)-1]) > l.window {
				delete(l.hits, k)
			}
		}
	}
	recent := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if now.Sub(t) < l.window {
			recent = append(recent, t)
		}
	}
	if len(recent) >= l.limit {
		l.hits[key] = recent
		return false
	}
	l.hits[key] = append(recent, now)
	return true
}
