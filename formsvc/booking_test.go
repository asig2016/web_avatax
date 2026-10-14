package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

const testBookingToken = "12.abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"

// fakeEGW plays EGroupware's acbooking API
type fakeEGW struct {
	mu        sync.Mutex
	calls     []string // "METHOD path?query"
	bodies    []map[string]any
	status    int    // answer status, 0 = default per route
	errBody   string // body for status >= 400
	confirmed bool   // what the next confirm answers as alreadyConfirmed
}

func (f *fakeEGW) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "booking-api" || pass != "tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept %q", r.Header.Get("Accept"))
		}
		f.mu.Lock()
		f.calls = append(f.calls, r.Method+" "+strings.TrimPrefix(r.URL.Path, "/egw/acbooking")+queryOf(r))
		var body map[string]any
		if r.Method == "POST" {
			json.NewDecoder(r.Body).Decode(&body)
		}
		f.bodies = append(f.bodies, body)
		status, errBody, confirmed := f.status, f.errBody, f.confirmed
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if status >= 400 {
			w.WriteHeader(status)
			io.WriteString(w, errBody)
			return
		}
		view := `"id": 12, "service": {"title": "Erstberatung", "durationMinutes": 30, "price": null, "currency": "EUR", "video": false, "location": "Syngrou 76"},
			"person": {"name": "Maria P."}, "start": "2026-10-19T10:00:00+03:00", "end": "2026-10-19T10:30:00+03:00", "timeZone": "Europe/Athens",
			"client": {"name": "Erika Mustermann", "email": "erika@example.com"}, "lang": "de"`
		switch {
		case strings.HasSuffix(r.URL.Path, "/holds"):
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, `{`+view+`, "status": "held", "expires": "2026-10-14T17:20:00+03:00", "confirmToken": "`+testBookingToken+`"}`)
		case strings.HasSuffix(r.URL.Path, "/confirm"):
			c := "false"
			if confirmed {
				c = "true"
			}
			io.WriteString(w, `{`+view+`, "status": "confirmed", "cancelUntil": "2026-10-18T10:00:00+03:00", "cancelToken": "13.cancelcancelcancelcancelcancelcancelcance", "alreadyConfirmed": `+c+`}`)
		case strings.HasSuffix(r.URL.Path, "/cancel"):
			io.WriteString(w, `{`+view+`, "status": "cancelled"}`)
		case strings.HasSuffix(r.URL.Path, "/lookup"):
			io.WriteString(w, `{`+view+`, "status": "confirmed"}`)
		default:
			io.WriteString(w, `{"ok": "`+r.URL.Path+`"}`)
		}
	}
}

func queryOf(r *http.Request) string {
	if r.URL.RawQuery == "" {
		return ""
	}
	return "?" + r.URL.RawQuery
}

func bookingServer(t *testing.T) (*server, *fakeMailer, *fakeEGW) {
	s, fm := testServer()
	f := &fakeEGW{}
	ts := httptest.NewServer(f.handler(t))
	t.Cleanup(ts.Close)
	s.cfg.SiteURL = "https://avatax.eu"
	s.cfg.MailToBookings = "termine@avatax.eu"
	s.booking = newBooking(ts.URL+"/egw/acbooking", "booking-api", "tok")
	s.booking.now = s.now
	return s, fm, f
}

func get(s *server, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("X-Real-IP", "203.0.113.5")
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, req)
	return w
}

func postJSON(s *server, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Real-IP", "203.0.113.5")
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, req)
	return w
}

func holdForm(s *server) url.Values {
	return url.Values{
		"lang": {"de"}, "service": {"3"}, "person": {"any"}, "start": {"2026-10-19T10:00:00+03:00"},
		"name": {"Erika Mustermann"}, "email": {"erika@example.com"}, "phone": {"+49 30 1234"}, "message": {"Frage zur Steuer"},
		"consent": {"1"}, "token": {tokenAt(s, 10*time.Second)}, "confirm_page": {"/de/termin/bestaetigen/"},
	}
}

func TestBookingDisabled(t *testing.T) {
	s, _ := testServer()
	if w := get(s, "/api/booking/catalog?lang=de"); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d", w.Code)
	}
}

