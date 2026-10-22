package doctor

import (
	"strings"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

// machineIdentityRemedy is the one command that declares an identity.
const machineIdentityRemedy = "dotf env identity set <ID>"

// checkMachineIdentity (#1236, #1547) reports whether this machine declares who
// it is. Without an id every non-local pool is denied (ADR-032 §8), which is the
// right default and a poor way to find out: the first dispatch refuses, and it
// reads like the dispatch feature is broken. Read-only, as every doctor check.
func checkMachineIdentity(sys *System, rep *Report) {
	rep.Section("Machine identity")
	p, err := envpkg.LoadMachinePolicy(envpkg.MachinePath(sys.home()))
	if err != nil {
		rep.Warn(err.Error())
		return
	}
	if !p.Identified {
		rep.Warn("this machine has not declared an identity, so dispatch denies every non-local pool: " + machineIdentityRemedy)
		return
	}
	msg := "declared as " + p.ID
	if len(p.Deny) > 0 {
		msg += "; denies " + strings.Join(p.Deny, ", ")
	}
	rep.Pass(msg)
}
