package alert

import (
	"testing"
	"time"
)

func play(t *testing.T, threshold int, outcomes ...Outcome) (State, []Event) {
	t.Helper()
	state := State{Phase: PhaseNew}
	at := time.Unix(1000, 0)
	var events []Event
	for _, outcome := range outcomes {
		var event Event
		state, event = Next(state, outcome, at, threshold)
		if event != None {
			events = append(events, event)
		}
		at = at.Add(time.Minute)
	}
	return state, events
}

func TestTransitions(t *testing.T) {
	cases := []struct {
		name     string
		outcomes []Outcome
		phase    Phase
		events   int
	}{
		{"first success", []Outcome{OK}, PhaseUp, 0},
		{"below threshold", []Outcome{OK, Fail, Fail}, PhaseFailing, 0},
		{"blip resets the streak", []Outcome{Fail, Fail, OK, Fail, Fail}, PhaseFailing, 0},
		{"threshold opens", []Outcome{OK, Fail, Fail, Fail}, PhaseDown, 1},
		{"stays down without repeating", []Outcome{Fail, Fail, Fail, Fail, Fail}, PhaseDown, 1},
		{"one success is not recovery", []Outcome{Fail, Fail, Fail, OK}, PhaseRecovering, 1},
		{"failure during recovery", []Outcome{Fail, Fail, Fail, OK, Fail, OK}, PhaseRecovering, 1},
		{"two successes recover", []Outcome{Fail, Fail, Fail, OK, OK}, PhaseUp, 2},
		{"unknown never alarms", []Outcome{OK, Unknown, Unknown, Unknown, Unknown}, PhaseUp, 0},
		{"unknown keeps the streak", []Outcome{Fail, Unknown, Fail, Unknown, Fail}, PhaseDown, 1},
		{"unknown keeps an incident open", []Outcome{Fail, Fail, Fail, Unknown, Unknown}, PhaseDown, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state, events := play(t, 3, c.outcomes...)
			if state.Phase != c.phase {
				t.Errorf("phase = %s, want %s", state.Phase, c.phase)
			}
			if len(events) != c.events {
				t.Errorf("events = %v, want %d", events, c.events)
			}
		})
	}
}

func TestThresholdOneAlarmsImmediately(t *testing.T) {
	if _, events := play(t, 1, Fail); len(events) != 1 || events[0] != Down {
		t.Fatalf("events = %v", events)
	}
}

func TestFailSinceIsTheFirstFailure(t *testing.T) {
	state, _ := play(t, 3, OK, Fail, Fail, Fail)
	if want := time.Unix(1000, 0).Add(time.Minute); !state.FailSince.Equal(want) {
		t.Fatalf("FailSince = %v, want %v", state.FailSince, want)
	}
}
