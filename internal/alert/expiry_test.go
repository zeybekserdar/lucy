package alert

import (
	"testing"
	"time"
)

func TestDaysLeft(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	cases := map[time.Duration]int{
		10 * 24 * time.Hour:           10,
		10*24*time.Hour - time.Second: 9,
		time.Hour:                     0,
		-time.Hour:                    -1,
	}
	for offset, want := range cases {
		if got := DaysLeft(now.Add(offset), now); got != want {
			t.Errorf("%v = %d, want %d", offset, got, want)
		}
	}
}

func TestWarningsFireOncePerThreshold(t *testing.T) {
	stage, sent := 0, []int{}
	for days := 40; days >= -3; days-- {
		next, notify := NextExpiryStage(stage, ExpiryStage(days))
		stage = next
		if notify {
			sent = append(sent, days)
		}
	}
	want := []int{30, 14, 7, 0}
	if len(sent) != len(want) {
		t.Fatalf("sent = %v", sent)
	}
	for i := range want {
		if sent[i] != want[i] {
			t.Fatalf("sent = %v", sent)
		}
	}
}

func TestStartingInsideAThresholdWarnsImmediately(t *testing.T) {
	if _, notify := NextExpiryStage(0, ExpiryStage(5)); !notify {
		t.Fatal("expected a warning")
	}
}

func TestRenewalRearmsWarnings(t *testing.T) {
	stage, _ := NextExpiryStage(0, ExpiryStage(5))
	stage, notify := NextExpiryStage(stage, ExpiryStage(20)) // renewed for a short period
	if notify {
		t.Fatal("a renewal is not a warning")
	}
	if _, notify := NextExpiryStage(stage, ExpiryStage(14)); !notify {
		t.Fatal("the 14 day warning must fire again")
	}
	stage, _ = NextExpiryStage(stage, ExpiryStage(90))
	if stage != 0 {
		t.Fatalf("stage = %d", stage)
	}
}
