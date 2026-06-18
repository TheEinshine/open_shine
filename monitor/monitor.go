// Package monitor samples host metrics into history, evaluates threshold rules,
// health-checks external targets, and fires alerts on state transitions.
package monitor

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/TheEinshine/open_shine/db"
	"github.com/TheEinshine/open_shine/notify"
	"github.com/TheEinshine/open_shine/sysstat"
)

// Engine runs the periodic monitoring loop.
type Engine struct {
	store    *db.Store
	notifier notify.Notifier
	interval time.Duration
	retain   time.Duration
	httpc    *http.Client

	mu       sync.Mutex
	breached map[string]bool // source -> currently in breach
}

// New builds an Engine. interval is the sampling cadence; retain is how long
// metric history is kept.
func New(store *db.Store, n notify.Notifier, interval, retain time.Duration) *Engine {
	if interval <= 0 {
		interval = time.Minute
	}
	if retain <= 0 {
		retain = 7 * 24 * time.Hour
	}
	return &Engine{
		store:    store,
		notifier: n,
		interval: interval,
		retain:   retain,
		httpc:    &http.Client{Timeout: 10 * time.Second},
		breached: map[string]bool{},
	}
}

// Run ticks until ctx is cancelled. The first tick happens immediately.
func (e *Engine) Run(ctx context.Context) {
	log.Printf("monitor started (interval %s)", e.interval)
	e.tick(ctx)
	t := time.NewTicker(e.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("monitor stopped")
			return
		case <-t.C:
			e.tick(ctx)
		}
	}
}

func (e *Engine) tick(ctx context.Context) {
	stats := sysstat.Collect()
	if stats.HostAvailable {
		if err := e.store.InsertMetric(db.MetricPoint{
			TS:    time.Now(),
			CPU:   nonNeg(stats.CPUPercent),
			Mem:   nonNeg(stats.MemPercent()),
			Disk:  nonNeg(stats.DiskPercent()),
			Load1: stats.Load1,
		}); err != nil {
			log.Printf("monitor: insert metric failed: %v", err)
		}
		e.evalThresholds(stats)
	}
	e.evalTargets(ctx)

	if err := e.store.PruneMetrics(time.Now().Add(-e.retain)); err != nil {
		log.Printf("monitor: prune metrics failed: %v", err)
	}
	if err := e.store.DeleteExpiredSessions(); err != nil {
		log.Printf("monitor: prune sessions failed: %v", err)
	}
}

func (e *Engine) evalThresholds(stats sysstat.Stats) {
	thresholds, err := e.store.ListThresholds(true)
	if err != nil {
		log.Printf("monitor: list thresholds failed: %v", err)
		return
	}
	for _, t := range thresholds {
		val, ok := metricValue(stats, t.Metric)
		if !ok {
			continue
		}
		source := fmt.Sprintf("threshold:%s#%d", t.Metric, t.ID)
		breach := compare(val, t.Op, t.Value)
		msg := fmt.Sprintf("%s is %.1f (rule: %s %s %.1f)", t.Metric, val, t.Metric, t.Op, t.Value)
		e.setState(source, breach, msg)
	}
}

func (e *Engine) evalTargets(ctx context.Context) {
	targets, err := e.store.ListTargets(true)
	if err != nil {
		log.Printf("monitor: list targets failed: %v", err)
		return
	}
	for _, t := range targets {
		ok, detail := e.checkTarget(ctx, t)
		source := fmt.Sprintf("target:%s", t.Name)
		status := "DOWN"
		if ok {
			status = "up"
		}
		msg := fmt.Sprintf("%s %s (%s) — %s", t.Kind, t.Address, status, detail)
		e.setState(source, !ok, msg)
	}
}

func (e *Engine) checkTarget(ctx context.Context, t db.Target) (bool, string) {
	switch t.Kind {
	case "http":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.Address, nil)
		if err != nil {
			return false, err.Error()
		}
		resp, err := e.httpc.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer resp.Body.Close()
		ok := resp.StatusCode >= 200 && resp.StatusCode < 400
		return ok, fmt.Sprintf("HTTP %d", resp.StatusCode)
	case "tcp":
		d := net.Dialer{Timeout: 5 * time.Second}
		conn, err := d.DialContext(ctx, "tcp", t.Address)
		if err != nil {
			return false, err.Error()
		}
		_ = conn.Close()
		return true, "connected"
	default:
		return false, "unknown check kind " + t.Kind
	}
}

// setState records a state change and alerts only on transition (ok<->breach),
// so a sustained breach doesn't spam.
func (e *Engine) setState(source string, breach bool, msg string) {
	e.mu.Lock()
	was, seen := e.breached[source]
	e.breached[source] = breach
	e.mu.Unlock()

	// First observation of an OK source: record baseline, no alert.
	if !seen && !breach {
		return
	}
	if seen && was == breach {
		return // no change
	}

	state := "recovered"
	if breach {
		state = "breach"
	}
	a := db.Alert{TS: time.Now(), Source: source, State: state, Message: msg}
	if err := e.store.InsertAlert(a); err != nil {
		log.Printf("monitor: insert alert failed: %v", err)
	}
	if err := e.notifier.Alert(a); err != nil {
		log.Printf("monitor: notify failed: %v", err)
	}
	log.Printf("alert %s %s: %s", state, source, msg)
}

func metricValue(s sysstat.Stats, metric string) (float64, bool) {
	switch metric {
	case "cpu":
		return s.CPUPercent, s.CPUPercent >= 0
	case "mem":
		return s.MemPercent(), s.MemTotal > 0
	case "disk":
		return s.DiskPercent(), s.DiskTotal > 0
	case "load1":
		return s.Load1, true
	default:
		return 0, false
	}
}

func compare(val float64, op string, threshold float64) bool {
	switch op {
	case "gt":
		return val > threshold
	case "gte":
		return val >= threshold
	case "lt":
		return val < threshold
	case "lte":
		return val <= threshold
	default:
		return false
	}
}

func nonNeg(f float64) float64 {
	if f < 0 {
		return 0
	}
	return f
}
