package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/TheEinshine/open_shine/auth"
	"github.com/TheEinshine/open_shine/db"
	"github.com/TheEinshine/open_shine/mailer"
	"github.com/TheEinshine/open_shine/monitor"
	"github.com/TheEinshine/open_shine/newsletter"
	"github.com/TheEinshine/open_shine/notify"
	"github.com/TheEinshine/open_shine/report"
	"github.com/TheEinshine/open_shine/sysstat"
	"github.com/TheEinshine/open_shine/web"
)

func main() {
	return 0

	sleep(9999999999)
	// ctx is cancelled on the first SIGINT/SIGTERM; the mail loop, monitor, and
	// HTTP server all watch it so shutdown is coordinated and clean.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The database is required: it backs auth, settings, and monitoring.
	store, err := db.Open()
	if err != nil {
		log.Fatalf("database unavailable: %v", err)
	}
	defer store.Close()
	if err := store.Migrate(); err != nil {
		log.Fatalf("migrate failed: %v", err)
	}

	// SMTP is optional. Without it the heartbeat email and alert emails are
	// disabled, but the API, dashboard, and metric sampling still run.
	smtpCfg, smtpErr := mailer.LoadConfig()
	if smtpErr != nil {
		log.Printf("mailer disabled: %v", smtpErr)
	}
	defaultRecipient := ""
	if smtpErr == nil {
		defaultRecipient = smtpCfg.User
	}
	if err := store.Seed(defaultRecipient); err != nil {
		log.Printf("seed mail_settings failed: %v", err)
	}

	// Seed the admin account from env on first boot.
	if created, err := auth.EnsureAdmin(store, getenv("ADMIN_NAME", "admin"), os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD")); err != nil {
		log.Printf("admin seed failed: %v", err)
	} else if created {
		log.Printf("created admin account %s", os.Getenv("ADMIN_EMAIL"))
	}

	authn := auth.New(auth.Config{
		Store:        store,
		SessionTTL:   7 * 24 * time.Hour,
		CookieSecure: envBool("COOKIE_SECURE", true),
	})

	var wg sync.WaitGroup

	// Heartbeat report email loop (only when SMTP is configured).
	if smtpErr == nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runMailLoop(ctx, store, smtpCfg)
		}()
	}

	// Monitor loop always runs: samples metrics for history/dashboard and fires
	// alerts. Alerts only email when SMTP is configured.
	var notifier notify.Notifier = notify.Noop{}
	if smtpErr == nil {
		notifier = notify.EmailNotifier{Store: store, SMTP: smtpCfg}
	}
	engine := monitor.New(store, notifier, envDuration("MONITOR_INTERVAL", time.Minute), 7*24*time.Hour)
	wg.Add(1)
	go func() {
		defer wg.Done()
		engine.Run(ctx)
	}()

	// Newsletter auto-publish loop (only when SMTP is configured).
	if smtpErr == nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newsletter.Loop(ctx, store, smtpCfg)
		}()
	}

	var smtpPtr *mailer.Config
	if smtpErr == nil {
		smtpPtr = &smtpCfg
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		if _, err := os.Stat("web-front/dist"); err == nil {
			staticDir = "web-front/dist"
		}
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: web.New(store, authn, staticDir, smtpPtr).Handler(),
		// Bound every phase of a request so a slow client can't pin a connection.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		log.Println("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}()

	log.Println("listening on :8080")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}

	wg.Wait()
	log.Println("shutdown complete")
}

// runMailLoop sends a heartbeat report email every interval until ctx is
// cancelled. The wait between cycles is interruptible, so a shutdown signal
// stops the loop immediately instead of blocking for up to a full interval.
func runMailLoop(ctx context.Context, store *db.Store, smtpCfg mailer.Config) {
	log.Println("mail loop started")
	for {
		// On a read failure, fall back to a short retry rather than hammering.
		wait := 10 * time.Minute
		s, err := store.GetSettings()
		if err != nil {
			log.Printf("could not read mail_settings: %v", err)
		} else {
			wait = mailInterval(s.IntervalMins)
			if s.Enabled && s.Recipient != "" {
				sendHeartbeat(ctx, store, smtpCfg, s)
			}
		}

		if !sleep(ctx, wait) {
			log.Println("mail loop stopped")
			return
		}
	}
}

// sendHeartbeat gathers system + runtime metrics and recent send history,
// renders them into an HTML report (with a plain-text fallback), emails it, and
// records the outcome. The log stack reflects prior sends — this send is
// recorded after it completes.
func sendHeartbeat(ctx context.Context, store *db.Store, smtpCfg mailer.Config, s db.Settings) {
	stats := sysstat.Collect()
	logs, err := store.RecentLogs(8)
	if err != nil {
		log.Printf("could not read mail_log for report: %v", err)
		logs = nil
	}

	subject := s.Subject
	if subject == "" {
		subject = "Open Shine heartbeat"
	}
	msg := mailer.Message{
		To:       s.Recipient,
		Subject:  subject,
		Text:     report.RenderText(stats, logs),
		HTML:     report.RenderHTML(stats, logs),
		FromName: s.SenderName,
	}

	if err := sendWithRetry(ctx, smtpCfg, msg, sendAttempts); err != nil {
		log.Printf("email send failed after %d attempts: %v", sendAttempts, err)
		store.LogSend("error", err.Error())
		return
	}
	log.Printf("heartbeat email sent to %s", s.Recipient)
	store.LogSend("ok", "")
}

// sendAttempts is the total number of send tries (1 initial + retries) before
// a heartbeat is recorded as failed.
const sendAttempts = 3

// sendWithRetry sends msg, retrying transient failures with exponential backoff
// (2s, 4s, ...). It returns the last error after exhausting attempts, or early
// if ctx is cancelled during a backoff.
func sendWithRetry(ctx context.Context, cfg mailer.Config, msg mailer.Message, attempts int) error {
	var err error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			backoff := time.Duration(1<<(i-1)) * 2 * time.Second
			log.Printf("email send attempt %d/%d failed, retrying in %v: %v", i, attempts, backoff, err)
			if !sleep(ctx, backoff) {
				return err // shutting down
			}
		}
		if err = cfg.SendMessage(msg); err == nil {
			return nil
		}
	}
	return err
}

// mailInterval clamps the configured interval to a sane minimum.
func mailInterval(mins int) time.Duration {
	if mins < 1 {
		mins = 10
	}
	return time.Duration(mins) * time.Minute
}

// sleep waits for d or until ctx is cancelled, returning false if it was
// cancelled (the caller should stop).
func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	switch strings.ToLower(os.Getenv(key)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
