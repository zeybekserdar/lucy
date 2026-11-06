package store

import (
	"context"
	"time"
)

// Series averages successful response times into equal slots between from and
// to. A slot without one stays nil, and the second value is the overall average.
func (s *Store) Series(ctx context.Context, id string, from, to time.Time, slots int) ([]*int, *int, error) {
	start := from.UnixMilli()
	width := max((to.UnixMilli()-start)/int64(slots), 1)
	rows, err := s.db.QueryContext(ctx, `SELECT (at - ?) / ?, SUM(response_ms), COUNT(response_ms) FROM checks
		WHERE monitor_id = ? AND at >= ? AND at <= ? AND response_ms IS NOT NULL GROUP BY 1`,
		start, width, id, start, to.UnixMilli())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	points := make([]*int, slots)
	var total, count int64
	for rows.Next() {
		var slot, sum, n int64
		if err := rows.Scan(&slot, &sum, &n); err != nil {
			return nil, nil, err
		}
		value := int(sum / n)
		points[min(int(slot), slots-1)] = &value // the last millisecond falls past the final slot
		total += sum
		count += n
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if count == 0 {
		return points, nil, nil
	}
	average := int(total / count)
	return points, &average, nil
}

// Gap is a stretch without checks: Lucy was not running.
type Gap struct {
	Start time.Time
	End   time.Time
}

// Gaps finds silences longer than limit since from, including one still open
// at now. Time before a monitor's first check is not a gap: it did not exist.
func (s *Store) Gaps(ctx context.Context, id string, from, now time.Time, limit time.Duration) ([]Gap, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT prev, at FROM (
			SELECT at, LAG(at) OVER (ORDER BY at) AS prev FROM checks WHERE monitor_id = ? AND at >= ?
		) WHERE at - prev > ? ORDER BY at`, id, from.UnixMilli(), limit.Milliseconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	gaps := []Gap{}
	for rows.Next() {
		var start, end int64
		if err := rows.Scan(&start, &end); err != nil {
			return nil, err
		}
		gaps = append(gaps, Gap{Start: fromMillis(start), End: fromMillis(end)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var latest int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(at), 0) FROM checks WHERE monitor_id = ? AND at >= ?`,
		id, from.UnixMilli()).Scan(&latest); err != nil {
		return nil, err
	}
	if latest != 0 && now.UnixMilli()-latest > limit.Milliseconds() {
		gaps = append(gaps, Gap{Start: fromMillis(latest), End: now})
	}
	return gaps, nil
}

// Incidents returns the outages of a monitor that touch the time since from.
func (s *Store) Incidents(ctx context.Context, id string, from time.Time) ([]Incident, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT started_at, COALESCE(ended_at, 0), cause FROM incidents
		WHERE monitor_id = ? AND (ended_at IS NULL OR ended_at >= ?) ORDER BY started_at`, id, from.UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	incidents := []Incident{}
	for rows.Next() {
		i := Incident{MonitorID: id}
		var started, ended int64
		if err := rows.Scan(&started, &ended, &i.Cause); err != nil {
			return nil, err
		}
		i.StartedAt, i.EndedAt = fromMillis(started), fromMillis(ended)
		incidents = append(incidents, i)
	}
	return incidents, rows.Err()
}

// FailedChecks counts the failures of a monitor since a moment.
func (s *Store) FailedChecks(ctx context.Context, id string, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM checks WHERE monitor_id = ? AND at >= ? AND outcome = 'fail'`,
		id, since.UnixMilli()).Scan(&count)
	return count, err
}
