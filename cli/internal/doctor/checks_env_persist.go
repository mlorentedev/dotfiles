package doctor

import (
	"fmt"
	"strings"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

// checkPersistedEnv (CLI-058, #1324) reports whether the resolved contract
// variables also live in the per-user PERSISTENT scope — what a process
// started without a profile inherits (Copilot's `pwsh -NoProfile` tool calls,
// a Scheduled Task). The rc files' cascade is invisible there: measured on
// the Windows work box, DOTFILES_REPO_DIR, DOTFILES_DIR, VAULT_PATH and
// COPILOT_HOME were all empty at User scope while every shell had them, and
// `dotf harness mirror` inside a Copilot tool call could not find the
// checkout. WARN, not FAIL: shells keep working; the remedy is one command.
// Skipped where the OS has no such scope (sys.UserEnv is nil).
//
// The contract is located through the seam (checkout via DOTFILES_REPO_DIR,
// else the deploy mirror), never through the process environment, so the
// check is testable against a fixture and reads the same file setup mirrored.
func checkPersistedEnv(sys *System, cfg *Config, rep *Report) {
	if sys.UserEnv == nil {
		return
	}
	rep.Section("Persisted environment (user scope)")
	home := sys.home()
	contractPath := envpkg.ResolveRepoFirst("env-contract.json", sys.Getenv("DOTFILES_REPO_DIR"), cfg.DotfilesDir, "")
	if contractPath == "" {
		rep.Skip("env-contract.json not found — nothing to compare")
		return
	}
	vars, err := envpkg.ResolveVars(contractPath, envpkg.MachinePath(home), sys.GOOS, home)
	if err != nil {
		rep.Warn("contract variables unresolvable (" + err.Error() + ")")
		return
	}
	drift, err := envpkg.Drift(vars, userEnvAdapter{sys.UserEnv})
	if err != nil {
		rep.Warn("user-scope environment unreadable (" + err.Error() + ")")
		return
	}
	if len(drift) == 0 {
		rep.Pass(fmt.Sprintf("%d contract variable(s) persisted at user scope", len(vars)))
		return
	}
	names := make([]string, 0, len(drift))
	for _, v := range drift {
		names = append(names, v.Name)
	}
	rep.Warn(fmt.Sprintf("%d contract variable(s) missing or stale at user scope (%s) — profile-less processes such as Copilot tool calls see none of them; run `dotf env persist`",
		len(drift), strings.Join(names, ", ")))
}

// userEnvAdapter lets the doctor seam (a function) satisfy env.UserEnvStore.
type userEnvAdapter struct {
	get func(name string) (string, bool, error)
}

func (a userEnvAdapter) Get(name string) (string, bool, error) { return a.get(name) }
func (userEnvAdapter) Set(string, string) error                { return nil }
