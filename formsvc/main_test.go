package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type fakeMailer struct {
	sent []message
	err  error
}

func (f *fakeMailer) Send(m message) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, m)
	return nil
}

var t0 = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func testServer() (*server, *fakeMailer) {
	cfg := config{
		SiteName: "avatax.eu", MailFrom: "Website <web@avatax.eu>", MailToContact: "info@avatax.eu", MailToCV: "cv@avatax.eu",
		TokenSecret: []byte("secret"), MinFill: 3 * time.Second, MaxTokenAge: time.Hour,
		RateLimit: 100, RateWindow: 10 * time.Minute, MaxUploadBytes: 1 << 20,
	}
	fm := &fakeMailer{}
	s := newServer(cfg, fm)
	s.now = func() time.Time { return t0 }
	return s, fm
}

// tokenAt returns a token created `ago` before t0.
func tokenAt(s *server, ago time.Duration) string {
	now := s.now
	s.now = func() time.Time { return t0.Add(-ago) }
	tok := s.newToken()
	s.now = now
	return tok
}

func contactForm(s *server) url.Values {
	return url.Values{
		"lang": {"el"}, "name": {"Γιώργος Π."}, "email": {"g@example.gr"}, "phone": {"2101234567"},
		"subject": {"Ερώτηση"}, "message": {"Γεια σας,\nμια ερώτηση."}, "consent": {"1"},
		"token": {tokenAt(s, 10*time.Second)}, "next": {"/el/efcharistoume/"}, "error_page": {"/el/sfalma/"},
	}
}

func post(s *server, path string, form url.Values, jsonAccept bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Real-IP", "203.0.113.5")
	if jsonAccept {
		req.Header.Set("Accept", "application/json")
	}
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, req)
	return w
}

func TestContactOK(t *testing.T) {
	s, fm := testServer()
	w := post(s, "/api/contact", contactForm(s), true)
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	if len(fm.sent) != 1 {
		t.Fatalf("expected 1 mail, got %d", len(fm.sent))
	}
	m := fm.sent[0]
	if m.To != "info@avatax.eu" || !strings.Contains(m.ReplyTo, "g@example.gr") || !strings.Contains(m.Body, "μια ερώτηση") {
		t.Errorf("unexpected mail: %+v", m)
	}
}

func TestContactRedirectWithoutJS(t *testing.T) {
	s, _ := testServer()
	w := post(s, "/api/contact", contactForm(s), false)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/el/efcharistoume/" {
		t.Fatalf("got %d %q", w.Code, w.Header().Get("Location"))
	}
	f := contactForm(s)
	f.Set("email", "nope")
	w = post(s, "/api/contact", f, false)
	if w.Header().Get("Location") != "/el/sfalma/" {
		t.Fatalf("error redirect: %q", w.Header().Get("Location"))
	}
}

func TestOpenRedirectBlocked(t *testing.T) {
	for _, p := range []string{"https://evil.com", "//evil.com", "/\\evil.com", "javascript:x"} {
		if got := localPath(p, "/"); got != "/" {
			t.Errorf("localPath(%q) = %q", p, got)
		}
	}
}

