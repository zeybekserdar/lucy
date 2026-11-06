package scheduler

import (
	"context"
	"time"

	"lucy/internal/checker"
	"lucy/internal/store"
)

// Registries change slowly and throttle eager clients.
const (
	domainRefresh = 24 * time.Hour
	domainPass    = time.Hour
)

type DomainLookup interface {
	Lookup(ctx context.Context, rawURL string) (checker.DomainInfo, error)
}

func (s *Scheduler) domainLoop(ctx context.Context) {
	ticker := time.NewTicker(domainPass)
	defer ticker.Stop()
	for {
		s.RefreshDomains(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-s.kick:
		}
	}
}

// RefreshDomains looks up every stale monitor, asking once per domain. A failed
// lookup leaves the old answer and the next pass tries again.
func (s *Scheduler) RefreshDomains(ctx context.Context) {
	monitors, err := s.Store.ListMonitors(ctx)
	if err != nil {
		s.Log.Error("list monitors", "err", err)
		return
	}
	now := time.Now()
	answers := map[string]checker.DomainInfo{}
	for _, m := range monitors {
		if now.Sub(m.DomainChecked) < domainRefresh {
			continue
		}
		key, _ := checker.Registrable(m.URL)
		info, known := answers[key]
		if !known || key == "" {
			info, err = s.Domains.Lookup(ctx, m.URL)
			if err != nil {
				s.Log.Warn("domain lookup", "monitor", m.Name, "err", err)
				continue
			}
			answers[key] = info
		}
		if err := s.applyDomain(ctx, m, info, now); err != nil {
			s.Log.Error("save domain", "monitor", m.Name, "err", err)
		}
	}
}

func (s *Scheduler) applyDomain(ctx context.Context, m store.Monitor, info checker.DomainInfo, now time.Time) error {
	if err := s.Store.SaveDomain(ctx, m.ID, info.Expires, info.Reason, now); err != nil {
		return err
	}
	return s.warnExpiry(ctx, m.Config, "domain", info.Expires, m.DomainStage)
}
