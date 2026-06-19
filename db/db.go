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
	Recipient    string `json:"recipient"`
	IntervalMins int    `json:"intervalMins"`
	Subject      string `json:"subject"`
	Enabled      bool   `json:"enabled"`
}

// LogEntry is one row of mail_log, newest first when returned by RecentLogs.
type LogEntry struct {
	SentAt time.Time `json:"sentAt"`
	Status string    `json:"status"`
	Error  string    `json:"error"`
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

// migrations is the ordered list of structural statements. Each must be
// idempotent (CREATE TABLE IF NOT EXISTS / ALTER ... IF NOT EXISTS) so running
// Migrate repeatedly on every startup is always safe. Add new tables and
// columns here as the app grows.
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
  id            INT AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(255) NOT NULL,
  email         VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NULL,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)`,
	`CREATE TABLE IF NOT EXISTS mail_settings (
  id            INT PRIMARY KEY,
  recipient     VARCHAR(255) NOT NULL,
  interval_mins INT NOT NULL DEFAULT 10,
  subject       VARCHAR(255) NOT NULL DEFAULT 'Open Shine heartbeat',
  enabled       BOOLEAN NOT NULL DEFAULT TRUE
)`,
	`CREATE TABLE IF NOT EXISTS mail_log (
  id      INT AUTO_INCREMENT PRIMARY KEY,
  sent_at DATETIME NOT NULL,
  status  VARCHAR(20) NOT NULL,
  error   TEXT
)`,
	`CREATE TABLE IF NOT EXISTS sessions (
  id         CHAR(64) PRIMARY KEY,
  user_id    INT NOT NULL,
  csrf_token CHAR(64) NOT NULL,
  created_at DATETIME NOT NULL,
  expires_at DATETIME NOT NULL,
  INDEX idx_sessions_expires (expires_at),
  CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
)`,
	`CREATE TABLE IF NOT EXISTS thresholds (
  id         INT AUTO_INCREMENT PRIMARY KEY,
  metric     VARCHAR(20) NOT NULL,          -- cpu | mem | disk | load1
  op         VARCHAR(4)  NOT NULL,          -- gt | gte | lt | lte
  value      DOUBLE      NOT NULL,
  enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
  created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP
)`,
	`CREATE TABLE IF NOT EXISTS targets (
  id         INT AUTO_INCREMENT PRIMARY KEY,
  name       VARCHAR(100) NOT NULL,
  kind       VARCHAR(10)  NOT NULL,         -- http | tcp
  address    VARCHAR(255) NOT NULL,         -- URL for http, host:port for tcp
  enabled    BOOLEAN      NOT NULL DEFAULT TRUE,
  created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
)`,
	`CREATE TABLE IF NOT EXISTS metric_history (
  id      INT AUTO_INCREMENT PRIMARY KEY,
  ts      DATETIME NOT NULL,
  cpu     DOUBLE   NOT NULL,
  mem     DOUBLE   NOT NULL,
  disk    DOUBLE   NOT NULL,
  load1   DOUBLE   NOT NULL,
  INDEX idx_metric_history_ts (ts)
)`,
	`CREATE TABLE IF NOT EXISTS alert_log (
  id       INT AUTO_INCREMENT PRIMARY KEY,
  ts       DATETIME NOT NULL,
  source   VARCHAR(100) NOT NULL,           -- e.g. threshold:cpu, target:api
  state    VARCHAR(20)  NOT NULL,           -- breach | recovered
  message  TEXT NOT NULL,
  INDEX idx_alert_log_ts (ts)
)`,
	`CREATE TABLE IF NOT EXISTS newsletters (
  id           INT AUTO_INCREMENT PRIMARY KEY,
  title        VARCHAR(255) NOT NULL,
  subject      VARCHAR(255) NOT NULL,
  body_html    LONGTEXT NOT NULL,
  body_text    TEXT,
  recipient    VARCHAR(255) NOT NULL,
  status       VARCHAR(20) NOT NULL DEFAULT 'draft',
  scheduled_at DATETIME NULL,
  sent_at      DATETIME NULL,
  error        TEXT,
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_newsletters_status_scheduled (status, scheduled_at)
)`,
}

// Close releases the underlying connection pool.
func (s *Store) Close() error {
	return s.db.Close()
}

// Migrate creates the schema. Safe to run on every boot.
func (s *Store) Migrate() error {
	for i, stmt := range migrations {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}
	return nil
}

// Seed inserts default rows that the app needs to function. Each insert uses
// INSERT IGNORE so it only writes when the row is absent — it never overwrites
// values you've changed by hand.
func (s *Store) Seed(defaultRecipient string) error {
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

// UpdateSettings overwrites the single mail_settings row (id = 1).
func (s *Store) UpdateSettings(in Settings) error {
	_, err := s.db.Exec(
		`UPDATE mail_settings SET recipient = ?, interval_mins = ?, subject = ?, enabled = ? WHERE id = 1`,
		in.Recipient, in.IntervalMins, in.Subject, in.Enabled,
	)
	return err
}

// LogSend records the outcome of a send attempt.
func (s *Store) LogSend(status, errMsg string) error {
	_, err := s.db.Exec(
		`INSERT INTO mail_log (sent_at, status, error) VALUES (?, ?, ?)`,
		time.Now(), status, nullable(errMsg),
	)
	return err
}

// RecentLogs returns up to limit most-recent mail_log rows, newest first.
func (s *Store) RecentLogs(limit int) ([]LogEntry, error) {
	rows, err := s.db.Query(
		`SELECT sent_at, status, COALESCE(error, '') FROM mail_log ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.SentAt, &e.Status, &e.Error); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
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