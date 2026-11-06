// Package store persists monitors, check history and notifications in SQLite.
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"lucy/internal/alert"
	"lucy/internal/model"
)

// BucketSeconds is the width of a history bucket.
const BucketSeconds = 1800

var ErrNotFound = errors.New("not found")

var migrations = []string{`
CREATE TABLE monitors (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	url TEXT NOT NULL,
	method TEXT NOT NULL,
	headers TEXT NOT NULL,
	body TEXT NOT NULL,
	timeout_sec INTEGER NOT NULL,
	interval_sec INTEGER NOT NULL,
	alert_threshold INTEGER NOT NULL,
	conditions TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	phase TEXT NOT NULL DEFAULT 'new',
	fail_streak INTEGER NOT NULL DEFAULT 0,
	ok_streak INTEGER NOT NULL DEFAULT 0,
	fail_since INTEGER NOT NULL DEFAULT 0,
	since INTEGER NOT NULL DEFAULT 0,
	last_check INTEGER NOT NULL DEFAULT 0,
	last_outcome TEXT NOT NULL DEFAULT '',
	response_ms INTEGER,
	failure TEXT NOT NULL DEFAULT ''
);
CREATE TABLE checks (
	id INTEGER PRIMARY KEY,
	monitor_id TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
	at INTEGER NOT NULL,
	outcome TEXT NOT NULL,
	response_ms INTEGER,
	failure TEXT NOT NULL
);
CREATE INDEX checks_monitor_at ON checks(monitor_id, at);
CREATE INDEX checks_at ON checks(at);
CREATE TABLE check_rollups (
	monitor_id TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
	bucket INTEGER NOT NULL,
	ok INTEGER NOT NULL DEFAULT 0,
	fail INTEGER NOT NULL DEFAULT 0,
	unknown INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (monitor_id, bucket)
) WITHOUT ROWID;
CREATE TABLE incidents (
	id INTEGER PRIMARY KEY,
	monitor_id TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
	started_at INTEGER NOT NULL,
	ended_at INTEGER,
	cause TEXT NOT NULL,
	notified_at INTEGER
);
CREATE INDEX incidents_monitor ON incidents(monitor_id, started_at);
CREATE TABLE notifications (
	id INTEGER PRIMARY KEY,
	monitor_id TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
	incident_id INTEGER REFERENCES incidents(id) ON DELETE CASCADE,
	title TEXT NOT NULL,
	message TEXT NOT NULL,
	priority INTEGER NOT NULL,
	tags TEXT NOT NULL,
	click TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	sent_at INTEGER,
	attempts INTEGER NOT NULL DEFAULT 0,
	next_attempt INTEGER NOT NULL
);
CREATE INDEX notifications_due ON notifications(sent_at, next_attempt);
`, `
CREATE TABLE settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
CREATE TABLE sessions (
	token_hash TEXT PRIMARY KEY,
	created_at INTEGER NOT NULL,
	expires_at INTEGER NOT NULL
);
`, `
ALTER TABLE monitors ADD COLUMN ssl_expires INTEGER NOT NULL DEFAULT 0;
ALTER TABLE monitors ADD COLUMN ssl_issuer TEXT NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN ssl_stage INTEGER NOT NULL DEFAULT 0;
ALTER TABLE monitors ADD COLUMN domain_expires INTEGER NOT NULL DEFAULT 0;
ALTER TABLE monitors ADD COLUMN domain_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN domain_checked INTEGER NOT NULL DEFAULT 0;
ALTER TABLE monitors ADD COLUMN domain_stage INTEGER NOT NULL DEFAULT 0;
`}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// One connection serializes writes and removes SQLITE_BUSY entirely.
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	var version int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	for i := version; i < len(migrations); i++ {
		if _, err := s.db.Exec(migrations[i]); err != nil {
			return err
		}
		if _, err := s.db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, i+1)); err != nil {
			return err
		}
	}
	return nil
}

// Monitor is a configuration plus its live state.
type Monitor struct {
	model.Config
	State       alert.State
	Since       time.Time
	LastCheck   time.Time
	LastOutcome alert.Outcome
	ResponseMs  *int
	Failure     string

	SSLExpires    time.Time
	SSLIssuer     string
	SSLStage      int
	DomainExpires time.Time
	DomainReason  string
	DomainChecked time.Time
	DomainStage   int
}