func TestContactValidation(t *testing.T) {
	s, fm := testServer()
	f := contactForm(s)
	f.Set("email", "not-an-email")
	f.Del("phone")
	f.Del("consent")
	w := post(s, "/api/contact", f, true)
	var e formError
	json.Unmarshal(w.Body.Bytes(), &e)
	if w.Code != 422 || e.Code != "invalid" {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if len(fm.sent) != 0 {
		t.Fatal("mail should not be sent")
	}
}

func TestHeaderInjectionStripped(t *testing.T) {
	s, fm := testServer()
	f := contactForm(s)
	f.Set("subject", "Hi\r\nBcc: victim@example.com")
	post(s, "/api/contact", f, true)
	if len(fm.sent) != 1 || strings.ContainsAny(fm.sent[0].Subject, "\r\n") {
		t.Fatalf("subject not sanitized: %q", fm.sent[0].Subject)
	}
	raw := string(fm.sent[0].build(t0))
	if strings.Contains(raw, "\r\nBcc:") {
		t.Fatal("header injection in built message")
	}
}

func TestHoneypot(t *testing.T) {
	s, fm := testServer()
	f := contactForm(s)
	f.Set("website", "http://spam")
	w := post(s, "/api/contact", f, true)
	if w.Code != 200 || len(fm.sent) != 0 {
		t.Fatalf("honeypot: status %d, sent %d", w.Code, len(fm.sent))
	}
}

func TestTokenChecks(t *testing.T) {
	s, _ := testServer()
	cases := map[string]string{
		"missing":  "",
		"too fast": tokenAt(s, time.Second),
		"expired":  tokenAt(s, 2*time.Hour),
		"tampered": strings.Replace(tokenAt(s, 10*time.Second), "1", "2", 1),
	}
	for name, tok := range cases {
		f := contactForm(s)
		f.Set("token", tok)
		if w := post(s, "/api/contact", f, true); w.Code != 400 {
			t.Errorf("%s: expected 400, got %d", name, w.Code)
		}
	}
}

func TestRateLimit(t *testing.T) {
	s, _ := testServer()
	s.limiter = newRateLimiter(3, 10*time.Minute)
	for i := 0; i < 3; i++ {
		if w := post(s, "/api/contact", contactForm(s), true); w.Code != 200 {
			t.Fatalf("request %d: %d", i, w.Code)
		}
	}
	if w := post(s, "/api/contact", contactForm(s), true); w.Code != 429 {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	s.now = func() time.Time { return t0.Add(11 * time.Minute) }
	f := contactForm(s)
	f.Set("token", tokenAt(s, -11*time.Minute+10*time.Second))
	if w := post(s, "/api/contact", f, true); w.Code != 200 {
		t.Fatalf("after window: %d %s", w.Code, w.Body)
	}
}

func TestMailerFailure(t *testing.T) {
	s, fm := testServer()
	fm.err = errTooFast
	if w := post(s, "/api/contact", contactForm(s), true); w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func cvRequest(s *server, filename string, content []byte) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range map[string]string{
		"lang": "de", "position": "Junior Auditor", "name": "Anna Müller", "email": "anna@example.de",
		"start_date": "2026-11-01", "commute": "30", "salary": "1500", "consent": "1",
		"token": tokenAt(s, 30*time.Second),
	} {
		mw.WriteField(k, v)
	}
	if filename != "" {
		fw, _ := mw.CreateFormFile("cv", filename)
		fw.Write(content)
	}
	mw.Close()
	req := httptest.NewRequest("POST", "/api/cv", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, req)
	return w
}

func TestCVOK(t *testing.T) {
	s, fm := testServer()
	w := cvRequest(s, "lebenslauf.pdf", []byte("%PDF-1.7 fake pdf"))
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	m := fm.sent[0]
	if m.To != "cv@avatax.eu" || len(m.Attachments) != 1 || m.Attachments[0].Name != "CV_Anna_Mller.pdf" {
		t.Fatalf("unexpected mail: to=%s atts=%+v", m.To, m.Attachments)
	}
	raw := string(m.build(t0))
	for _, want := range []string{"multipart/mixed", "Content-Disposition: attachment", "JVBERi0xLjcgZmFrZSBwZGY="} {
		if !strings.Contains(raw, want) {
			t.Errorf("built message lacks %q", want)
		}
	}
}

func TestCVRejects(t *testing.T) {
	s, fm := testServer()
	if w := cvRequest(s, "cv.exe", []byte("MZ...")); w.Code != 422 || !strings.Contains(w.Body.String(), "bad_file") {
		t.Errorf("exe: %d %s", w.Code, w.Body)
	}
	if w := cvRequest(s, "cv.pdf", []byte("MZ not a pdf")); w.Code != 422 {
		t.Errorf("fake pdf: %d", w.Code)
	}
	if w := cvRequest(s, "", nil); w.Code != 422 {
		t.Errorf("missing file: %d", w.Code)
	}
	big := append([]byte("%PDF-"), make([]byte, 2<<20)...)
	if w := cvRequest(s, "cv.pdf", big); w.Code != 413 {
		t.Errorf("too large: %d %s", w.Code, w.Body)
	}
	if len(fm.sent) != 0 {
		t.Errorf("no mail expected, got %d", len(fm.sent))
	}
}

func TestBuildPlainMessage(t *testing.T) {
	m := message{From: "Website <web@avatax.eu>", To: "info@avatax.eu", ReplyTo: `"Δοκιμή" <x@y.gr>`, Subject: "Γεια σου", Body: "Zeile1\nÄÖÜ"}
	raw := string(m.build(t0))
	for _, want := range []string{"Subject: =?utf-8?q?", "Reply-To: =?utf-8?", "Content-Transfer-Encoding: quoted-printable", "Zeile1\r\n"} {
		if !strings.Contains(raw, want) {
			t.Errorf("missing %q in\n%s", want, raw)
		}
	}
}

func TestContactMultipart(t *testing.T) {
	s, fm := testServer()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range contactForm(s) {
		mw.WriteField(k, v[0])
	}
	mw.Close()
	req := httptest.NewRequest("POST", "/api/contact", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, req)
	if w.Code != 200 || len(fm.sent) != 1 {
		t.Fatalf("multipart contact: %d %s", w.Code, w.Body)
	}
}

func TestSiteNameInMail(t *testing.T) {
	s, fm := testServer()
	post(s, "/api/contact", contactForm(s), true)
	if len(fm.sent) != 1 || !strings.HasPrefix(fm.sent[0].Subject, "[avatax.eu] ") || !strings.Contains(fm.sent[0].Body, "submitted on avatax.eu") {
		t.Fatalf("site name missing: %q / %q", fm.sent[0].Subject, fm.sent[0].Body)
	}
}
