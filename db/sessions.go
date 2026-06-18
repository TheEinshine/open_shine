package db

import "time"

// Session is a server-side login session, keyed by an opaque random id stored
// in the user's cookie. CSRFToken is the synchronizer token for this session.
type Session struct {
	ID        string
	UserID    int
	CSRFToken string
	ExpiresAt time.Time
}

// CreateSession stores a new session valid for ttl from now.
func (s *Store) CreateSession(id string, userID int, csrf string, ttl time.Duration) error {
	now := time.Now()
	_, err := s.db.Exec(
		`INSERT INTO sessions (id, user_id, csrf_token, created_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
		id, userID, csrf, now, now.Add(ttl),
	)
	return err
}

// SessionByID returns a non-expired session, or sql.ErrNoRows if missing/expired.
func (s *Store) SessionByID(id string) (Session, error) {
	var sess Session
	err := s.db.QueryRow(
		`SELECT id, user_id, csrf_token, expires_at FROM sessions WHERE id = ? AND expires_at > NOW()`,
		id,
	).Scan(&sess.ID, &sess.UserID, &sess.CSRFToken, &sess.ExpiresAt)
	return sess, err
}

// DeleteSession removes a session (logout).
func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteExpiredSessions prunes lapsed sessions.
func (s *Store) DeleteExpiredSessions() error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at <= NOW()`)
	return err
}
