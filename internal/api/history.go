package api

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"lucy/internal/store"
)

const (
	eventDays = 30
	// A restart is not worth a line in the history, however short the interval.
	minGap = 5 * time.Minute
)

type chartRange struct {
	span  time.Duration
	slots int
}

var chartRanges = map[string]chartRange{
	"24h": {24 * time.Hour, 96},
	"7d":  {7 * 24 * time.Hour, 168},
	"30d": {30 * 24 * time.Hour, 180},
}

// event is an outage or a stretch without data; End is nil while it lasts.
type event struct {
	Kind  string     `json:"kind"` // outage, no_data
	Start time.Time  `json:"start"`
	End   *time.Time `json:"end"`
	Cause string     `json:"cause,omitempty"`
}

type historyView struct {
	Monitor      monitorView     `json:"monitor"`
	From         time.Time       `json:"from"`
	To           time.Time       `json:"to"`
	Points       []*int          `json:"points"` // evenly spaced from From to To
	AvgMs        *int            `json:"avgMs"`
	Events       []event         `json:"events"` // newest first, always the last 30 days
	FailedChecks int             `json:"failedChecks"`
	LastResult   json.RawMessage `json:"lastResult"`
}

func gapLimit(intervalSec int) time.Duration {
	return max(3*time.Duration(intervalSec)*time.Second, minGap)
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("range")
	if name == "" {
		name = "7d"
	}
	span, ok := chartRanges[name]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_range"})
		return
	}
	view, err := s.loadHistory(r.Context(), r.PathValue("id"), span, time.Now())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) loadHistory(ctx context.Context, id string, span chartRange, now time.Time) (historyView, error) {
	var view historyView
	m, err := s.Store.GetMonitor(ctx, id)
	if err != nil {
		return view, err
	}
	buckets, err := s.Store.Buckets(ctx, now.AddDate(0, 0, -(dayCount+1)))
	if err != nil {
		return view, err
	}
	latest, err := s.Store.LatestIncidents(ctx)
	if err != nil {
		return view, err
	}
	view.Monitor = toView(m, buckets[id], latest, now)
	view.From, view.To = now.Add(-span.span), now
	if view.Points, view.AvgMs, err = s.Store.Series(ctx, id, view.From, now, span.slots); err != nil {
		return view, err
	}
	if view.Events, err = s.events(ctx, m, now); err != nil {
		return view, err
	}
	if view.Monitor.Status == "down" {
		if view.FailedChecks, err = s.Store.FailedChecks(ctx, id, m.Since); err != nil {
			return view, err
		}
	}
	if m.LastResult != "" {
		view.LastResult = json.RawMessage(m.LastResult)
	}
	return view, nil
}

func (s *Server) events(ctx context.Context, m store.Monitor, now time.Time) ([]event, error) {
	from := now.AddDate(0, 0, -eventDays)
	incidents, err := s.Store.Incidents(ctx, m.ID, from)
	if err != nil {
		return nil, err
	}
	gaps, err := s.Store.Gaps(ctx, m.ID, from, now, gapLimit(m.IntervalSec))
	if err != nil {
		return nil, err
	}
	events := make([]event, 0, len(incidents)+len(gaps))
	for _, i := range incidents {
		events = append(events, event{Kind: "outage", Start: i.StartedAt, End: optional(i.EndedAt), Cause: i.Cause})
	}
	for _, g := range gaps {
		events = append(events, event{Kind: "no_data", Start: g.Start, End: optional(g.End)})
	}
	slices.SortFunc(events, func(a, b event) int { return b.Start.Compare(a.Start) })
	return events, nil
}
