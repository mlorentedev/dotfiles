// Package secrets implements the on-demand (JIT) secrets path of ADR-028: decrypt
// the age-mapped secrets and inject them into a single child process, never the
// ambient shell. It ports the resolution logic of scripts/load-secrets.sh into Go
// behind the `dotf secrets run` facade (#493), over the existing age store with
// no migration.
package secrets

import (
	"strings"
)

// Entry is one flattened registry secret — the form Registry.Entries produces and
// the age Loader consumes. Two shapes:
//   - env secret:  decrypt sensitive/<File>.secret.age into $Var.
//   - file secret: IsFile, decrypt to Dest (0600) and set $Var=Dest.
type Entry struct {
	Var    string // env var name
	File   string // base name under sensitive/, without the .secret.age suffix
	IsFile bool   // true for a file secret
	Dest   string // materialization path (~ expanded); only when IsFile
}

// expandHome rewrites a leading ~ (or ~/...) to home, leaving other paths intact.
func expandHome(p, home string) string {
	switch {
	case p == "~":
		return home
	case strings.HasPrefix(p, "~/"):
		return home + p[1:]
	default:
		return p
	}
}
