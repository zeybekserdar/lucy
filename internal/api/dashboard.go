package api

import (
	"context"
	"time"

	"lucy/internal/alert"
	"lucy/internal/model"
	"lucy/internal/store"
)

const (
	dayCount      = 30
	halfHourCount = 48
)

type expiry struct {
	DaysLeft  *int       `json:"daysLeft"`
	ExpiresAt *time.Time `json:"expiresAt"`
	Issuer    string     `json:"issuer,omitempty"`
	Reason    string     `json:"reason,omitempty"`
}

func toExpiry(expires time.Time, now time.Time) expiry {
	if expires.IsZero() {
		return expiry{}
	}
	days := alert.DaysLeft(expires, now)
	return expiry{DaysLeft: &days, ExpiresAt: &expires}
}

func expiring(e expiry) bool {
	return e.DaysLeft != nil && *e.DaysLeft <= alert.ExpiryWarnDays
}

type uptime struct {
	D1  *float64 `json:"d1"`
	D7  *float64 `json:"d7"`
	D30 *float64 `json:"d30"`
}

type lastIncident struct {
	EndedAt     time.Time `json:"endedAt"`
	DurationSec int       `json:"durationSec"`
}

type monitorView struct {
	model.Config
	Status       string        `json:"status"`
	Since        *time.Time    `json:"since"`
	LastCheck    *time.Time    `json:"lastCheck"`
	ResponseMs   *int          `json:"responseMs"`
	Uptime       uptime        `json:"uptime"`
	Days         []string      `json:"days"`
	HalfHours    []string      `json:"halfHours"`
	SSL          expiry        `json:"ssl"`
	Domain       expiry        `json:"domain"`
	Failure      string        `json:"failure,omitempty"`
	NotifiedAt   *time.Time    `json:"notifiedAt,omitempty"`
	LastIncident *lastIncident `json:"lastIncident,omitempty"`
}

type network struct {
	Online       bool       `json:"online"`
	OfflineSince *time.Time `json:"offlineSince,omitempty"`
}

type dashboardView struct {
	Monitors []monitorView `json:"monitors"`
	Network  network       `json:"network"`
}

func optional(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func (s *Server) dashboard(ctx context.Context) (dashboardView, error) {
	now := time.Now()
	monitors, err := s.Store.ListMonitors(ctx)
	if err != nil {
		return dashboardView{}, err
	}
	buckets, err := s.Store.Buckets(ctx, now.AddDate(0, 0, -(dayCount+1)))
	if err != nil {
		return dashboardView{}, err
	}
	incidents, err := s.Store.LatestIncidents(ctx)
	if err != nil {
		return dashboardView{}, err
	}

	online, offlineSince := s.Guard.State()
	view := dashboardView{
		Monitors: make([]monitorView, 0, len(monitors)),
		Network:  network{Online: online, OfflineSince: optional(offlineSince)},
	}
	for _, m := range monitors {
		view.Monitors = append(view.Monitors, toView(m, buckets[m.ID], incidents, now))
	}
	return view, nil
}

func status(m store.Monitor) string {
	switch m.State.Phase {
	case alert.PhaseNew:
		return "unknown"
	case alert.PhaseDown, alert.PhaseRecovering:
		return "down"
	}
	if m.LastOutcome == alert.Unknown {
		return "unknown"
	}
	return "up"
}

func toView(m store.Monitor, buckets []store.Bucket, incidents map[string]store.Incident, now time.Time) monitorView {
	ssl := toExpiry(m.SSLExpires, now)
	ssl.Issuer = m.SSLIssuer
	domain := toExpiry(m.DomainExpires, now)
	domain.Reason = m.DomainReason
	view := monitorView{
		Config:    m.Config,
		SSL:       ssl,
		Domain:    domain,
		Status:    status(m),
		Since:     optional(m.Since),
		LastCheck: optional(m.LastCheck),
		Uptime: uptime{
			D1:  uptimeSince(buckets, now.AddDate(0, 0, -1)),
			D7:  uptimeSince(buckets, now.AddDate(0, 0, -7)),
			D30: uptimeSince(buckets, now.AddDate(0, 0, -30)),
		},
		Days:      dayStates(buckets, now),
		HalfHours: halfHourStates(buckets, now),
	}
	if view.Status == "up" {
		view.ResponseMs = m.ResponseMs
		if expiring(ssl) || expiring(domain) {
			view.Status = "warn"
		}
	}
	if view.Status == "down" {
		view.Failure = m.Failure
	}
	if incident, ok := incidents[m.ID]; ok {
		if incident.EndedAt.IsZero() {
			view.NotifiedAt = optional(incident.NotifiedAt)
		} else {
			view.LastIncident = &lastIncident{
				EndedAt:     incident.EndedAt,
				DurationSec: int(incident.EndedAt.Sub(incident.StartedAt).Seconds()),
			}
		}
	}
	return view
}

// uptimeSince ignores unknown results and periods without data, so neither an
// internet outage nor Lucy being off counts against a service.
func uptimeSince(buckets []store.Bucket, from time.Time) *float64 {
	ok, fail := 0, 0
	for _, b := range buckets {
		if !b.Start.Before(from) {
			ok += b.OK
			fail += b.Fail
		}
	}
	if ok+fail == 0 {
		return nil
	}
	value := float64(ok) / float64(ok+fail) * 100
	return &value
}

func classify(ok, fail, unknown int) string {
	switch {
	case ok+fail+unknown == 0:
		return "no_data"
	case ok+fail == 0:
		return "unknown"
	case fail == 0:
		return "ok"
	case fail*2 >= ok+fail:
		return "down"
	}
	return "partial"
}

// dayStates buckets by the server's calendar days; the last entry is today.
func dayStates(buckets []store.Bucket, now time.Time) []string {
	counts := make([][3]int, dayCount)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for _, b := range buckets {
		start := b.Start.In(now.Location())
		day := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
		ago := int(today.Sub(day).Hours()/24 + 0.5)
		if ago >= 0 && ago < dayCount {
			c := &counts[dayCount-1-ago]
			c[0] += b.OK
			c[1] += b.Fail
			c[2] += b.Unknown
		}
	}
	states := make([]string, dayCount)
	for i, c := range counts {
		states[i] = classify(c[0], c[1], c[2])
	}
	return states
}

func halfHourStates(buckets []store.Bucket, now time.Time) []string {
	latest := now.Unix() / store.BucketSeconds * store.BucketSeconds
	byStart := make(map[int64]store.Bucket, len(buckets))
	for _, b := range buckets {
		byStart[b.Start.Unix()] = b
	}
	states := make([]string, halfHourCount)
	for i := range states {
		b := byStart[latest-int64(halfHourCount-1-i)*store.BucketSeconds]
		states[i] = classify(b.OK, b.Fail, b.Unknown)
	}
	return states
}
