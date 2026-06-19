package db

import (
	"database/sql"
	"time"
)

// Newsletter is a scheduled or sent email newsletter.
type Newsletter struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Subject     string     `json:"subject"`
	BodyHTML    string     `json:"bodyHtml"`
	BodyText    string     `json:"bodyText"`
	Recipient   string     `json:"recipient"`
	Status      string     `json:"status"`      // draft | scheduled | sent | failed
	ScheduledAt *time.Time `json:"scheduledAt"` // nil = no schedule
	SentAt      *time.Time `json:"sentAt"`
	Error       string     `json:"error"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// scanNewsletter scans a single newsletter row into a Newsletter struct,
// handling nullable columns (scheduled_at, sent_at, error) via sql.Null types.
func scanNewsletter(row interface{ Scan(...any) error }) (Newsletter, error) {
	var n Newsletter
	var scheduledAt sql.NullTime
	var sentAt sql.NullTime
	var errMsg sql.NullString

	err := row.Scan(
		&n.ID, &n.Title, &n.Subject, &n.BodyHTML, &n.BodyText,
		&n.Recipient, &n.Status, &scheduledAt, &sentAt, &errMsg,
		&n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return n, err
	}
	if scheduledAt.Valid {
		n.ScheduledAt = &scheduledAt.Time
	}
	if sentAt.Valid {
		n.SentAt = &sentAt.Time
	}
	if errMsg.Valid {
		n.Error = errMsg.String
	}
	return n, nil
}

// ListNewsletters returns all newsletters ordered by created_at descending.
func (s *Store) ListNewsletters() ([]Newsletter, error) {
	rows, err := s.db.Query(
		`SELECT id, title, subject, body_html, COALESCE(body_text, ''), recipient, status, scheduled_at, sent_at, error, created_at, updated_at
		 FROM newsletters ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Newsletter
	for rows.Next() {
		n, err := scanNewsletter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// GetNewsletter returns a single newsletter by ID.
func (s *Store) GetNewsletter(id int) (Newsletter, error) {
	row := s.db.QueryRow(
		`SELECT id, title, subject, body_html, COALESCE(body_text, ''), recipient, status, scheduled_at, sent_at, error, created_at, updated_at
		 FROM newsletters WHERE id = ?`, id,
	)
	return scanNewsletter(row)
}

// CreateNewsletter inserts a new newsletter and returns its auto-generated ID.
// If scheduled_at is set and status is empty, status defaults to "scheduled".
// If no scheduled_at, status defaults to "draft".
func (s *Store) CreateNewsletter(n Newsletter) (int, error) {
	if n.Status == "" {
		if n.ScheduledAt != nil {
			n.Status = "scheduled"
		} else {
			n.Status = "draft"
		}
	}
	res, err := s.db.Exec(
		`INSERT INTO newsletters (title, subject, body_html, body_text, recipient, status, scheduled_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		n.Title, n.Subject, n.BodyHTML, nullable(n.BodyText), n.Recipient, n.Status, n.ScheduledAt,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// UpdateNewsletter updates an existing newsletter by ID.
// Same status-defaulting logic as CreateNewsletter.
func (s *Store) UpdateNewsletter(n Newsletter) error {
	if n.Status == "" {
		if n.ScheduledAt != nil {
			n.Status = "scheduled"
		} else {
			n.Status = "draft"
		}
	}
	_, err := s.db.Exec(
		`UPDATE newsletters SET title = ?, subject = ?, body_html = ?, body_text = ?, recipient = ?, status = ?, scheduled_at = ?
		 WHERE id = ?`,
		n.Title, n.Subject, n.BodyHTML, nullable(n.BodyText), n.Recipient, n.Status, n.ScheduledAt, n.ID,
	)
	return err
}

// DeleteNewsletter removes a newsletter by ID.
func (s *Store) DeleteNewsletter(id int) error {
	_, err := s.db.Exec(`DELETE FROM newsletters WHERE id = ?`, id)
	return err
}

// DueNewsletters returns newsletters that are scheduled and whose scheduled_at
// time has arrived (i.e. status = 'scheduled' AND scheduled_at <= NOW()).
func (s *Store) DueNewsletters() ([]Newsletter, error) {
	rows, err := s.db.Query(
		`SELECT id, title, subject, body_html, COALESCE(body_text, ''), recipient, status, scheduled_at, sent_at, error, created_at, updated_at
		 FROM newsletters WHERE status = 'scheduled' AND scheduled_at <= NOW()`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Newsletter
	for rows.Next() {
		n, err := scanNewsletter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// MarkSent updates a newsletter's status to "sent" and records the current time.
func (s *Store) MarkSent(id int) error {
	_, err := s.db.Exec(
		`UPDATE newsletters SET status = 'sent', sent_at = NOW() WHERE id = ?`, id,
	)
	return err
}

// MarkFailed updates a newsletter's status to "failed" and records the error.
func (s *Store) MarkFailed(id int, errMsg string) error {
	_, err := s.db.Exec(
		`UPDATE newsletters SET status = 'failed', error = ? WHERE id = ?`, errMsg, id,
	)
	return err
}
