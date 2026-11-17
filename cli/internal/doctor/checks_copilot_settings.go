package doctor

import (
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/copilotsettings"
)

func checkCopilotSettings(sys *System, rep *Report, fix bool) {
	rep.Section("GitHub Copilot local settings")
	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found: the retired-settings policy is checkout-owned")
		return
	}
	found, err := copilotsettings.Inspect(repo, sys.home(), sys.ResolvePath)
	if err != nil {
		rep.Fail("cannot inspect Copilot settings: " + err.Error())
		return
	}
	if found.Missing {
		rep.Skip(found.Settings + " not deployed; normal deploy creates it")
		return
	}
	if len(found.Keys) == 0 {
		rep.Pass("no retired local Copilot settings")
		return
	}
	if fix {
		fixCopilotSettings(sys, repo, found, rep)
		return
	}
	for _, key := range found.Keys {
		rep.Fail(fmt.Sprintf("%s contains retired local setting %s (run: dotf doctor --fix)",
			found.Settings, key))
	}
}

func fixCopilotSettings(sys *System, repo string, found copilotsettings.Result, rep *Report) {
	fixed, err := copilotsettings.Repair(repo, sys.home(), sys.ResolvePath, false)
	if err == nil {
		for _, key := range fixed.Keys {
			rep.Fix("removed " + key + " from " + fixed.Settings)
		}
		return
	}
	for _, key := range found.Keys {
		rep.Fail(fmt.Sprintf("%s contains retired local setting %s (--fix could not rewrite the file: %v)",
			found.Settings, key, err))
	}
}
