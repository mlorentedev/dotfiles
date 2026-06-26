// Package secrets implements the on-demand (JIT) secrets path of ADR-028: resolve
// the registry-mapped secrets and inject them into a single child process, never
// the ambient shell. It ports the resolution logic of scripts/load-secrets.sh into
// Go behind the `dotf secrets run` facade (#493). The age backend reads the local
// age store; the bw backend reads Bitwarden (ADR-028 Phase 3), dispatched per
// Entry.Backend through the Resolver seam.
package secrets

import (
	"strings"
)

// Entry is one flattened registry secret — the form Registry.Entries produces and
// the Loader resolves. Backend selects the Resolver; the source fields are
// backend-specific (File for age, Item+Field for bw). Two exposure shapes:
//   - env secret:  resolve the source into $Var.
//   - file secret: IsFile, resolve to Dest (0600) and set $Var=Dest.
type Entry struct {
	Var     string // env var name
	Backend string // age | age-offline | bw; "" resolves as age (back-compat)
	File    string // age source: base name under sensitive/, without .secret.age
	Item    string // bw source: Bitwarden item name/id
	Field   string // bw source: field within the item
	IsFile  bool   // true for a file secret
	Dest    string // materialization path (~ expanded); only when IsFile
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
