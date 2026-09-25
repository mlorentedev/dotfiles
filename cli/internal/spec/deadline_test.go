package spec

import (
	"strings"
	"testing"
	"time"
)

// HARNESS-152: the reviewer is told when it will be stopped and when to aim to
// finish, so a slow review delivers a verdict on time instead of being killed
// without one. The times are absolute, because "30 minutes" says nothing to a
// model that cannot tell when the run started, and `date` can.
func TestTimeBudgetStatesTheTargetAndTheDeadline(t *testing.T) {
	start := time.Date(2026, 9, 25, 15, 0, 0, 0, time.FixedZone("MDT", -6*3600))
	b := TimeBudget(start, 45*time.Minute)
	for _, want := range []string{"15:45 MDT", "15:30 MDT", "`date`", "UNVERIFIED", "no verdict"} {
		if !strings.Contains(b, want) {
			t.Errorf("the time budget must mention %q:\n%s", want, b)
		}
	}
}

// A zero timeout means the default, as it does for ReviewerCommand: a budget
// that read "stopped at 15:00" would tell the reviewer it had no time at all.
func TestTimeBudgetTreatsZeroAsTheDefault(t *testing.T) {
	start := time.Date(2026, 9, 25, 15, 0, 0, 0, time.UTC)
	if got, want := TimeBudget(start, 0), TimeBudget(start, DefaultReviewerTimeout); got != want {
		t.Errorf("zero must mean the default:\n%s\nwant:\n%s", got, want)
	}
}