func TestBookingCatalogIsProxiedAndCached(t *testing.T) {
	s, _, f := bookingServer(t)
	for i := 0; i < 3; i++ {
		w := get(s, "/api/booking/catalog?lang=el")
		if w.Code != 200 || !strings.Contains(w.Body.String(), "/catalog") {
			t.Fatalf("status %d: %s", w.Code, w.Body)
		}
	}
	if len(f.calls) != 1 || f.calls[0] != "GET /catalog?lang=el" {
		t.Fatalf("calls %v", f.calls)
	}
	// an unknown language becomes the default
	get(s, "/api/booking/catalog?lang=xx")
	if f.calls[1] != "GET /catalog?lang=en" {
		t.Fatalf("calls %v", f.calls)
	}
	// expired cache asks again
	s.booking.now = func() time.Time { return t0.Add(bookingCatalogTTL + time.Second) }
	get(s, "/api/booking/catalog?lang=el")
	if len(f.calls) != 3 {
		t.Fatalf("calls %v", f.calls)
	}
}

func TestBookingSlotsValidatesTheQuery(t *testing.T) {
	s, _, f := bookingServer(t)
	for _, q := range []string{"service=x&from=2026-10-19&to=2026-10-26", "service=3&person=abc&from=2026-10-19&to=2026-10-26",
		"service=3&from=19.10.2026&to=2026-10-26", "service=3&from=2026-10-19"} {
		if w := get(s, "/api/booking/slots?"+q); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d", q, w.Code)
		}
	}
	if len(f.calls) != 0 {
		t.Fatalf("bad queries reached EGroupware: %v", f.calls)
	}
	if w := get(s, "/api/booking/slots?service=3&from=2026-10-19&to=2026-10-26"); w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	if f.calls[0] != "GET /slots?from=2026-10-19&personId=any&serviceId=3&to=2026-10-26" {
		t.Fatalf("calls %v", f.calls)
	}
}

func TestBookingHoldMailsTheConfirmLink(t *testing.T) {
	s, fm, f := bookingServer(t)
	get(s, "/api/booking/slots?service=3&from=2026-10-19&to=2026-10-26") // cached ...
	w := post(s, "/api/booking/hold", holdForm(s), true)
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), testBookingToken) {
		t.Fatal("the confirm token must only travel by mail")
	}
	if got := f.calls[1]; got != "POST /holds" {
		t.Fatalf("calls %v", f.calls)
	}
	if b := f.bodies[1]; b["ip"] != "203.0.113.5" || b["serviceId"] != float64(3) || b["personId"] != "any" || b["lang"] != "de" {
		t.Fatalf("payload %v", b)
	}
	if len(fm.sent) != 1 {
		t.Fatalf("mails %d", len(fm.sent))
	}
	m := fm.sent[0]
	if !strings.Contains(m.To, "erika@example.com") || !strings.Contains(m.Subject, "Bitte bestätigen") {
		t.Fatalf("mail %q to %q", m.Subject, m.To)
	}
	if !strings.Contains(m.Body, "https://avatax.eu/de/termin/bestaetigen/#t="+testBookingToken) {
		t.Fatalf("no confirm link in\n%s", m.Body)
	}
	if !strings.Contains(m.Body, "Montag, 19. Oktober 2026, 10:00 Uhr") || !strings.Contains(m.Body, "bis 17:20") {
		t.Fatalf("date / expiry not localized in\n%s", m.Body)
	}
	// ... and the hold dropped the cached slots
	get(s, "/api/booking/slots?service=3&from=2026-10-19&to=2026-10-26")
	if len(f.calls) != 3 {
		t.Fatalf("slots not asked again after a hold: %v", f.calls)
	}
}

func TestBookingHoldNeedsTheAntiSpamChecks(t *testing.T) {
	s, fm, f := bookingServer(t)
	form := holdForm(s)
	form.Set("token", "garbage")
	if w := post(s, "/api/booking/hold", form, true); w.Code != http.StatusBadRequest {
		t.Fatalf("status %d", w.Code)
	}
	form = holdForm(s)
	form.Set("website", "http://spam")
	post(s, "/api/booking/hold", form, true)
	form = holdForm(s)
	form.Del("consent")
	if w := post(s, "/api/booking/hold", form, true); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", w.Code)
	}
	form = holdForm(s)
	form.Set("confirm_page", "https://evil.example/")
	if w := post(s, "/api/booking/hold", form, true); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("open redirect target accepted: %d", w.Code)
	}
	if len(f.calls) != 0 || len(fm.sent) != 0 {
		t.Fatalf("calls %v, mails %d", f.calls, len(fm.sent))
	}
}

