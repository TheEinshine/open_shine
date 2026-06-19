package newsletter

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/TheEinshine/open_shine/db"
	"github.com/TheEinshine/open_shine/mailer"
)

// Loop polls for due newsletters every tick and sends them.
// It runs until ctx is cancelled.
func Loop(ctx context.Context, store *db.Store, smtp mailer.Config) {
	log.Println("newsletter loop started")
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// Check immediately on startup, then every tick.
	processDue(store, smtp)
	for {
		select {
		case <-ctx.Done():
			log.Println("newsletter loop stopped")
			return
		case <-ticker.C:
			processDue(store, smtp)
		}
	}
}

func processDue(store *db.Store, smtp mailer.Config) {
	newsletters, err := store.DueNewsletters()
	if err != nil {
		log.Printf("newsletter: could not query due newsletters: %v", err)
		return
	}
	if len(newsletters) == 0 {
		return
	}

	subscribers, err := store.ActiveSubscribers()
	if err != nil {
		log.Printf("newsletter: could not query active subscribers: %v", err)
		return
	}

	for _, nl := range newsletters {
		if len(subscribers) == 0 {
			log.Printf("newsletter: skipped #%d %q (no active subscribers)", nl.ID, nl.Title)
			store.MarkSent(nl.ID)
			continue
		}
		var errs []string
		for _, sub := range subscribers {
			msg := mailer.Message{
				To:      sub.Email,
				Subject: nl.Subject,
				HTML:    WrapHTML(nl.BodyHTML),
				Text:    nl.BodyText,
			}
			if err := smtp.SendMessage(msg); err != nil {
				log.Printf("newsletter: failed to send #%d to %s: %v", nl.ID, sub.Email, err)
				errs = append(errs, err.Error())
			} else {
				log.Printf("newsletter: sent #%d %q to %s", nl.ID, nl.Title, sub.Email)
			}
		}
		
		if len(errs) > 0 {
			store.MarkFailed(nl.ID, strings.Join(errs, "; "))
		} else {
			store.MarkSent(nl.ID)
		}
	}
}

// SendNow immediately sends a single newsletter by ID to all active subscribers.
func SendNow(store *db.Store, smtp mailer.Config, id int) error {
	nl, err := store.GetNewsletter(id)
	if err != nil {
		return err
	}
	
	subscribers, err := store.ActiveSubscribers()
	if err != nil {
		return err
	}
	
	if len(subscribers) == 0 {
		store.MarkSent(id)
		return nil
	}

	var errs []string
	for _, sub := range subscribers {
		msg := mailer.Message{
			To:      sub.Email,
			Subject: nl.Subject,
			HTML:    nl.BodyHTML,
			Text:    nl.BodyText,
		}
		if err := smtp.SendMessage(msg); err != nil {
			errs = append(errs, err.Error())
		}
	}
	
	if len(errs) > 0 {
		store.MarkFailed(id, strings.Join(errs, "; "))
		return nil // We still return nil for the HTTP response so the UI doesn't crash on partial failure
	}
	store.MarkSent(id)
	return nil
}
