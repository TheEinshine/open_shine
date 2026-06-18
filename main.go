package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/TheEinshine/open_shine/db"
	"github.com/TheEinshine/open_shine/mailer"
	"github.com/TheEinshine/open_shine/report"
	"github.com/TheEinshine/open_shine/sysstat"
)

func main() {
	// ctx is cancelled on the first SIGINT/SIGTERM, which both the mail loop
	// and the HTTP server watch so shutdown is coordinated and clean.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startMailLoop(ctx)
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Shine's Service v4 is running")
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
		// Bound every phase of a request so a slow or stalled client can't
		// pin a connection open indefinitely (slow-loris protection).
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Drain in-flight requests when a shutdown signal arrives.
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

// startMailLoop wires up the mailer and runs the heartbeat loop until ctx is
// cancelled. Any setup failure logs the reason and disables the loop without
// taking down the HTTP server.
func startMailLoop(ctx context.Context) {
	smtpCfg, err := mailer.LoadConfig()
	if err != nil {
		log.Printf("mailer disabled: %v", err)
		return
	}

	store, err := db.Open()
	if err != nil {
		log.Printf("mailer disabled: %v", err)
		return
	}
	defer store.Close()

	// Build the structure first, then seed default rows.
	if err := store.Migrate(); err != nil {
		log.Printf("mailer disabled: migrate failed: %v", err)
		return
	}
	if err := store.Seed(smtpCfg.User); err != nil {
		log.Printf("mailer disabled: seed failed: %v", err)
		return
	}

	runMailLoop(ctx, store, smtpCfg)
}

// runMailLoop sends a heartbeat email every interval until ctx is cancelled.
// The wait between cycles is interruptible, so a shutdown signal stops the loop
// immediately instead of blocking for up to a full interval.
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
				sendHeartbeat(store, smtpCfg, s)
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
func sendHeartbeat(store *db.Store, smtpCfg mailer.Config, s db.Settings) {
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
		To:      s.Recipient,
		Subject: subject,
		Text:    report.RenderText(stats, logs),
		HTML:    report.RenderHTML(stats, logs),
	}

	if err := smtpCfg.SendMessage(msg); err != nil {
		log.Printf("email send failed: %v", err)
		store.LogSend("error", err.Error())
		return
	}
	log.Printf("heartbeat email sent to %s", s.Recipient)
	store.LogSend("ok", "")
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