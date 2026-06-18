// Package report renders a heartbeat snapshot (system + runtime metrics and
// recent send history) into an email body. RenderHTML produces a dark-themed,
// table-based layout built for email clients (inline styles, table layout, no
// external assets); RenderText is the plain-text fallback.
package report

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/TheEinshine/open_shine/db"
	"github.com/TheEinshine/open_shine/sysstat"
)

// Dark palette. Kept muted and high-contrast so the table reads cleanly on the
// black background across clients.
const (
	colPage   = "#000000"
	colPanel  = "#0d0d0f"
	colBorder = "#1f1f23"
	colText   = "#e8e8ea"
	colMuted  = "#6b6b70"
	colLabel  = "#9a9aa0"
	colTrack  = "#1a1a1d"
	colGood   = "#34d399"
	colWarn   = "#fbbf24"
	colCrit   = "#f87171"
)

const (
	fontSans = "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif"
	fontMono = "'SFMono-Regular',Consolas,'Liberation Mono',Menlo,monospace"
)

// RenderHTML builds the dark, table-based HTML report.
func RenderHTML(s sysstat.Stats, logs []db.LogEntry) string {
	host := s.Hostname
	if host == "" {
		host = "unknown"
	}

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1"></head>`)
	b.WriteString(`<body style="margin:0;padding:0;background-color:` + colPage + `;">`)
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:` + colPage + `;"><tr><td align="center" style="padding:24px 12px;">`)
	b.WriteString(`<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px;max-width:600px;background-color:` + colPanel + `;border:1px solid ` + colBorder + `;border-radius:12px;overflow:hidden;">`)

	// Header: title + status dot + timestamp.
	b.WriteString(`<tr><td style="padding:22px 24px 18px 24px;border-bottom:1px solid ` + colBorder + `;">`)
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr>`)
	fmt.Fprintf(&b, `<td style="font-family:%s;font-size:16px;font-weight:600;color:%s;letter-spacing:0.5px;">`, fontSans, colText)
	fmt.Fprintf(&b, `<span style="display:inline-block;width:8px;height:8px;border-radius:50%%;background-color:%s;">&nbsp;</span>&nbsp; OPEN&nbsp;SHINE`, statusDot(s, logs))
	b.WriteString(`</td>`)
	fmt.Fprintf(&b, `<td align="right" style="font-family:%s;font-size:11px;color:%s;letter-spacing:1px;text-transform:uppercase;">heartbeat</td>`, fontMono, colMuted)
	b.WriteString(`</tr></table>`)
	fmt.Fprintf(&b, `<div style="margin-top:6px;font-family:%s;font-size:12px;color:%s;">%s</div>`, fontMono, colMuted, html.EscapeString(s.Time.Format("Mon, 02 Jan 2006 15:04:05 MST")))
	b.WriteString(`</td></tr>`)

	// System section.
	b.WriteString(sectionOpen("System"))
	if s.HostAvailable {
		b.WriteString(metric("CPU", pctValue(s.CPUPercent), bar(s.CPUPercent)))
		b.WriteString(metric("Memory", memValue(s), bar(s.MemPercent())))
		b.WriteString(metric("Storage", diskValue(s), bar(s.DiskPercent())))
		b.WriteString(metric("Load avg", loadValue(s), ""))
		b.WriteString(metric("Uptime", humanDuration(s.HostUp), ""))
	} else {
		b.WriteString(noteRow("Host metrics unavailable on this platform."))
	}
	b.WriteString(sectionClose())

	// Runtime section.
	b.WriteString(sectionOpen("Runtime"))
	b.WriteString(metric("Go", html.EscapeString(s.GoVersion), ""))
	b.WriteString(metric("Goroutines", fmt.Sprintf("%d", s.Goroutines), ""))
	b.WriteString(metric("Heap in use", humanBytes(s.HeapAlloc), ""))
	b.WriteString(metric("Host", html.EscapeString(host), ""))
	b.WriteString(sectionClose())

	// Log stack section.
	title := "Log stack"
	if len(logs) > 0 {
		title = fmt.Sprintf("Log stack · last %d", len(logs))
	}
	b.WriteString(sectionOpen(title))
	if len(logs) == 0 {
		b.WriteString(noteRow("No sends recorded yet."))
	} else {
		for _, e := range logs {
			b.WriteString(logRow(e))
		}
	}
	b.WriteString(sectionClose())

	// Footer.
	fmt.Fprintf(&b, `<tr><td style="padding:16px 24px;border-top:1px solid %s;font-family:%s;font-size:11px;color:%s;">open_shine · %s</td></tr>`, colBorder, fontMono, colMuted, html.EscapeString(host))

	b.WriteString(`</table></td></tr></table></body></html>`)
	return b.String()
}

// sectionOpen starts a labeled section: a padded cell holding a full-width
// table, with the uppercase section label as the first row.
func sectionOpen(title string) string {
	return `<tr><td style="padding:16px 24px 8px 24px;">` +
		`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">` +
		fmt.Sprintf(`<tr><td style="padding-bottom:6px;font-family:%s;font-size:10px;letter-spacing:2px;text-transform:uppercase;color:%s;">%s</td></tr>`, fontSans, colMuted, html.EscapeString(title))
}