func newID() string {
	data := make([]byte, 8)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	return hex.EncodeToString(data)
}

func millis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func fromMillis(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

func (s *Store) CreateMonitor(ctx context.Context, cfg model.Config) (model.Config, error) {
	cfg.ID = newID()
	headers, conditions, err := encode(cfg)
	if err != nil {
		return cfg, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO monitors (id, name, url, method, headers, body, timeout_sec, interval_sec,
			alert_threshold, conditions, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cfg.ID, cfg.Name, cfg.URL, cfg.Method, headers, cfg.Body, cfg.TimeoutSec, cfg.IntervalSec,
		cfg.AlertThreshold, conditions, time.Now().UnixMilli())
	return cfg, err
}

func (s *Store) UpdateMonitor(ctx context.Context, cfg model.Config) error {
	headers, conditions, err := encode(cfg)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// A new address means another certificate and possibly another domain.
	if _, err := tx.ExecContext(ctx, `
		UPDATE monitors SET ssl_expires = 0, ssl_issuer = '', ssl_stage = 0, domain_expires = 0,
			domain_reason = '', domain_checked = 0, domain_stage = 0
		WHERE id = ? AND url != ?`, cfg.ID, cfg.URL); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE monitors SET name = ?, url = ?, method = ?, headers = ?, body = ?, timeout_sec = ?,
			interval_sec = ?, alert_threshold = ?, conditions = ?
		WHERE id = ?`,
		cfg.Name, cfg.URL, cfg.Method, headers, cfg.Body, cfg.TimeoutSec, cfg.IntervalSec,
		cfg.AlertThreshold, conditions, cfg.ID)
	if err != nil {
		return err
	}
	if err := mustAffect(result); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteMonitor(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM monitors WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return mustAffect(result)
}

func mustAffect(result sql.Result) error {
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func encode(cfg model.Config) (string, string, error) {
	headers, err := json.Marshal(cfg.Headers)
	if err != nil {
		return "", "", err
	}
	conditions, err := json.Marshal(cfg.Conditions)
	return string(headers), string(conditions), err
}

const monitorColumns = `id, name, url, method, headers, body, timeout_sec, interval_sec, alert_threshold,
	conditions, phase, fail_streak, ok_streak, fail_since, since, last_check, last_outcome, response_ms, failure,
	ssl_expires, ssl_issuer, ssl_stage, domain_expires, domain_reason, domain_checked, domain_stage`

func scanMonitor(row interface{ Scan(...any) error }) (Monitor, error) {
	var m Monitor
	var headers, conditions, phase, outcome string
	var failSince, since, lastCheck, sslExpires, domainExpires, domainChecked int64
	var responseMs sql.NullInt64
	err := row.Scan(&m.ID, &m.Name, &m.URL, &m.Method, &headers, &m.Body, &m.TimeoutSec, &m.IntervalSec,
		&m.AlertThreshold, &conditions, &phase, &m.State.FailStreak, &m.State.OkStreak, &failSince, &since,
		&lastCheck, &outcome, &responseMs, &m.Failure,
		&sslExpires, &m.SSLIssuer, &m.SSLStage, &domainExpires, &m.DomainReason, &domainChecked, &m.DomainStage)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal([]byte(headers), &m.Headers); err != nil {
		return m, err
	}
	if err := json.Unmarshal([]byte(conditions), &m.Conditions); err != nil {
		return m, err
	}
	m.State.Phase = alert.Phase(phase)
	m.State.FailSince = fromMillis(failSince)
	m.Since = fromMillis(since)
	m.LastCheck = fromMillis(lastCheck)
	m.LastOutcome = alert.Outcome(outcome)
	m.SSLExpires = fromMillis(sslExpires)
	m.DomainExpires = fromMillis(domainExpires)
	m.DomainChecked = fromMillis(domainChecked)
	if responseMs.Valid {
		value := int(responseMs.Int64)
		m.ResponseMs = &value
	}
	return m, nil
}

func (s *Store) GetMonitor(ctx context.Context, id string) (Monitor, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+monitorColumns+` FROM monitors WHERE id = ?`, id)
	m, err := scanMonitor(row)
	if errors.Is(err, sql.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

func (s *Store) ListMonitors(ctx context.Context) ([]Monitor, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+monitorColumns+` FROM monitors ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	monitors := []Monitor{}
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

// Notification is a queued message; the worker delivers it with retries.
type Notification struct {
	ID       int64
	Title    string
	Message  string
	Priority int
	Tags     []string
	Click    string
	Attempts int
}

// Record is the full effect of one check, applied atomically.
type Record struct {
	MonitorID    string
	At           time.Time
	Outcome      alert.Outcome
	ResponseMs   *int
	Failure      string
	Next         alert.State
	Event        alert.Event
	Notification *Notification
	CertExpires  time.Time // zero keeps the known certificate
	CertIssuer   string
}

func (s *Store) Apply(ctx context.Context, r Record) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var since int64
	var phase, failure string
	err = tx.QueryRowContext(ctx, `SELECT since, phase, failure FROM monitors WHERE id = ?`, r.MonitorID).
		Scan(&since, &phase, &failure)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // deleted while the check was running
	}
	if err != nil {
		return err
	}

	at := r.At.UnixMilli()
	if _, err := tx.ExecContext(ctx, `INSERT INTO checks (monitor_id, at, outcome, response_ms, failure)
		VALUES (?, ?, ?, ?, ?)`, r.MonitorID, at, r.Outcome, r.ResponseMs, r.Failure); err != nil {
		return err
	}
	bucket := r.At.Unix() / BucketSeconds * BucketSeconds
	column := map[alert.Outcome]string{alert.OK: "ok", alert.Fail: "fail", alert.Unknown: "unknown"}[r.Outcome]
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO check_rollups (monitor_id, bucket, %[1]s) VALUES (?, ?, 1)
		ON CONFLICT (monitor_id, bucket) DO UPDATE SET %[1]s = %[1]s + 1`, column), r.MonitorID, bucket); err != nil {
		return err
	}

	var incidentID sql.NullInt64
	switch r.Event {
	case alert.Down:
		since = millis(r.Next.FailSince)
		failure = r.Failure
		result, err := tx.ExecContext(ctx, `INSERT INTO incidents (monitor_id, started_at, cause) VALUES (?, ?, ?)`,
			r.MonitorID, since, r.Failure)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		incidentID = sql.NullInt64{Int64: id, Valid: true}
	case alert.Recovered:
		since = at
		failure = ""
		if _, err := tx.ExecContext(ctx, `UPDATE incidents SET ended_at = ? WHERE monitor_id = ? AND ended_at IS NULL`,
			at, r.MonitorID); err != nil {
			return err
		}
	default:
		if alert.Phase(phase) == alert.PhaseNew && r.Next.Phase != alert.PhaseNew {
			since = at
		}
		if r.Next.Phase == alert.PhaseDown && r.Failure != "" {
			failure = r.Failure
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE monitors SET phase = ?, fail_streak = ?, ok_streak = ?, fail_since = ?,
		since = ?, last_check = ?, last_outcome = ?, response_ms = ?, failure = ? WHERE id = ?`,
		r.Next.Phase, r.Next.FailStreak, r.Next.OkStreak, millis(r.Next.FailSince), since, at, r.Outcome,
		r.ResponseMs, failure, r.MonitorID); err != nil {
		return err
	}

	if !r.CertExpires.IsZero() {
		if _, err := tx.ExecContext(ctx, `UPDATE monitors SET ssl_expires = ?, ssl_issuer = ? WHERE id = ?`,
			millis(r.CertExpires), r.CertIssuer, r.MonitorID); err != nil {
			return err
		}
	}

	if err := enqueue(ctx, tx, r.MonitorID, incidentID, r.Notification, at); err != nil {
		return err
	}
	return tx.Commit()
}

