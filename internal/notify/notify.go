// Package notify delivers queued notifications. New channels implement Notifier.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"lucy/internal/store"
)

type Notifier interface {
	Send(ctx context.Context, n store.Notification) error
}

// Ntfy publishes as JSON, because HTTP headers cannot carry Turkish titles.
type Ntfy struct {
	Server string
	Topic  string
	Token  string
	Client *http.Client
}

func (n Ntfy) Send(ctx context.Context, msg store.Notification) error {
	_, err := n.Publish(ctx, msg)
	return err
}

// Publish also reports the HTTP status, 0 when the server gave no answer.
func (n Ntfy) Publish(ctx context.Context, msg store.Notification) (int, error) {
	payload, err := json.Marshal(map[string]any{
		"topic":    n.Topic,
		"title":    msg.Title,
		"message":  msg.Message,
		"priority": msg.Priority,
		"tags":     msg.Tags,
		"click":    msg.Click,
	})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(n.Server, "/")+"/", bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if n.Token != "" {
		req.Header.Set("Authorization", "Bearer "+n.Token)
	}
	client := n.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("ntfy: %s", resp.Status)
	}
	return resp.StatusCode, nil
}

// ErrNotConfigured keeps a message queued until a topic is set.
var ErrNotConfigured = errors.New("ntfy: no topic configured")

// Live reads the settings on every send, so changes apply without a restart.
type Live struct {
	Store *store.Store
}

func (l Live) Send(ctx context.Context, msg store.Notification) error {
	settings, err := l.Store.Ntfy(ctx)
	if err != nil {
		return err
	}
	if settings.Topic == "" {
		return ErrNotConfigured
	}
	return Ntfy{Server: settings.Server, Topic: settings.Topic, Token: settings.Token}.Send(ctx, msg)
}

// Configured reports whether notifications have somewhere to go.
func (l Live) Configured(ctx context.Context) bool {
	settings, err := l.Store.Ntfy(ctx)
	return err == nil && settings.Topic != ""
}

const (
	retryBase = 5 * time.Second
	retryMax  = 5 * time.Minute
)

// Backoff is the wait before the next attempt.
func Backoff(attempts int) time.Duration {
	wait := retryBase
	for i := 0; i < attempts && wait < retryMax; i++ {
		wait *= 2
	}
	return min(wait, retryMax)
}

// Worker drains the queue. A message stays queued until delivered, so an
// internet outage delays notifications without losing them.
type Worker struct {
	Store    *store.Store
	Notifier Notifier
	Log      *slog.Logger
	Now      func() time.Time // nil means time.Now
}

func (w Worker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

func (w Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		w.Flush(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w Worker) Flush(ctx context.Context) {
	due, err := w.Store.DueNotifications(ctx, w.now())
	if err != nil {
		w.Log.Error("read notification queue", "err", err)
		return
	}
	for _, n := range due {
		if err := w.Notifier.Send(ctx, n); err != nil {
			w.Log.Warn("send notification", "title", n.Title, "attempts", n.Attempts+1, "err", err)
			if err := w.Store.MarkRetry(ctx, n.ID, w.now().Add(Backoff(n.Attempts))); err != nil {
				w.Log.Error("schedule retry", "err", err)
			}
			continue
		}
		if err := w.Store.MarkSent(ctx, n.ID, w.now()); err != nil {
			w.Log.Error("mark notification sent", "err", err)
		}
	}
}