func sectionClose() string {
	return `</table></td></tr>`
}

// metric renders one label/value row with an optional usage bar beneath it.
// value is inserted verbatim — callers must escape any dynamic text.
func metric(label, value, barHTML string) string {
	var b strings.Builder
	b.WriteString(`<tr><td style="padding:7px 0;">`)
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr>`)
	fmt.Fprintf(&b, `<td style="font-family:%s;font-size:13px;color:%s;">%s</td>`, fontSans, colLabel, html.EscapeString(label))
	fmt.Fprintf(&b, `<td align="right" style="font-family:%s;font-size:13px;color:%s;white-space:nowrap;">%s</td>`, fontMono, colText, value)
	b.WriteString(`</tr></table>`)
	if barHTML != "" {
		b.WriteString(`<div style="font-size:1px;line-height:6px;height:6px;">&nbsp;</div>`)
		b.WriteString(barHTML)
	}
	b.WriteString(`</td></tr>`)
	return b.String()
}

// bar renders a thin two-cell usage bar (filled vs track) coloured by severity.
func bar(pct float64) string {
	if pct < 0 {
		return ""
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct + 0.5)
	var b strings.Builder
	fmt.Fprintf(&b, `<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="border-radius:4px;overflow:hidden;background-color:%s;"><tr>`, colTrack)
	if filled > 0 {
		fmt.Fprintf(&b, `<td width="%d%%" height="6" style="height:6px;line-height:6px;font-size:1px;background-color:%s;">&nbsp;</td>`, filled, barColor(pct))
	}
	if filled < 100 {
		fmt.Fprintf(&b, `<td width="%d%%" height="6" style="height:6px;line-height:6px;font-size:1px;">&nbsp;</td>`, 100-filled)
	}
	b.WriteString(`</tr></table>`)
	return b.String()
}

func barColor(pct float64) string {
	switch {
	case pct >= 90:
		return colCrit
	case pct >= 70:
		return colWarn
	default:
		return colGood
	}
}

// logRow renders one mail_log entry: timestamp, status (colour-coded), message.
func logRow(e db.LogEntry) string {
	color := colGood
	if strings.ToLower(e.Status) != "ok" {
		color = colCrit
	}
	msg := e.Error
	if msg == "" {
		msg = "—"
	}
	if r := []rune(msg); len(r) > 90 {
		msg = string(r[:90]) + "…"
	}

	var b strings.Builder
	b.WriteString(`<tr>`)
	fmt.Fprintf(&b, `<td style="padding:5px 0;font-family:%s;font-size:12px;color:%s;white-space:nowrap;vertical-align:top;">%s</td>`, fontMono, colMuted, html.EscapeString(e.SentAt.Format("02 Jan 15:04")))
	fmt.Fprintf(&b, `<td style="padding:5px 12px;font-family:%s;font-size:12px;color:%s;white-space:nowrap;vertical-align:top;">%s</td>`, fontMono, color, html.EscapeString(strings.ToLower(e.Status)))
	fmt.Fprintf(&b, `<td style="padding:5px 0;font-family:%s;font-size:12px;color:%s;vertical-align:top;word-break:break-all;">%s</td>`, fontMono, colMuted, html.EscapeString(msg))
	b.WriteString(`</tr>`)
	return b.String()
}

func noteRow(text string) string {
	return fmt.Sprintf(`<tr><td style="padding:6px 0;font-family:%s;font-size:12px;color:%s;">%s</td></tr>`, fontMono, colMuted, html.EscapeString(text))
}

// statusDot picks the header indicator colour: red if the most recent send
// failed, else amber/red for high resource pressure, else green.
func statusDot(s sysstat.Stats, logs []db.LogEntry) string {
	if len(logs) > 0 && strings.ToLower(logs[0].Status) != "ok" {
		return colCrit
	}
	if s.HostAvailable {
		peak := s.CPUPercent
		if mp := s.MemPercent(); mp > peak {
			peak = mp
		}
		if dp := s.DiskPercent(); dp > peak {
			peak = dp
		}
		if peak >= 90 {
			return colCrit
		}
		if peak >= 70 {
			return colWarn
		}
	}
	return colGood
}

// RenderText is the plain-text fallback (used as the text/plain MIME part).
func RenderText(s sysstat.Stats, logs []db.LogEntry) string {
	host := s.Hostname
	if host == "" {
		host = "unknown"
	}

	var b strings.Builder
	b.WriteString("OPEN SHINE — heartbeat\n")
	b.WriteString(s.Time.Format("Mon, 02 Jan 2006 15:04:05 MST") + "\n\n")

	b.WriteString("SYSTEM\n")
	if s.HostAvailable {
		fmt.Fprintf(&b, "  %-12s %s\n", "CPU", pctValue(s.CPUPercent))
		fmt.Fprintf(&b, "  %-12s %s\n", "Memory", memValue(s))
		fmt.Fprintf(&b, "  %-12s %s\n", "Storage", diskValue(s))
		fmt.Fprintf(&b, "  %-12s %s\n", "Load avg", loadValue(s))
		fmt.Fprintf(&b, "  %-12s %s\n", "Uptime", humanDuration(s.HostUp))
	} else {
		b.WriteString("  host metrics unavailable on this platform\n")
	}

	b.WriteString("\nRUNTIME\n")
	fmt.Fprintf(&b, "  %-12s %s\n", "Go", s.GoVersion)
	fmt.Fprintf(&b, "  %-12s %d\n", "Goroutines", s.Goroutines)
	fmt.Fprintf(&b, "  %-12s %s\n", "Heap", humanBytes(s.HeapAlloc))
	fmt.Fprintf(&b, "  %-12s %s\n", "Host", host)

	if len(logs) == 0 {
		b.WriteString("\nLOG STACK\n  no sends recorded yet\n")
	} else {
		fmt.Fprintf(&b, "\nLOG STACK (last %d)\n", len(logs))
		for _, e := range logs {
			line := fmt.Sprintf("  %-13s %-6s", e.SentAt.Format("02 Jan 15:04"), strings.ToLower(e.Status))
			if e.Error != "" {
				line += " " + e.Error
			}
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\nopen_shine\n")
	return b.String()
}

func pctValue(p float64) string {
	if p < 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", p)
}

func memValue(s sysstat.Stats) string {
	if s.MemTotal == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%s / %s · %.0f%%", humanBytes(s.MemUsed), humanBytes(s.MemTotal), s.MemPercent())
}

func diskValue(s sysstat.Stats) string {
	if s.DiskTotal == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%s / %s · %.0f%%", humanBytes(s.DiskUsed), humanBytes(s.DiskTotal), s.DiskPercent())
}

func loadValue(s sysstat.Stats) string {
	return fmt.Sprintf("%.2f · %.2f · %.2f", s.Load1, s.Load5, s.Load15)
}

func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func humanDuration(d time.Duration) string {
	if d <= 0 {
		return "n/a"
	}
	d = d.Round(time.Minute)
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, h, m)
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}
