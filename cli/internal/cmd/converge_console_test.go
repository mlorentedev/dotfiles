package cmd

import (
	"bytes"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/console/consoletest"
	"github.com/mlorentedev/dotfiles/cli/internal/converge"
)

// consoleReport holds one entry of every status, the shape a real run prints.
func consoleReport(dryRun bool) converge.Report {
	return converge.Report{DryRun: dryRun, GOOS: "linux", Entries: []converge.Entry{
		{Name: "tools", Status: converge.StatusChange, Changes: 2, Detail: "installed herdr, lazygit"},
		{Name: "pi-settings", Status: converge.StatusOK, Detail: "settings.json present, enabledModels in sync"},
		{Name: "legacy-setup", Status: converge.StatusOpaque, Detail: "setup-linux.sh exited 0"},
		{Name: "identity", Status: converge.StatusFailed, Detail: "the age key is locked"},
		{Name: "records-harness", Status: converge.StatusSkipped, Detail: "not reached"},
	}}
}

// CLI-114 AC3: a pipe reads the same text it read before colour existed.
func TestConvergeConsole_Colour(t *testing.T) {
	for _, plan := range []bool{false, true} {
		name := "converge-apply"
		if plan {
			name = "converge-plan"
		}
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			printConvergeReport(&buf, consoleReport(plan))
			consoletest.Golden(t, name+"-plain", buf.Bytes())
		})
	}
}
