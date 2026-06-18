package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/TheEinshine/open_shine/db"
	"github.com/TheEinshine/open_shine/mailer"
)

func main() {
	go startMailLoop()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Shine's Service v4 is running")
	})

	fmt.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func startMailLoop() {
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

	// Build the structure first, then seed default rows.
	if err := store.Migrate(); err != nil {
		log.Printf("mailer disabled: migrate failed: %v", err)
		return
	}
	if err := store.Seed(smtpCfg.User); err != nil {
		log.Printf("mailer disabled: seed failed: %v", err)
		return
	}

	log.Println("mail loop started")

	for {
		s, err := store.GetSettings()
		if err != nil {
			log.Printf("could not read mail_settings: %v", err)
			time.Sleep(10 * time.Minute)
			continue
		}

		interval := s.IntervalMins
		if interval < 1 {
			interval = 10
		}

		if s.Enabled && s.Recipient != "" {
			body := fmt.Sprintf("Open Shine is still running at %s", time.Now().Format(time.RFC1123))
			if sendErr := smtpCfg.Send(s.Recipient, s.Subject, body); sendErr != nil {
				log.Printf("email send failed: %v", sendErr)
				store.LogSend("error", sendErr.Error())
			} else {
				log.Printf("heartbeat email sent to %s", s.Recipient)
				store.LogSend("ok", "")
			}
		}

		time.Sleep(time.Duration(interval) * time.Minute)
	}
}