package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Settings is the single configurable row in mail_settings (id = 1).
type Settings struct {
	Recipient    string
	IntervalMins int
	Subject      string
	Enabled      bool
}

type Store struct {
	db *sql.DB
}

// Open connects to MySQL using DB_* environment variables.
func Open() (*Store, error) {
	host := envOr("DB_HOST", "127.0.0.1")
	port := envOr("DB_PORT", "3306")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	name := os.Getenv("DB_NAME")
	if user == "" || name == "" {
		return nil, fmt.Errorf("missing DB_USER or DB_NAME")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, name)
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	conn.SetConnMaxLifetime(3 * time.Minute)
	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(2)
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("cannot reach mysql: %w", err)
	}
	return &Store{db: conn}, nil
}

// EnsureSchema creates the tables if they don't exist and seeds a default
// settings row. defaultRecipient is used only on first run (when no row
// exists yet) so the heartbeat works out of the box; change it later with SQL.
func (s *Store) EnsureSchema(defaultRecipient string) error {
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS mail_settings (
  id            INT PRIMARY KEY,
  recipient     VARCHAR(255) NOT NULL,
  interval_mins INT NOT NULL DEFAULT 10,
  subject       VARCHAR(255) NOT NULL DEFAULT 'Open Shine heartbeat',
  enabled       BOOLEAN NOT NULL DEFAULT TRUE
)`); err != nil {
		return err
	}

	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS mail_log (
  id      INT AUTO_INCREMENT PRIMARY KEY,
  sent_at DATETIME NOT NULL,
  status  VARCHAR(20) NOT NULL,
  error   TEXT
)`); err != nil {
		return err
	}

	// Only inserts if id=1 doesn't already exist, so it never clobbers
	// settings you've changed.
	_, err := s.db.Exec(
		`INSERT IGNORE INTO mail_settings (id, recipient) VALUES (1, ?)`,
		defaultRecipient,
	)
	return err
}

// GetSettings reads the single settings row.
func (s *Store) GetSettings() (Settings, error) {
	var out Settings
	err := s.db.QueryRow(
		`SELECT recipient, interval_mins, subject, enabled FROM mail_settings WHERE id = 1`,
	).Scan(&out.Recipient, &out.IntervalMins, &out.Subject, &out.Enabled)
	return out, err
}

// LogSend records the outcome of a send attempt.
func (s *Store) LogSend(status, errMsg string) error {
	_, err := s.db.Exec(
		`INSERT INTO mail_log (sent_at, status, error) VALUES (?, ?, ?)`,
		time.Now(), status, nullable(errMsg),
	)
	return err
}

func nullable(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}