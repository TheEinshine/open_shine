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
	colPage   = "#09090b"
	colPanel  = "#0d0d0f"
	colPanel2 = "#141417"
	colBorder = "#1f1f23"
	colText   = "#e8e8ea"
	colMuted  = "#6b6b70"
	colLabel  = "#9a9aa0"
	colTrack  = "#1a1a1d"
	colGood   = "#34d399"
	colWarn   = "#fbbf24"
	colCrit   = "#f87171"
	colAccent = "#60a5fa"
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

	// Outer table (centers content).
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:` + colPage + `;"><tr><td align="center" style="padding:32px 12px;">`)

	// Main container card.
	b.WriteString(`<table role="presentation" width="620" cellpadding="0" cellspacing="0" border="0" style="width:620px;max-width:100%;background-color:` + colPanel + `;border:1px solid ` + colBorder + `;border-radius:16px;overflow:hidden;">`)

	// Accent gradient bar at top.
	b.WriteString(`<tr><td style="height:4px;background:linear-gradient(90deg,` + colAccent + `,#818cf8,#a78bfa);font-size:1px;line-height:1px;">&nbsp;</td></tr>`)

	// Header.
	headerBg := colPanel
	b.WriteString(`<tr><td style="padding:28px 32px 20px 32px;background-color:` + headerBg + `;">`)
	// Brand row.
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr>`)
	fmt.Fprintf(&b, `<td style="font-family:%s;font-size:18px;font-weight:700;color:%s;letter-spacing:0.3px;">`, fontSans, colText)
	fmt.Fprintf(&b, `<span style="display:inline-block;width:10px;height:10px;border-radius:50%%;background-color:%s;margin-right:8px;vertical-align:middle;"></span>`, statusDot(s, logs))
	b.WriteString(`Open Shine</td>`)
	fmt.Fprintf(&b, `<td align="right" style="font-family:%s;font-size:10px;color:%s;letter-spacing:2px;text-transform:uppercase;">HEARTBEAT</td>`, fontMono, colMuted)
	b.WriteString(`</tr></table>`)
	// Timestamp + hostname.
	fmt.Fprintf(&b, `<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="margin-top:12px;"><tr>`)
	fmt.Fprintf(&b, `<td style="font-family:%s;font-size:12px;color:%s;">%s</td>`, fontMono, colMuted, html.EscapeString(s.Time.Format("Monday, 02 Jan 2006 · 15:04:05 MST")))
	fmt.Fprintf(&b, `<td align="right" style="font-family:%s;font-size:12px;color:%s;">%s</td>`, fontMono, colMuted, html.EscapeString(host))
	b.WriteString(`</tr></table>`)
	b.WriteString(`</td></tr>`)

	// Divider.
	fmt.Fprintf(&b, `<tr><td style="height:1px;background-color:%s;font-size:1px;line-height:1px;">&nbsp;</td></tr>`, colBorder)

	// Status summary sentence.
	summaryColor := colGood
	summaryText := "All systems operating normally"
	summaryIcon := "✓"
	if s.HostAvailable {
		peak := s.CPUPercent
		if mp := s.MemPercent(); mp > peak {
			peak = mp
		}
		if dp := s.DiskPercent(); dp > peak {
			peak = dp
		}
		if peak >= 90 {
			summaryColor = colCrit
			summaryText = "Critical: resource usage above 90%"
			summaryIcon = "✕"
		} else if peak >= 70 {
			summaryColor = colWarn
			summaryText = "Warning: resource usage elevated"
			summaryIcon = "!"
		}
	} else {
		summaryColor = colMuted
		summaryText = "Host metrics unavailable on this platform"
		summaryIcon = "—"
	}
	fmt.Fprintf(&b, `<tr><td style="padding:20px 32px 16px 32px;">`)
	fmt.Fprintf(&b, `<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="background-color:%s;border-radius:10px;"><tr>`, colPanel2)
	fmt.Fprintf(&b, `<td style="padding:14px 20px;font-family:%s;font-size:13px;color:%s;font-weight:500;">`, fontSans, summaryColor)
	fmt.Fprintf(&b, `<span style="display:inline-block;width:22px;height:22px;border-radius:50%%;background-color:rgba(%s);text-align:center;line-height:22px;font-size:12px;font-weight:700;color:%s;margin-right:10px;vertical-align:middle;">%s</span>`, summaryBgRGBA(summaryColor), summaryColor, summaryIcon)
	fmt.Fprintf(&b, `%s</td></tr></table>`, summaryText)
	b.WriteString(`</td></tr>`)

	// System metrics cards (3 side by side).
	if s.HostAvailable {
		b.WriteString(`<tr><td style="padding:4px 32px 8px 32px;">`)
		fmt.Fprintf(&b, `<div style="font-family:%s;font-size:10px;letter-spacing:2px;text-transform:uppercase;color:%s;margin-bottom:12px;">System Resources</div>`, fontSans, colMuted)
		b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr>`)
		// CPU card.
		b.WriteString(metricCard("CPU", pctValue(s.CPUPercent), s.CPUPercent, "33%"))
		// Memory card.
		b.WriteString(metricCard("Memory", memValueShort(s), s.MemPercent(), "34%"))
		// Disk card.
		b.WriteString(metricCard("Storage", diskValueShort(s), s.DiskPercent(), "33%"))
		b.WriteString(`</tr></table>`)
		b.WriteString(`</td></tr>`)

		// Secondary metrics as info table.
		b.WriteString(`<tr><td style="padding:12px 32px 8px 32px;">`)
		b.WriteString(infoTable([]infoRow{
			{label: "Memory", value: memValue(s)},
			{label: "Storage", value: diskValue(s)},
			{label: "Load Average", value: loadValue(s)},
			{label: "Uptime", value: humanDuration(s.HostUp)},
		}))
		b.WriteString(`</td></tr>`)
	}

	// Runtime section.
	b.WriteString(`<tr><td style="padding:16px 32px 8px 32px;">`)
	fmt.Fprintf(&b, `<div style="font-family:%s;font-size:10px;letter-spacing:2px;text-transform:uppercase;color:%s;margin-bottom:12px;">Runtime</div>`, fontSans, colMuted)
	b.WriteString(infoTable([]infoRow{
		{label: "Go Version", value: s.GoVersion},
		{label: "Goroutines", value: fmt.Sprintf("%d", s.Goroutines)},
		{label: "Heap Allocated", value: humanBytes(s.HeapAlloc)},
		{label: "Hostname", value: host},
	}))
	b.WriteString(`</td></tr>`)

	// Environment section.
	b.WriteString(`<tr><td style="padding:16px 32px 8px 32px;">`)
	fmt.Fprintf(&b, `<div style="font-family:%s;font-size:10px;letter-spacing:2px;text-transform:uppercase;color:%s;margin-bottom:12px;">Environment</div>`, fontSans, colMuted)
	internetStatus := "Offline"
	if s.InternetUp {
		internetStatus = fmt.Sprintf("Online (%s)", s.InternetLatency.Round(time.Millisecond))
	}
	b.WriteString(infoTable([]infoRow{
		{label: "Latest Commit", value: s.LatestCommit},
		{label: "Internet Status", value: internetStatus},
	}))
	b.WriteString(`</td></tr>`)

	// Log history section.
	logTitle := "Recent Activity"
	if len(logs) > 0 {
		logTitle = fmt.Sprintf("Recent Activity · %d entries", len(logs))
	}
	b.WriteString(`<tr><td style="padding:16px 32px 8px 32px;">`)
	fmt.Fprintf(&b, `<div style="font-family:%s;font-size:10px;letter-spacing:2px;text-transform:uppercase;color:%s;margin-bottom:12px;">%s</div>`, fontSans, colMuted, logTitle)
	if len(logs) == 0 {
		fmt.Fprintf(&b, `<div style="padding:16px 0;font-family:%s;font-size:13px;color:%s;">No send history recorded yet.</div>`, fontSans, colMuted)
	} else {
		b.WriteString(logTable(logs))
	}
	b.WriteString(`</td></tr>`)

	// Footer.
	fmt.Fprintf(&b, `<tr><td style="height:1px;background-color:%s;font-size:1px;line-height:1px;">&nbsp;</td></tr>`, colBorder)
	b.WriteString(`<tr><td style="padding:20px 32px;">`)
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr>`)
	fmt.Fprintf(&b, `<td style="font-family:%s;font-size:11px;color:%s;">`, fontSans, colMuted)
	fmt.Fprintf(&b, `Sent by <span style="color:%s;font-weight:500;">Open Shine</span> · Automated heartbeat report`, colLabel)
	b.WriteString(`</td>`)
	fmt.Fprintf(&b, `<td align="right" style="font-family:%s;font-size:11px;color:%s;">%s</td>`, fontMono, colMuted, html.EscapeString(host))
	b.WriteString(`</tr></table>`)
	b.WriteString(`</td></tr>`)

	// Close.
	b.WriteString(`</table></td></tr></table></body></html>`)
	return b.String()
}

// metricCard renders one of the three main stat cards.
func metricCard(label, value string, pct float64, width string) string {
	var b strings.Builder
	color := barColor(pct)
	fmt.Fprintf(&b, `<td width="%s" style="padding:0 4px;" valign="top">`, width)
	fmt.Fprintf(&b, `<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="background-color:%s;border:1px solid %s;border-radius:10px;overflow:hidden;">`, colPanel2, colBorder)
	fmt.Fprintf(&b, `<tr><td style="padding:16px 16px 10px 16px;">`)
	fmt.Fprintf(&b, `<div style="font-family:%s;font-size:10px;letter-spacing:1.5px;text-transform:uppercase;color:%s;margin-bottom:6px;">%s</div>`, fontSans, colLabel, html.EscapeString(label))
	fmt.Fprintf(&b, `<div style="font-family:%s;font-size:24px;font-weight:700;color:%s;letter-spacing:-0.5px;">%s</div>`, fontMono, colText, html.EscapeString(value))
	b.WriteString(`</td></tr>`)
	// Progress bar.
	fmt.Fprintf(&b, `<tr><td style="padding:0 16px 14px 16px;">`)
	if pct >= 0 {
		filled := int(pct + 0.5)
		if filled > 100 {
			filled = 100
		}
		fmt.Fprintf(&b, `<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="border-radius:3px;overflow:hidden;background-color:%s;"><tr>`, colTrack)
		if filled > 0 {
			fmt.Fprintf(&b, `<td width="%d%%" height="4" style="height:4px;line-height:4px;font-size:1px;background-color:%s;border-radius:3px;">&nbsp;</td>`, filled, color)
		}
		if filled < 100 {
			fmt.Fprintf(&b, `<td width="%d%%" height="4" style="height:4px;line-height:4px;font-size:1px;">&nbsp;</td>`, 100-filled)
		}
		b.WriteString(`</tr></table>`)
	}
	b.WriteString(`</td></tr></table></td>`)
	return b.String()
}

type infoRow struct {
	label string
	value string
}

// infoTable renders a clean key/value table.
func infoTable(rows []infoRow) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="background-color:%s;border:1px solid %s;border-radius:10px;overflow:hidden;">`, colPanel2, colBorder)
	for i, r := range rows {
		borderBottom := fmt.Sprintf("border-bottom:1px solid %s;", colBorder)
		if i == len(rows)-1 {
			borderBottom = ""
		}
		b.WriteString(`<tr>`)
		fmt.Fprintf(&b, `<td style="padding:10px 16px;%sfont-family:%s;font-size:13px;color:%s;white-space:nowrap;">%s</td>`, borderBottom, fontSans, colLabel, html.EscapeString(r.label))
		fmt.Fprintf(&b, `<td align="right" style="padding:10px 16px;%sfont-family:%s;font-size:13px;color:%s;white-space:nowrap;">%s</td>`, borderBottom, fontMono, colText, html.EscapeString(r.value))
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</table>`)
	return b.String()
}

// logTable renders the send history in a structured table with status chips.
func logTable(logs []db.LogEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="background-color:%s;border:1px solid %s;border-radius:10px;overflow:hidden;">`, colPanel2, colBorder)

	// Header row.
	b.WriteString(`<tr>`)
	fmt.Fprintf(&b, `<td style="padding:10px 16px;border-bottom:1px solid %s;font-family:%s;font-size:10px;letter-spacing:1.5px;text-transform:uppercase;color:%s;">Time</td>`, colBorder, fontSans, colMuted)
	fmt.Fprintf(&b, `<td style="padding:10px 12px;border-bottom:1px solid %s;font-family:%s;font-size:10px;letter-spacing:1.5px;text-transform:uppercase;color:%s;">Status</td>`, colBorder, fontSans, colMuted)
	fmt.Fprintf(&b, `<td style="padding:10px 16px;border-bottom:1px solid %s;font-family:%s;font-size:10px;letter-spacing:1.5px;text-transform:uppercase;color:%s;">Details</td>`, colBorder, fontSans, colMuted)
	b.WriteString(`</tr>`)

	for i, e := range logs {
		isOk := strings.ToLower(e.Status) == "ok"
		statusColor := colCrit
		statusBg := "rgba(248,113,113,0.12)"
		statusLabel := strings.ToUpper(e.Status)
		if isOk {
			statusColor = colGood
			statusBg = "rgba(52,211,153,0.12)"
			statusLabel = "OK"
		}
		msg := e.Error
		if msg == "" {
			msg = "Delivered successfully"
		}
		if r := []rune(msg); len(r) > 80 {
			msg = string(r[:80]) + "…"
		}

		borderBottom := fmt.Sprintf("border-bottom:1px solid %s;", colBorder)
		if i == len(logs)-1 {
			borderBottom = ""
		}

		b.WriteString(`<tr>`)
		fmt.Fprintf(&b, `<td style="padding:10px 16px;%sfont-family:%s;font-size:12px;color:%s;white-space:nowrap;vertical-align:top;">%s</td>`,
			borderBottom, fontMono, colLabel, html.EscapeString(e.SentAt.Local().Format("02 Jan 15:04")))
		fmt.Fprintf(&b, `<td style="padding:10px 12px;%svertical-align:top;">`, borderBottom)
		fmt.Fprintf(&b, `<span style="display:inline-block;font-family:%s;font-size:10px;font-weight:600;letter-spacing:1px;color:%s;background-color:%s;padding:3px 8px;border-radius:4px;">%s</span>`,
			fontMono, statusColor, statusBg, statusLabel)
		b.WriteString(`</td>`)
		fmt.Fprintf(&b, `<td style="padding:10px 16px;%sfont-family:%s;font-size:12px;color:%s;vertical-align:top;word-break:break-word;">%s</td>`,
			borderBottom, fontSans, colMuted, html.EscapeString(msg))
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</table>`)
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

// summaryBgRGBA returns an rgba value for the summary icon background based on the status color hex.
func summaryBgRGBA(hex string) string {
	switch hex {
	case colCrit:
		return "248,113,113,0.15"
	case colWarn:
		return "251,191,36,0.15"
	default:
		return "52,211,153,0.15"
	}
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
	b.WriteString("═══════════════════════════════════════════════════\n")
	b.WriteString("  OPEN SHINE · Heartbeat Report\n")
	b.WriteString("═══════════════════════════════════════════════════\n\n")
	b.WriteString("  " + s.Time.Format("Monday, 02 Jan 2006 · 15:04:05 MST") + "\n")
	b.WriteString("  Host: " + host + "\n\n")

	b.WriteString("───────────────────────────────────────────────────\n")
	b.WriteString("  SYSTEM RESOURCES\n")
	b.WriteString("───────────────────────────────────────────────────\n")
	if s.HostAvailable {
		fmt.Fprintf(&b, "  %-16s %s\n", "CPU Usage", pctValue(s.CPUPercent))
		fmt.Fprintf(&b, "  %-16s %s\n", "Memory", memValue(s))
		fmt.Fprintf(&b, "  %-16s %s\n", "Storage", diskValue(s))
		fmt.Fprintf(&b, "  %-16s %s\n", "Load Average", loadValue(s))
		fmt.Fprintf(&b, "  %-16s %s\n", "Uptime", humanDuration(s.HostUp))
	} else {
		b.WriteString("  Host metrics unavailable on this platform.\n")
	}

	b.WriteString("\n───────────────────────────────────────────────────\n")
	b.WriteString("  RUNTIME\n")
	b.WriteString("───────────────────────────────────────────────────\n")
	fmt.Fprintf(&b, "  %-16s %s\n", "Go Version", s.GoVersion)
	fmt.Fprintf(&b, "  %-16s %d\n", "Goroutines", s.Goroutines)
	fmt.Fprintf(&b, "  %-16s %s\n", "Heap Allocated", humanBytes(s.HeapAlloc))
	fmt.Fprintf(&b, "  %-16s %s\n", "Hostname", host)

	b.WriteString("\n───────────────────────────────────────────────────\n")
	b.WriteString("  ENVIRONMENT\n")
	b.WriteString("───────────────────────────────────────────────────\n")
	internetStatus := "Offline"
	if s.InternetUp {
		internetStatus = fmt.Sprintf("Online (%s)", s.InternetLatency.Round(time.Millisecond))
	}
	fmt.Fprintf(&b, "  %-16s %s\n", "Latest Commit", s.LatestCommit)
	fmt.Fprintf(&b, "  %-16s %s\n", "Internet Status", internetStatus)

	b.WriteString("\n───────────────────────────────────────────────────\n")
	if len(logs) == 0 {
		b.WriteString("  RECENT ACTIVITY\n")
		b.WriteString("───────────────────────────────────────────────────\n")
		b.WriteString("  No send history recorded yet.\n")
	} else {
		fmt.Fprintf(&b, "  RECENT ACTIVITY · %d entries\n", len(logs))
		b.WriteString("───────────────────────────────────────────────────\n")
		for _, e := range logs {
			status := strings.ToUpper(e.Status)
			detail := "Delivered successfully"
			if e.Error != "" {
				detail = e.Error
			}
			fmt.Fprintf(&b, "  %-14s %-8s %s\n",
				e.SentAt.Local().Format("02 Jan 15:04"),
				status,
				detail,
			)
		}
	}

	b.WriteString("\n═══════════════════════════════════════════════════\n")
	b.WriteString("  Open Shine · Automated heartbeat report\n")
	b.WriteString("═══════════════════════════════════════════════════\n")
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

func memValueShort(s sysstat.Stats) string {
	if s.MemTotal == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.0f%%", s.MemPercent())
}

func diskValue(s sysstat.Stats) string {
	if s.DiskTotal == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%s / %s · %.0f%%", humanBytes(s.DiskUsed), humanBytes(s.DiskTotal), s.DiskPercent())
}

func diskValueShort(s sysstat.Stats) string {
	if s.DiskTotal == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.0f%%", s.DiskPercent())
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
