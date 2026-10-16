package worktree

import "testing"

// ProcessInside forwards Gate f's reading, and reports the platform without a
// scan as unsupported instead of passing its refusing "Inside" on as an answer.
func TestProcessInside_ForwardsTheGateReadingWhereSupported(t *testing.T) {
	if !processDiscoverySupported {
		r, supported := ProcessInside(t.TempDir())
		if supported || r != (GateFReading{}) {
			t.Errorf("ProcessInside = %+v, %v; want a zero reading and unsupported", r, supported)
		}
		return
	}
	original := hostProcessInside
	t.Cleanup(func() { hostProcessInside = original })
	want := GateFReading{Inside: true, Uninspectable: 3}
	hostProcessInside = func(string) GateFReading { return want }
	if got, supported := ProcessInside("/wt"); !supported || got != want {
		t.Errorf("ProcessInside = %+v, %v; want %+v, true", got, supported, want)
	}
}
