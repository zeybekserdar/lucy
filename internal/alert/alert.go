// Package alert is the monitor state machine: new, up, failing, down, recovering.
package alert

import "time"

type Phase string

const (
	PhaseNew        Phase = "new"
	PhaseUp         Phase = "up"
	PhaseFailing    Phase = "failing"
	PhaseDown       Phase = "down"
	PhaseRecovering Phase = "recovering"
)

type Outcome string

const (
	OK      Outcome = "ok"
	Fail    Outcome = "fail"
	Unknown Outcome = "unknown"
)

type Event string

const (
	None      Event = ""
	Down      Event = "down"
	Recovered Event = "recovered"
)

// RecoveryThreshold is how many successes in a row close an incident.
const RecoveryThreshold = 2

type State struct {
	Phase      Phase
	FailStreak int
	OkStreak   int
	FailSince  time.Time // first failure of the current streak
}

// Next applies one check outcome. Unknown leaves the state untouched: a check
// that could not reach the service says nothing about it.
func Next(s State, outcome Outcome, at time.Time, threshold int) (State, Event) {
	switch outcome {
	case OK:
		return onOK(s)
	case Fail:
		return onFail(s, at, threshold)
	}
	return s, None
}

func onOK(s State) (State, Event) {
	switch s.Phase {
	case PhaseDown, PhaseRecovering:
		streak := s.OkStreak + 1
		if streak >= RecoveryThreshold {
			return State{Phase: PhaseUp}, Recovered
		}
		return State{Phase: PhaseRecovering, OkStreak: streak, FailSince: s.FailSince}, None
	}
	return State{Phase: PhaseUp}, None
}

func onFail(s State, at time.Time, threshold int) (State, Event) {
	switch s.Phase {
	case PhaseDown, PhaseRecovering:
		return State{Phase: PhaseDown, FailSince: s.FailSince}, None
	}
	next := State{Phase: PhaseFailing, FailStreak: s.FailStreak + 1, FailSince: s.FailSince}
	if next.FailStreak == 1 {
		next.FailSince = at
	}
	if next.FailStreak >= threshold {
		next.Phase = PhaseDown
		next.FailStreak = 0
		return next, Down
	}
	return next, None
}
