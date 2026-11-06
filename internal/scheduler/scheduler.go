// Package scheduler runs each monitor on its own interval.
package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"lucy/internal/alert"
	"lucy/internal/checker"
	"lucy/internal/model"
	"lucy/internal/notify"
	"lucy/internal/store"
)

const maxConcurrentChecks = 8

type Scheduler struct {
	Store    *store.Store
	Checker  *checker.Checker
	Guard    *checker.Guard
	Composer *notify.Composer // nil turns notifications off
	Domains  DomainLookup     // nil turns domain checks off
	Log      *slog.Logger

	slots chan struct{}
	kick  chan struct{}
	ctx   context.Context
	mu    sync.Mutex
	loops map[string]context.CancelFunc
	locks sync.Map // monitor id -> *sync.Mutex
}

// Start launches every stored monitor, spreading first checks so a restart
// does not fire them all at once.
func (s *Scheduler) Start(ctx context.Context) error {
	s.ctx = ctx
	s.slots = make(chan struct{}, maxConcurrentChecks)
	s.kick = make(chan struct{}, 1)
	s.loops = map[string]context.CancelFunc{}
	monitors, err := s.Store.ListMonitors(ctx)
	if err != nil {
		return err
	}
	for _, m := range monitors {
		delay := time.Duration(rand.Int64N(int64(10 * time.Second)))
		s.launch(m.Config, delay)
	}
	if s.Domains != nil {
		go s.domainLoop(ctx)
	}
	return nil
}

// Upsert starts a monitor or restarts it with a new configuration, checking at once.
func (s *Scheduler) Upsert(cfg model.Config) {
	s.launch(cfg, 0)
	select {
	case s.kick <- struct{}{}: // look up its domain soon, not on the next hourly pass
	default:
	}
}

func (s *Scheduler) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cancel, ok := s.loops[id]; ok {
		cancel()
		delete(s.loops, id)
	}
}

func (s *Scheduler) launch(cfg model.Config, delay time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cancel, ok := s.loops[cfg.ID]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.loops[cfg.ID] = cancel
	go s.loop(ctx, cfg, delay)
}

func (s *Scheduler) loop(ctx context.Context, cfg model.Config, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		select {
		case s.slots <- struct{}{}:
		case <-ctx.Done():
			return
		}
		if _, err := s.Run(ctx, cfg); err != nil && ctx.Err() == nil {
			s.Log.Error("check", "monitor", cfg.Name, "err", err)
		}
		<-s.slots
		timer.Reset(time.Duration(cfg.IntervalSec) * time.Second)
	}
}

func (s *Scheduler) lock(id string) *sync.Mutex {
	lock, _ := s.locks.LoadOrStore(id, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// Run performs one check and applies its effects. Scheduled and manual checks
// of a monitor take turns, so its state machine never sees a lost update.
func (s *Scheduler) Run(ctx context.Context, cfg model.Config) (checker.Result, error) {
	lock := s.lock(cfg.ID)
	lock.Lock()
	defer lock.Unlock()

	result := s.Checker.Run(ctx, cfg)
	if ctx.Err() != nil {
		return result, ctx.Err() // shutting down or reconfigured, not a real failure
	}
	outcome := s.outcome(ctx, cfg, result)

	current, err := s.Store.GetMonitor(ctx, cfg.ID)
	if err != nil {
		return result, err
	}
	next, event := alert.Next(current.State, outcome, result.At, cfg.AlertThreshold)
	last, err := json.Marshal(result)
	if err != nil {
		return result, err
	}

	record := store.Record{
		MonitorID: cfg.ID,
		At:        result.At,
		Outcome:   outcome,
		Failure:   result.Failure(),
		Next:      next,
		Event:     event,

		CertExpires: result.CertExpires,
		CertIssuer:  result.CertIssuer,
		LastResult:  string(last),
	}
	if outcome == alert.OK {
		ms := result.ResponseMs
		record.ResponseMs = &ms
	}
	if s.Composer != nil {
		switch event {
		case alert.Down:
			record.Notification = s.Composer.Down(cfg.ID, cfg.Name, cfg.URL, result.Failure())
		case alert.Recovered:
			record.Notification = s.Composer.Recovered(cfg.ID, cfg.Name, result.At.Sub(current.Since))
		}
	}
	if err := s.Store.Apply(ctx, record); err != nil {
		return result, err
	}
	// A failed handshake reads no certificate, so fall back to the known date:
	// that is how an expired certificate still reaches its last warning.
	expires := result.CertExpires
	if expires.IsZero() {
		expires = current.SSLExpires
	}
	return result, s.warnExpiry(ctx, cfg, "ssl", expires, current.SSLStage)
}

// warnExpiry sends a stage's warning once and keeps the stage current.
func (s *Scheduler) warnExpiry(ctx context.Context, cfg model.Config, kind string, expires time.Time, stage int) error {
	if expires.IsZero() {
		return nil
	}
	now := time.Now()
	days := alert.DaysLeft(expires, now)
	next, notifyNow := alert.NextExpiryStage(stage, alert.ExpiryStage(days))
	if next == stage {
		return nil
	}
	var message *store.Notification
	if notifyNow && s.Composer != nil {
		message = s.Composer.Expiry(cfg.ID, cfg.Name, kind, days, expires)
	}
	return s.Store.SaveExpiryStage(ctx, cfg.ID, kind, next, message, now)
}

func (s *Scheduler) outcome(ctx context.Context, cfg model.Config, r checker.Result) alert.Outcome {
	local := checker.IsLocal(cfg.URL)
	if r.Error == "" && !local {
		s.Guard.MarkOnline() // any HTTP answer proves the internet works
	}
	if r.OK {
		return alert.OK
	}
	if r.Network() && !local && !s.Guard.Online(ctx) {
		return alert.Unknown
	}
	return alert.Fail
}
