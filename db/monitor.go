package db

import "time"

// Threshold is an alert rule on a sampled metric.
type Threshold struct {
	ID      int     `json:"id"`
	Metric  string  `json:"metric"` // cpu | mem | disk | load1
	Op      string  `json:"op"`     // gt | gte | lt | lte
	Value   float64 `json:"value"`
	Enabled bool    `json:"enabled"`
}

// Target is an external endpoint to health-check.
type Target struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`    // http | tcp
	Address string `json:"address"` // URL (http) or host:port (tcp)
	Enabled bool   `json:"enabled"`
}

// MetricPoint is one stored sample for dashboard history.
type MetricPoint struct {
	TS    time.Time `json:"ts"`
	CPU   float64   `json:"cpu"`
	Mem   float64   `json:"mem"`
	Disk  float64   `json:"disk"`
	Load1 float64   `json:"load1"`
}

// Alert is one row of alert_log.
type Alert struct {
	TS      time.Time `json:"ts"`
	Source  string    `json:"source"`
	State   string    `json:"state"` // breach | recovered
	Message string    `json:"message"`
}

// ---- Thresholds ----

func (s *Store) ListThresholds(activeOnly bool) ([]Threshold, error) {
	q := `SELECT id, metric, op, value, enabled FROM thresholds`
	if activeOnly {
		q += ` WHERE enabled = TRUE`
	}
	q += ` ORDER BY id`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Threshold
	for rows.Next() {
		var t Threshold
		if err := rows.Scan(&t.ID, &t.Metric, &t.Op, &t.Value, &t.Enabled); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CreateThreshold(t Threshold) (int, error) {
	res, err := s.db.Exec(
		`INSERT INTO thresholds (metric, op, value, enabled) VALUES (?, ?, ?, ?)`,
		t.Metric, t.Op, t.Value, t.Enabled,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) UpdateThreshold(t Threshold) error {
	_, err := s.db.Exec(
		`UPDATE thresholds SET metric = ?, op = ?, value = ?, enabled = ? WHERE id = ?`,
		t.Metric, t.Op, t.Value, t.Enabled, t.ID,
	)
	return err
}

func (s *Store) DeleteThreshold(id int) error {
	_, err := s.db.Exec(`DELETE FROM thresholds WHERE id = ?`, id)
	return err
}

// ---- Targets ----

func (s *Store) ListTargets(activeOnly bool) ([]Target, error) {
	q := `SELECT id, name, kind, address, enabled FROM targets`
	if activeOnly {
		q += ` WHERE enabled = TRUE`
	}
	q += ` ORDER BY id`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Target
	for rows.Next() {
		var t Target
		if err := rows.Scan(&t.ID, &t.Name, &t.Kind, &t.Address, &t.Enabled); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CreateTarget(t Target) (int, error) {
	res, err := s.db.Exec(
		`INSERT INTO targets (name, kind, address, enabled) VALUES (?, ?, ?, ?)`,
		t.Name, t.Kind, t.Address, t.Enabled,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) UpdateTarget(t Target) error {
	_, err := s.db.Exec(
		`UPDATE targets SET name = ?, kind = ?, address = ?, enabled = ? WHERE id = ?`,
		t.Name, t.Kind, t.Address, t.Enabled, t.ID,
	)
	return err
}

func (s *Store) DeleteTarget(id int) error {
	_, err := s.db.Exec(`DELETE FROM targets WHERE id = ?`, id)
	return err
}

// ---- Metric history ----

// InsertMetric stores one sample. Percentages are 0..100; pass -1 when unknown.
func (s *Store) InsertMetric(p MetricPoint) error {
	_, err := s.db.Exec(
		`INSERT INTO metric_history (ts, cpu, mem, disk, load1) VALUES (?, ?, ?, ?, ?)`,
		p.TS, p.CPU, p.Mem, p.Disk, p.Load1,
	)
	return err
}

// MetricHistory returns up to limit most-recent samples, oldest-first (chart order).
func (s *Store) MetricHistory(limit int) ([]MetricPoint, error) {
	rows, err := s.db.Query(
		`SELECT ts, cpu, mem, disk, load1 FROM (
		   SELECT ts, cpu, mem, disk, load1 FROM metric_history ORDER BY id DESC LIMIT ?
		 ) recent ORDER BY ts ASC`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MetricPoint
	for rows.Next() {
		var p MetricPoint
		if err := rows.Scan(&p.TS, &p.CPU, &p.Mem, &p.Disk, &p.Load1); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// PruneMetrics deletes samples older than the cutoff.
func (s *Store) PruneMetrics(olderThan time.Time) error {
	_, err := s.db.Exec(`DELETE FROM metric_history WHERE ts < ?`, olderThan)
	return err
}

// ---- Alerts ----

func (s *Store) InsertAlert(a Alert) error {
	_, err := s.db.Exec(
		`INSERT INTO alert_log (ts, source, state, message) VALUES (?, ?, ?, ?)`,
		a.TS, a.Source, a.State, a.Message,
	)
	return err
}

func (s *Store) RecentAlerts(limit int) ([]Alert, error) {
	rows, err := s.db.Query(
		`SELECT ts, source, state, message FROM alert_log ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.TS, &a.Source, &a.State, &a.Message); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