func TestBookingHoldRefusalIsPassedOn(t *testing.T) {
	s, fm, f := bookingServer(t)
	f.status, f.errBody = http.StatusConflict, `{"error": 409, "code": "slot_taken", "message": "The slot is no longer available"}`
	w := post(s, "/api/booking/hold", holdForm(s), true)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"slot_taken"`) {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	if len(fm.sent) != 0 {
		t.Fatal("mail sent for a refused hold")
	}
	// EGroupware's own credentials problem is ours, not the visitor's
	f.status, f.errBody = http.StatusUnauthorized, ""
	if w := post(s, "/api/booking/hold", holdForm(s), true); w.Code != http.StatusBadGateway {
		t.Fatalf("status %d", w.Code)
	}
}

func TestBookingConfirmMailsOnce(t *testing.T) {
	s, fm, f := bookingServer(t)
	body := `{"token": "` + testBookingToken + `", "cancel_page": "/de/termin/absagen/"}`
	w := postJSON(s, "/api/booking/confirm", body)
	if w.Code != 200 || strings.Contains(w.Body.String(), "cancelcancel") {
		t.Fatalf("status %d (no tokens in the answer): %s", w.Code, w.Body)
	}
	if len(fm.sent) != 2 {
		t.Fatalf("expected client + office mail, got %d", len(fm.sent))
	}
	client, office := fm.sent[0], fm.sent[1]
	if !strings.Contains(client.Body, "https://avatax.eu/de/termin/absagen/#t=13.cancel") || !strings.Contains(client.Body, "Sonntag, 18. Oktober 2026, 10:00 Uhr") {
		t.Fatalf("client mail\n%s", client.Body)
	}
	if office.To != "termine@avatax.eu" || !strings.Contains(office.Body, "erika@example.com") {
		t.Fatalf("office mail to %q\n%s", office.To, office.Body)
	}
	f.confirmed = true
	postJSON(s, "/api/booking/confirm", body)
	if len(fm.sent) != 2 {
		t.Fatal("a repeated confirm mailed again")
	}
}

func TestBookingTokensAreCheckedBeforeEGroupware(t *testing.T) {
	s, _, f := bookingServer(t)
	for _, body := range []string{`{"token": "1.short"}`, `{"token": "../../x"}`, `{}`, `nonsense`} {
		if w := postJSON(s, "/api/booking/confirm", body); w.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", body, w.Code)
		}
		if w := postJSON(s, "/api/booking/cancel", body); w.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", body, w.Code)
		}
	}
	if w := postJSON(s, "/api/booking/booking", `{"token": "../catalog"}`); w.Code != http.StatusNotFound {
		t.Errorf("status %d", w.Code)
	}
	if len(f.calls) != 0 {
		t.Fatalf("calls %v", f.calls)
	}
}

func TestBookingCancelAndRead(t *testing.T) {
	s, _, f := bookingServer(t)
	if w := postJSON(s, "/api/booking/booking", `{"token": "`+testBookingToken+`"}`); w.Code != 200 || !strings.Contains(w.Body.String(), "Maria P.") {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	w := postJSON(s, "/api/booking/cancel", `{"token": "`+testBookingToken+`", "reason": "krank"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"cancelled"`) {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	if b := f.bodies[1]; b["reason"] != "krank" || b["token"] != testBookingToken {
		t.Fatalf("payload %v", b)
	}
	f.status, f.errBody = http.StatusConflict, `{"error": 409, "code": "too_late"}`
	if w := postJSON(s, "/api/booking/cancel", `{"token": "`+testBookingToken+`"}`); w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "too_late") {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
}

func TestBookingTokenCallsHaveTheirOwnLimit(t *testing.T) {
	s, _, _ := bookingServer(t)
	s.limiter = newRateLimiter(1, time.Minute)
	s.tokenLimiter = newRateLimiter(3, time.Minute)
	body := `{"token": "` + testBookingToken + `"}`
	for i := 0; i < 3; i++ {
		if w := postJSON(s, "/api/booking/booking", body); w.Code != 200 {
			t.Fatalf("call %d: status %d", i, w.Code)
		}
	}
	if w := postJSON(s, "/api/booking/booking", body); w.Code != http.StatusTooManyRequests {
		t.Fatalf("status %d", w.Code)
	}
	// the form limit is untouched by them
	if w := post(s, "/api/booking/hold", holdForm(s), true); w.Code != 200 {
		t.Fatalf("hold status %d: %s", w.Code, w.Body)
	}
}

func TestLongDate(t *testing.T) {
	for lang, want := range map[string]string{
		"en": "Monday, 19 October 2026, 10:00",
		"de": "Montag, 19. Oktober 2026, 10:00 Uhr",
		"el": "Δευτέρα 19 Οκτωβρίου 2026, 10:00",
	} {
		if got := longDate(lang, "2026-10-19T10:00:00+03:00"); got != want {
			t.Errorf("%s: %q", lang, got)
		}
	}
}
