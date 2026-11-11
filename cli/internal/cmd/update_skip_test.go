package cmd

import (
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
)

// TestSystemUpgradeSkipReadsOnlyTheStepsSkip pins how a quiet day's `dotf
// update` tells a skipped step from one that ran: only the system-upgrade
// entry's own skip counts, and a failed step is not a skip (the run's error
// carries it).
func TestSystemUpgradeSkipReadsOnlyTheStepsSkip(t *testing.T) {
	const reason = "needs root: run `dotf converge --only system-upgrade` from a terminal"
	tests := []struct {
		name string
		rep  converge.Report
		want string
	}{
		{name: "skipped", rep: converge.Report{Entries: []converge.Entry{{Name: "system-upgrade", Status: converge.StatusSkipped, Detail: reason}}}, want: reason},
		{name: "ran", rep: converge.Report{Entries: []converge.Entry{{Name: "system-upgrade", Status: converge.StatusOK, Detail: "catalog entries up to date"}}}},
		{name: "changed", rep: converge.Report{Entries: []converge.Entry{{Name: "system-upgrade", Status: converge.StatusChange, Changes: 2}}}},
		{name: "failed is not a skip", rep: converge.Report{Entries: []converge.Entry{{Name: "system-upgrade", Status: converge.StatusFailed, Detail: "brew upgrade: exit 1"}}}},
		{name: "another step's skip", rep: converge.Report{Entries: []converge.Entry{{Name: "tools", Status: converge.StatusSkipped, Detail: "x"}}}},
		{name: "empty report", rep: converge.Report{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := systemUpgradeSkip(tt.rep); got != tt.want {
				t.Errorf("systemUpgradeSkip = %q, want %q", got, tt.want)
			}
		})
	}
}
