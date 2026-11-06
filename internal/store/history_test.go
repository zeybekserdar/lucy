package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"lucy/internal/alert"
	"lucy/internal/model"
)

func seeded(t *testing.T) (*Store, string) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "lucy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	cfg, err := db.CreateMonitor(context.Background(), model.Config{Name: "a", URL: "https://a.test", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	return db, cfg.ID
}

func addCheck(t *testing.T, db *Store, id string, at time.Time, outcome alert.Outcome, ms *int) {
	t.Helper()
	next := alert.State{Phase: alert.PhaseUp}
	if err := db.Apply(context.Background(), Record{MonitorID: id, At: at, Outcome: outcome, ResponseMs: ms, Next: next}); err != nil {
		t.Fatal(err)
	}
}

func ms(v int) *int { return &v }

func TestSeriesAveragesSlotsAndLeavesFailuresEmpty(t *testing.T) {
	db, id := seeded(t)
	from := time.UnixMilli(1_700_000_000_000)
	to := from.Add(4 * time.Hour)
	addCheck(t, db, id, from.Add(10*time.Minute), alert.OK, ms(100))
	addCheck(t, db, id, from.Add(20*time.Minute), alert.OK, ms(200))
	addCheck(t, db, id, from.Add(90*time.Minute), alert.Fail, nil)
	addCheck(t, db, id, to, alert.OK, ms(300)) // the very last moment stays in range

	points, average, err := db.Series(context.Background(), id, from, to, 4)
	if err != nil {
		t.Fatal(err)
	}
	if points[0] == nil || *points[0] != 150 {
		t.Fatalf("first slot: %v", points[0])
	}
	if points[1] != nil || points[2] != nil {
		t.Fatalf("failed and empty slots must stay empty: %v %v", points[1], points[2])
	}
	if points[3] == nil || *points[3] != 300 {
		t.Fatalf("last slot: %v", points[3])
	}
	if average == nil || *average != 200 {
		t.Fatalf("average: %v", average)
	}
}

func TestSeriesWithoutChecksHasNoAverage(t *testing.T) {
	db, id := seeded(t)
	now := time.Now()
	points, average, err := db.Series(context.Background(), id, now.Add(-time.Hour), now, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 3 || average != nil {
		t.Fatalf("points %v average %v", points, average)
	}
}

func TestGapsFindSilenceButNotTheTimeBeforeTheFirstCheck(t *testing.T) {
	db, id := seeded(t)
	now := time.UnixMilli(1_700_000_000_000)
	from := now.Add(-24 * time.Hour)
	first := now.Add(-10 * time.Hour)
	addCheck(t, db, id, first, alert.OK, ms(1))
	addCheck(t, db, id, first.Add(4*time.Minute), alert.OK, ms(1))
	back := first.Add(2 * time.Hour)
	addCheck(t, db, id, back, alert.OK, ms(1))
	addCheck(t, db, id, now.Add(-time.Minute), alert.OK, ms(1))

	gaps, err := db.Gaps(context.Background(), id, from, now, 5*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	// back to now-1m is the only silence over five hours.
	if len(gaps) != 1 || !gaps[0].Start.Equal(back) {
		t.Fatalf("gaps: %+v", gaps)
	}

	gaps, err = db.Gaps(context.Background(), id, from, now.Add(6*time.Hour), 5*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps) != 2 || !gaps[1].End.Equal(now.Add(6*time.Hour)) {
		t.Fatalf("an open silence must reach now: %+v", gaps)
	}
}

func TestGapsOfAMonitorWithoutChecks(t *testing.T) {
	db, id := seeded(t)
	now := time.Now()
	gaps, err := db.Gaps(context.Background(), id, now.Add(-time.Hour), now, time.Minute)
	if err != nil || len(gaps) != 0 {
		t.Fatalf("gaps %+v err %v", gaps, err)
	}
}
