package mailer

import (
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

// Send delivers a plain-text email to the given recipient via the relay.
func (c Config) Send(to, subject, body string) error {
	addr := c.Host + ":" + c.Port
	auth := smtp.PlainAuth("", c.User, c.Pass, c.Host)
	msg := buildMessage(c.From, to, subject, body)
	return smtp.SendMail(addr, auth, c.From, []string{to}, msg)
}

func buildMessage(from, to, subject, body string) []byte {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}