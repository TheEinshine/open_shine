package mailer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

// Config holds the SMTP connection settings, all read from environment
// variables. The recipient is NOT here — it lives in the DB (mail_settings)
// so you can change who gets the email without a redeploy.
type Config struct {
	Host string // SMTP_HOST, e.g. smtp.gmail.com
	Port string // SMTP_PORT, e.g. 587
	User string // SMTP_USER, your full Gmail address
	Pass string // SMTP_PASS, your 16-char app password
	From string // SMTP_FROM, optional; defaults to User
}

// LoadConfig reads SMTP settings from the environment and fails loudly if any
// are missing, so a misconfigured server logs the reason instead of silently
// doing nothing.
func LoadConfig() (Config, error) {
	c := Config{
		Host: os.Getenv("SMTP_HOST"),
		Port: os.Getenv("SMTP_PORT"),
		User: os.Getenv("SMTP_USER"),
		Pass: os.Getenv("SMTP_PASS"),
		From: os.Getenv("SMTP_FROM"),
	}
	if c.From == "" {
		c.From = c.User
	}

	var missing []string
	for name, val := range map[string]string{
		"SMTP_HOST": c.Host,
		"SMTP_PORT": c.Port,
		"SMTP_USER": c.User,
		"SMTP_PASS": c.Pass,
	} {
		if val == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return c, fmt.Errorf("missing env vars: %s", strings.Join(missing, ", "))
	}
	return c, nil
}

// Message is one outbound email. When HTML is non-empty the message is sent as
// multipart/alternative (plain text + HTML); otherwise it is plain text only.
type Message struct {
	To       string
	Subject  string
	Text     string
	HTML     string
	FromName string
}

// Send delivers a plain-text email. Kept for callers that don't need HTML.
func (c Config) Send(to, subject, body string) error {
	return c.SendMessage(Message{To: to, Subject: subject, Text: body})
}

// SendMessage delivers m via the configured relay. Go's smtp DotWriter handles
// CRLF line-ending conversion and dot-stuffing, so the body is built with plain
// "\n" line endings.
func (c Config) SendMessage(m Message) error {
	addr := c.Host + ":" + c.Port
	auth := smtp.PlainAuth("", c.User, c.Pass, c.Host)
	raw, err := c.build(m)
	if err != nil {
		return err
	}
	return smtp.SendMail(addr, auth, c.From, []string{m.To}, raw)
}

func (c Config) build(m Message) ([]byte, error) {
	var b strings.Builder
	from := c.From
	if m.FromName != "" {
		from = fmt.Sprintf("\"%s\" <%s>", m.FromName, c.From)
	}
	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeader(m.To) + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(m.Subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")

	if m.HTML == "" {
		b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
		b.WriteString(m.Text)
		return []byte(b.String()), nil
	}

	boundary, err := randomBoundary()
	if err != nil {
		return nil, err
	}
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")

	// Plain-text part first (least-capable client wins the fallback).
	text := m.Text
	if text == "" {
		text = "Open Shine heartbeat — enable HTML to view the report."
	}
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
	b.WriteString(text + "\r\n")

	// HTML part.
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n\r\n")
	b.WriteString(m.HTML + "\r\n")

	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String()), nil
}

func randomBoundary() (string, error) {
	var buf [18]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return "openshine-" + hex.EncodeToString(buf[:]), nil
}

// sanitizeHeader strips CR/LF so DB-sourced values (subject, recipient) can't
// inject extra headers.
func sanitizeHeader(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(v)
}