func enqueue(ctx context.Context, tx *sql.Tx, monitorID string, incidentID sql.NullInt64, n *Notification, at int64) error {
	if n == nil {
		return nil
	}
	tags, err := json.Marshal(n.Tags)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO notifications (monitor_id, incident_id, title, message, priority,
		tags, click, created_at, next_attempt) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		monitorID, incidentID, n.Title, n.Message, n.Priority, string(tags), n.Click, at, at)
	return err
}

// SaveDomain stores a settled lookup. A zero expiry with a reason means unknown.
func (s *Store) SaveDomain(ctx context.Context, id string, expires time.Time, reason string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE monitors SET domain_expires = ?, domain_reason = ?, domain_checked = ?
		WHERE id = ?`, millis(expires), reason, at.UnixMilli(), id)
	return err
}

// SaveExpiryStage moves a warning stage and queues its message in one step, so
// a crash cannot repeat or lose the warning.
func (s *Store) SaveExpiryStage(ctx context.Context, id, kind string, stage int, n *Notification, at time.Time) error {
	column := map[string]string{"ssl": "ssl_stage", "domain": "domain_stage"}[kind]
	if column == "" {
		return fmt.Errorf("unknown expiry kind %q", kind)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE monitors SET `+column+` = ? WHERE id = ?`, stage, id); err != nil {
		return err
	}
	if err := enqueue(ctx, tx, id, sql.NullInt64{}, n, at.UnixMilli()); err != nil {
		return err
	}
	return tx.Commit()
}

