package alert

import (
	"math"
	"time"
)

// ExpiryWarnDays is when an approaching expiry turns a monitor into a warning.
const ExpiryWarnDays = 14

// A stage is a warning threshold plus one, so zero can mean "nothing to say".
// Lower is more urgent. There is no 30-day stage: Let's Encrypt renews around
// then, so it would warn about certificates that are about to renew themselves.
var expiryThresholds = []int{0, 7, 14}

// DaysLeft counts whole days; zero or less means it has expired or expires today.
func DaysLeft(expires, now time.Time) int {
	return int(math.Floor(expires.Sub(now).Hours() / 24))
}

func ExpiryStage(daysLeft int) int {
	for _, threshold := range expiryThresholds {
		if daysLeft <= threshold {
			return threshold + 1
		}
	}
	return 0
}

// NextExpiryStage decides whether a stage is news. Each stage notifies once,
// and a renewal moves the stage back so later warnings fire again.
func NextExpiryStage(previous, current int) (next int, notify bool) {
	if current == 0 {
		return 0, false
	}
	if previous == 0 || current < previous {
		return current, true
	}
	return current, false
}
