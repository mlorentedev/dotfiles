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
	if err := envpkg.ValidateNames(vars); err != nil {
		rep.Warn(err.Error() + " — fix env-contract.json; `dotf env persist` refuses it")
		return
	}
	reader := userEnvAdapter{sys.UserEnv}
	drift, err := envpkg.Drift(vars, reader)
	if err != nil {
		rep.Warn("user-scope environment unreadable (" + err.Error() + ")")
		return
	}
	retired, err := envpkg.Retired(reader, vars)
	if err != nil {
		rep.Warn("user-scope environment unreadable (" + err.Error() + ")")
		return
	}
	stale, err := envpkg.MarkerStale(reader, vars)
	if err != nil {
		rep.Warn("user-scope environment unreadable (" + err.Error() + ")")
		return
	}
	fix, who := persistRemedy(sys.GOOS)
	if len(drift) == 0 && len(retired) == 0 && !stale {
		rep.Pass(fmt.Sprintf("%d contract variable(s) persisted at user scope", len(vars)))
		return
	}
	if len(drift) == 0 && len(retired) == 0 {
		// Every variable is in place; only the ownership record lags the
		// contract (a box that persisted before the record existed, or a
		// leftover a hand edit removed). One run rewrites it.
		rep.Warn(fmt.Sprintf("%d contract variable(s) persisted at user scope, but the ownership record (%s) is out of date — run %s",
			len(vars), envpkg.ManagedMarker, fix))
		return
	}
	if len(drift) > 0 {
		names := make([]string, 0, len(drift))
		for _, v := range drift {
			names = append(names, v.Name)
		}
		rep.Warn(fmt.Sprintf("%d contract variable(s) missing or stale at user scope (%s) — %s see none of them; run %s",
			len(drift), strings.Join(names, ", "), who, fix))
	}
	if len(retired) > 0 {
		// CLI-065 (#1363): a name the contract retired but the ownership marker
		// still lists — profile-less processes keep inheriting a value nothing
		// declares. The same command sweeps it; nothing else is touched.
		rep.Warn(fmt.Sprintf("%d retired name(s) still persisted at user scope (%s) — run %s to sweep them",
			len(retired), strings.Join(retired, ", "), fix))
	}
}

// persistRemedy names the command that repairs the user scope, and who reads
// it. On macOS the scope is the launchd session, which logout clears, so the
// fix is the converge step that also loads the login agent re-applying it.
func persistRemedy(goos string) (fix, who string) {
	if goos == "darwin" {
		return "`dotf converge --only env-persist`", "apps launched from the Dock, Finder or Spotlight"
	}
	return "`dotf env persist`", "profile-less processes such as Copilot tool calls"
}

// userEnvAdapter lets the doctor seam (a function) satisfy env.UserEnvReader —
// the reader alone, so the seam never has to fake a write.
type userEnvAdapter struct {
	get func(name string) (string, bool, error)
}

func (a userEnvAdapter) Get(name string) (string, bool, error) { return a.get(name) }
