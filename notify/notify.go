// Package notify delivers monitoring alerts over a channel (currently email).
package notify

import (
	"fmt"
	"html"
	"strings"

	"github.com/TheEinshine/open_shine/db"
	"github.com/TheEinshine/open_shine/mailer"
)

// Notifier delivers an alert somewhere. Implementations should be safe to call
// from the monitor loop and must not block indefinitely.
type Notifier interface {
	Alert(a db.Alert) error
}

// Noop is a Notifier that drops alerts — used when SMTP isn't configured so the
// monitor still samples metrics and records alerts to the DB.
type Noop struct{}

func (Noop) Alert(db.Alert) error { return nil }

// EmailNotifier sends alerts to the configured mail recipient (reusing the
// heartbeat recipient from mail_settings).
type EmailNotifier struct {
	Store *db.Store
	SMTP  mailer.Config
}

func (n EmailNotifier) Alert(a db.Alert) error {
	s, err := n.Store.GetSettings()
	if err != nil {
		return err
	}
	if s.Recipient == "" {
		return nil // nowhere to send
	}
	verb := "BREACH"
	if a.State == "recovered" {
		verb = "RECOVERED"
	}
	subject := fmt.Sprintf("[Open Shine] %s — %s", verb, a.Source)
	text, htmlBody := renderAlert(a)
	return n.SMTP.SendMessage(mailer.Message{
		To:       s.Recipient,
		Subject:  subject,
		Text:     text,
		HTML:     htmlBody,
		FromName: s.SenderName,
	})
}

// renderAlert builds a compact dark alert email matching the report aesthetic.
func renderAlert(a db.Alert) (text, htmlBody string) {
	accent := "#f87171" // breach = red
	label := "BREACH"
	if a.State == "recovered" {
		accent = "#34d399"
		label = "RECOVERED"
	}

	text = fmt.Sprintf("OPEN SHINE — %s\n%s\n\n%s\n%s\n",
		label, a.Source, a.Message, a.TS.Local().Format("Mon, 02 Jan 2006 15:04:05 MST"))

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"></head>`)
	b.WriteString(`<body style="margin:0;padding:0;background-color:#000000;">`)
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#000000;"><tr><td align="center" style="padding:24px 12px;">`)
	b.WriteString(`<table role="presentation" width="520" cellpadding="0" cellspacing="0" border="0" style="width:520px;max-width:520px;background-color:#0d0d0f;border:1px solid #1f1f23;border-left:3px solid ` + accent + `;border-radius:10px;overflow:hidden;">`)
	fmt.Fprintf(&b, `<tr><td style="padding:20px 24px 8px 24px;font-family:-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;font-size:13px;font-weight:600;letter-spacing:1px;color:%s;">%s</td></tr>`, accent, label)
	fmt.Fprintf(&b, `<tr><td style="padding:0 24px 4px 24px;font-family:'SFMono-Regular',Consolas,Menlo,monospace;font-size:15px;color:#e8e8ea;">%s</td></tr>`, html.EscapeString(a.Source))
	fmt.Fprintf(&b, `<tr><td style="padding:8px 24px 16px 24px;font-family:'SFMono-Regular',Consolas,Menlo,monospace;font-size:13px;color:#9a9aa0;word-break:break-word;">%s</td></tr>`, html.EscapeString(a.Message))
	fmt.Fprintf(&b, `<tr><td style="padding:14px 24px;border-top:1px solid #1f1f23;font-family:'SFMono-Regular',Consolas,Menlo,monospace;font-size:11px;color:#6b6b70;">open_shine · %s</td></tr>`, html.EscapeString(a.TS.Local().Format("Mon, 02 Jan 2006 15:04:05 MST")))
	b.WriteString(`</table></td></tr></table></body></html>`)
	return text, b.String()
}
