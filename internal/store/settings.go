package store

import (
	"context"
	"database/sql"
	"errors"
)

const (
	DefaultNtfyServer = "https://ntfy.sh"

	keyNtfyServer = "ntfy_server"
	keyNtfyTopic  = "ntfy_topic"
	keyNtfyToken  = "ntfy_token"
	keyPublicURL  = "public_url"
)

// NtfySettings says where notifications go. An empty topic turns them off.
// PublicURL is where tapping one leads, as seen from the phone.
type NtfySettings struct {
	Server    string
	Topic     string
	Token     string
	PublicURL string
}

func (s *Store) setting(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return value, err == nil, err
}

func (s *Store) Ntfy(ctx context.Context) (NtfySettings, error) {
	settings := NtfySettings{Server: DefaultNtfyServer}
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings WHERE key IN (?, ?, ?, ?)`,
		keyNtfyServer, keyNtfyTopic, keyNtfyToken, keyPublicURL)
	if err != nil {
		return settings, err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return settings, err
		}
		switch key {
		case keyNtfyServer:
			settings.Server = value
		case keyNtfyTopic:
			settings.Topic = value
		case keyNtfyToken:
			settings.Token = value
		case keyPublicURL:
			settings.PublicURL = value
		}
	}
	return settings, rows.Err()
}

func (s *Store) SaveNtfy(ctx context.Context, n NtfySettings) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	values := map[string]string{keyNtfyServer: n.Server, keyNtfyTopic: n.Topic, keyNtfyToken: n.Token, keyPublicURL: n.PublicURL}
	for key, value := range values {
		if _, err := tx.ExecContext(ctx, `INSERT INTO settings (key, value) VALUES (?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value`, key, value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SeedNtfy saves n unless settings were ever saved, even with an empty topic:
// clearing the topic on screen must survive a restart.
func (s *Store) SeedNtfy(ctx context.Context, n NtfySettings) error {
	_, saved, err := s.setting(ctx, keyNtfyTopic)
	if err != nil || saved {
		return err
	}
	return s.SaveNtfy(ctx, n)
}