// Bucket counts the outcomes of one half hour.
type Bucket struct {
	Start   time.Time
	OK      int
	Fail    int
	Unknown int
}

func (s *Store) Buckets(ctx context.Context, from time.Time) (map[string][]Bucket, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT monitor_id, bucket, ok, fail, unknown FROM check_rollups
		WHERE bucket >= ? ORDER BY bucket`, from.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	buckets := map[string][]Bucket{}
	for rows.Next() {
		var id string
		var start int64
		var b Bucket
		if err := rows.Scan(&id, &start, &b.OK, &b.Fail, &b.Unknown); err != nil {
			return nil, err
		}
		b.Start = time.Unix(start, 0)
		buckets[id] = append(buckets[id], b)
	}
	return buckets, rows.Err()
}

// Incident is one outage; EndedAt is zero while it lasts.
type Incident struct {
	MonitorID  string
	StartedAt  time.Time
	EndedAt    time.Time
	NotifiedAt time.Time
}

// LatestIncidents returns the newest incident of each monitor.
func (s *Store) LatestIncidents(ctx context.Context) (map[string]Incident, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT monitor_id, started_at, COALESCE(ended_at, 0), COALESCE(notified_at, 0)
		FROM incidents WHERE id IN (SELECT MAX(id) FROM incidents GROUP BY monitor_id)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	incidents := map[string]Incident{}
	for rows.Next() {
		var i Incident
		var started, ended, notified int64
		if err := rows.Scan(&i.MonitorID, &started, &ended, &notified); err != nil {
			return nil, err
		}
		i.StartedAt, i.EndedAt, i.NotifiedAt = fromMillis(started), fromMillis(ended), fromMillis(notified)
		incidents[i.MonitorID] = i
	}
	return incidents, rows.Err()
}

func (s *Store) DueNotifications(ctx context.Context, now time.Time) ([]Notification, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, title, message, priority, tags, click, attempts FROM notifications
		WHERE sent_at IS NULL AND next_attempt <= ? ORDER BY id`, now.UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	due := []Notification{}
	for rows.Next() {
		var n Notification
		var tags string
		if err := rows.Scan(&n.ID, &n.Title, &n.Message, &n.Priority, &tags, &n.Click, &n.Attempts); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(tags), &n.Tags); err != nil {
			return nil, err
		}
		due = append(due, n)
	}
	return due, rows.Err()
}

func (s *Store) MarkSent(ctx context.Context, id int64, at time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE notifications SET sent_at = ? WHERE id = ?`, at.UnixMilli(), id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE incidents SET notified_at = ? WHERE notified_at IS NULL
		AND id = (SELECT incident_id FROM notifications WHERE id = ?)`, at.UnixMilli(), id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) MarkRetry(ctx context.Context, id int64, next time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE notifications SET attempts = attempts + 1, next_attempt = ? WHERE id = ?`,
		next.UnixMilli(), id)
	return err
}

// Prune applies retention: raw checks 30 days, rollups one year.
func (s *Store) Prune(ctx context.Context, now time.Time) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM checks WHERE at < ?`, now.AddDate(0, 0, -30).UnixMilli()); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM check_rollups WHERE bucket < ?`, now.AddDate(-1, 0, 0).Unix()); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, now.UnixMilli()); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE sent_at < ?`, now.AddDate(0, 0, -30).UnixMilli())
	return err
}
