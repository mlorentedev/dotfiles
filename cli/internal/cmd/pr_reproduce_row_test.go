package cmd

import (
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/reproduce"
)

// #3333: the row names the control's reading, so a triage record shows both
// exits a probed reproduction rests on, and a row without a control says nothing
// about one.
func TestReproductionRowNamesTheControl(t *testing.T) {
	base := reproduce.Result{Exit: 1, Head: "0123456789abcdef", Command: []string{"go", "test", "./x"}}

	controlled := base
	controlled.State, controlled.Control, controlled.ControlExit = reproduce.CouldNotRun, reproduce.Reproduced, 1
	row := reproductionRow(controlled, "/tmp/probe.diff", false)
	want := "Reproduction: could-not-run (exit 1, control reproduced (exit 1) on the clean head) on 0123456789ab with probe probe.diff: `go test ./x`"
	if row != want {
		t.Errorf("row =\n%s\nwant\n%s", row, want)
	}

	plain := base
	plain.State = reproduce.Reproduced
	if row := reproductionRow(plain, "", false); strings.Contains(row, "control") {
		t.Errorf("a run without a control names one: %s", row)
	}
}
