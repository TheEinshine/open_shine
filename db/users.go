package db

import (
	"database/sql"
	"time"
)

// User is an admin account. PasswordHash is only populated by UserByEmail (for
// login verification) and is never sent to clients.
type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// CountUsers returns how many accounts exist (used to decide first-boot seeding).
func (s *Store) CountUsers() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// CreateUser inserts an account with an already-hashed password and returns its id.
func (s *Store) CreateUser(name, email, passwordHash string) (int, error) {
	res, err := s.db.Exec(
		`INSERT INTO users (name, email, password_hash) VALUES (?, ?, ?)`,
		name, email, passwordHash,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// SetPassword updates an account's password hash by email.
func (s *Store) SetPassword(email, passwordHash string) error {
	_, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE email = ?`, passwordHash, email)
	return err
}

// UserByEmail looks up an account by email, including the password hash.
func (s *Store) UserByEmail(email string) (User, error) {
	return s.scanUser(s.db.QueryRow(
		`SELECT id, name, email, COALESCE(password_hash, ''), created_at FROM users WHERE email = ?`,
		email,
	))
}

// UserByID looks up an account by id (password hash included but unused).
func (s *Store) UserByID(id int) (User, error) {
	return s.scanUser(s.db.QueryRow(
		`SELECT id, name, email, COALESCE(password_hash, ''), created_at FROM users WHERE id = ?`,
		id,
	))
}

func (s *Store) scanUser(row *sql.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	return u, err
}
