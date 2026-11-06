package store

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"
)

const (
	keyPassword        = "password_hash"
	keyPasswordChanged = "password_changed_at"
)

// PasswordHash returns the stored hash, or "" before setup.
func (s *Store) PasswordHash(ctx context.Context) (string, error) {
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, keyPassword).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return hash, err
}

func (s *Store) PasswordChangedAt(ctx context.Context) (time.Time, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, keyPasswordChanged).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	ms, err := strconv.ParseInt(value, 10, 64)
	return fromMillis(ms), err
}

// ClaimPassword sets the first password. It reports false when one exists, so
// two simultaneous setups cannot both win.
func (s *Store) ClaimPassword(ctx context.Context, hash string, at time.Time) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)`, keyPassword, hash)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	if err != nil || count == 0 {
		return false, err
	}
	if err := setChanged(ctx, tx, at); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// ReplacePassword changes the password and ends every session.
func (s *Store) ReplacePassword(ctx context.Context, hash string, at time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, keyPassword, hash); err != nil {
		return err
	}
	if err := setChanged(ctx, tx, at); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions`); err != nil {
		return err
	}
	return tx.Commit()
}

func setChanged(ctx context.Context, tx *sql.Tx, at time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, keyPasswordChanged, strconv.FormatInt(at.UnixMilli(), 10))
	return err
}

func (s *Store) CreateSession(ctx context.Context, tokenHash string, now, expires time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions (token_hash, created_at, expires_at) VALUES (?, ?, ?)`,
		tokenHash, now.UnixMilli(), expires.UnixMilli())
	return err
}

func (s *Store) SessionValid(ctx context.Context, tokenHash string, now time.Time) (bool, error) {
	var expires int64
	err := s.db.QueryRowContext(ctx, `SELECT expires_at FROM sessions WHERE token_hash = ?`, tokenHash).Scan(&expires)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil && expires > now.UnixMilli(), err
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}
