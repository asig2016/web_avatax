package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type attachment struct {
	Name        string
	ContentType string
	Data        []byte
}

type message struct {
	From, To, ReplyTo, Subject, Body string
	Attachments                      []attachment
}

type mailer interface {
	Send(message) error
}

// encodeAddress renders "Name <addr>" with an RFC 2047 encoded display name.
func encodeAddress(s string) string {
	a, err := mail.ParseAddress(s)
	if err != nil {
		return s
	}
	return a.String()
}

func envelopeAddress(s string) string {
	if a, err := mail.ParseAddress(s); err == nil {
		return a.Address
	}
	return s
}

// build renders the message as RFC 5322 bytes: a text/plain part (UTF-8,
// quoted-printable) plus base64 attachments in a multipart/mixed container.
func (m message) build(now time.Time) []byte {
	var buf bytes.Buffer
	h := func(k, v string) { fmt.Fprintf(&buf, "%s: %s\r\n", k, v) }

	domain := "localhost"
	if at := strings.LastIndex(envelopeAddress(m.From), "@"); at >= 0 {
		domain = envelopeAddress(m.From)[at+1:]
	}
	h("From", encodeAddress(m.From))
	h("To", encodeAddress(m.To))
	if m.ReplyTo != "" {
		h("Reply-To", encodeAddress(m.ReplyTo))
	}
	h("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	h("Date", now.Format(time.RFC1123Z))
	h("Message-ID", fmt.Sprintf("<%s.%s@%s>", strconv.FormatInt(now.UnixNano(), 36), randHex(6), domain))
	h("MIME-Version", "1.0")

	textPart := func() {
		h("Content-Type", "text/plain; charset=utf-8")
		h("Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		qp := quotedprintable.NewWriter(&buf)
		qp.Write([]byte(strings.ReplaceAll(m.Body, "\n", "\r\n")))
		qp.Close()
		buf.WriteString("\r\n")
	}

	if len(m.Attachments) == 0 {
		textPart()
		return buf.Bytes()
	}

	boundary := "=_" + randHex(16)
	h("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%q", boundary))
	buf.WriteString("\r\nThis is a multi-part message in MIME format.\r\n")
	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	textPart()
	for _, a := range m.Attachments {
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		name := mime.QEncoding.Encode("utf-8", a.Name)
		h("Content-Type", fmt.Sprintf("%s; name=%q", a.ContentType, name))
		h("Content-Transfer-Encoding", "base64")
		h("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
		buf.WriteString("\r\n")
		enc := base64.StdEncoding.EncodeToString(a.Data)
		for len(enc) > 76 {
			buf.WriteString(enc[:76] + "\r\n")
			enc = enc[76:]
		}
		buf.WriteString(enc + "\r\n")
	}
	fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	return buf.Bytes()
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// smtpMailer delivers via an SMTP relay (the mail server of the domain in production,
// Mailpit on the laptop).
type smtpMailer struct{ cfg config }

func (s *smtpMailer) Send(m message) error {
	c := s.cfg
	addr := net.JoinHostPort(c.SMTPHost, strconv.Itoa(c.SMTPPort))
	tlsCfg := &tls.Config{ServerName: c.SMTPHost, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: 15 * time.Second}

	var conn net.Conn
	var err error
	if c.SMTPTLS == "tls" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("connect %s: %w", addr, err)
	}
	conn.SetDeadline(time.Now().Add(60 * time.Second))

	cl, err := smtp.NewClient(conn, c.SMTPHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp greeting: %w", err)
	}
	defer cl.Close()

	if c.SMTPTLS == "starttls" {
		if err := cl.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if c.SMTPUser != "" {
		if err := cl.Auth(smtp.PlainAuth("", c.SMTPUser, c.SMTPPass, c.SMTPHost)); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	if err := cl.Mail(envelopeAddress(m.From)); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	for _, to := range strings.Split(m.To, ",") {
		if err := cl.Rcpt(envelopeAddress(strings.TrimSpace(to))); err != nil {
			return fmt.Errorf("RCPT TO %s: %w", to, err)
		}
	}
	wc, err := cl.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := wc.Write(m.build(time.Now())); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("finishing message: %w", err)
	}
	return cl.Quit()
}
