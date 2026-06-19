package newsletter

import (
	"context"
	"log"
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
	for _, nl := range newsletters {
		msg := mailer.Message{
			To:      nl.Recipient,
			Subject: nl.Subject,
			HTML:    nl.BodyHTML,
			Text:    nl.BodyText,
		}
		if err := smtp.SendMessage(msg); err != nil {
			log.Printf("newsletter: failed to send #%d %q: %v", nl.ID, nl.Title, err)
			store.MarkFailed(nl.ID, err.Error())
			continue
		}
		log.Printf("newsletter: sent #%d %q to %s", nl.ID, nl.Title, nl.Recipient)
		store.MarkSent(nl.ID)
	}
}

// SendNow immediately sends a single newsletter by ID, regardless of its schedule.
func SendNow(store *db.Store, smtp mailer.Config, id int) error {
	nl, err := store.GetNewsletter(id)
	if err != nil {
		return err
	}
	msg := mailer.Message{
		To:      nl.Recipient,
		Subject: nl.Subject,
		HTML:    nl.BodyHTML,
		Text:    nl.BodyText,
	}
	if err := smtp.SendMessage(msg); err != nil {
		store.MarkFailed(id, err.Error())
		return err
	}
	store.MarkSent(id)
	return nil
}
