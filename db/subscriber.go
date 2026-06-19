package db

import "time"

type Subscriber struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
}

// AddSubscriber adds a new subscriber or reactivates an existing one.
func (s *Store) AddSubscriber(email string) error {
	_, err := s.db.Exec(`
		INSERT INTO subscribers (email, active, created_at) 
		VALUES (?, TRUE, NOW())
		ON DUPLICATE KEY UPDATE active = TRUE
	`, email)
	return err
}

// ListSubscribers returns all subscribers.
func (s *Store) ListSubscribers() ([]Subscriber, error) {
	rows, err := s.db.Query(`SELECT id, email, active, created_at FROM subscribers ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Subscriber
	for rows.Next() {
		var sub Subscriber
		if err := rows.Scan(&sub.ID, &sub.Email, &sub.Active, &sub.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

// DeleteSubscriber removes a subscriber.
func (s *Store) DeleteSubscriber(id int) error {
	_, err := s.db.Exec(`DELETE FROM subscribers WHERE id = ?`, id)
	return err
}

// ActiveSubscribers returns all active subscribers.
func (s *Store) ActiveSubscribers() ([]Subscriber, error) {
	rows, err := s.db.Query(`SELECT id, email, active, created_at FROM subscribers WHERE active = TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Subscriber
	for rows.Next() {
		var sub Subscriber
		if err := rows.Scan(&sub.ID, &sub.Email, &sub.Active, &sub.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}
