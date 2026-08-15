// Package secrets implements the on-demand (JIT) secrets path of ADR-028: resolve
// the registry-mapped secrets and inject them into a single child process, never
// the ambient shell. It ports the resolution logic of scripts/load-secrets.sh into
// Go behind the `dotf secrets run` facade (#493). The age backend reads the local
// age store; the bw backend reads Bitwarden (ADR-028 Phase 3), dispatched per
// Entry.Backend through the Resolver seam.
package secrets

import (
	"os"
	"strings"
)

// Backend tags. Every place that reads Entry.Backend or Secret.Backend is a
// consumer of the same tagged union, and a bare literal at any of them is how a
// consumer gets taught about one variant and forgotten for another — the shape of
// BUG-077, where a bw entry's empty File read as a missing age blob and produced
// 56 FAILs telling the user to delete their DR floor (#969, #973, REFACTOR-012).
const (
	BackendAge        = "age"
	BackendAgeOffline = "age-offline"
	BackendBW         = "bw"
	// BackendDefault is the empty tag carried by a hand-built Entry{Var, File}
	// from a pre-bw caller. It is NOT a valid registry declaration — the parser
	// rejects it — but it IS resolvable, as age, for back-compat.
	BackendDefault = ""
)

// ValidBackends is the canonical set a registry may declare: the one answer to
// "which backends exist", consumed by the parser's validation and bound to the
// Loader's resolver map by TestResolversCoverEveryValidBackend. Adding a backend
// here without a resolver fails that test rather than surfacing as a runtime
// "unknown backend" on a user's machine — Go cannot make it a compile error
// without an exhaustive switch, which this repo's lint config does not enable.
func ValidBackends() []string {
	return []string{BackendAge, BackendAgeOffline, BackendBW}
}

// Entry is one flattened registry secret — the form Registry.Entries produces and
// the Loader resolves. Backend selects the Resolver; the source fields are
// backend-specific (File for age, Item+Field for bw). Two exposure shapes:
//   - env secret:  resolve the source into $Var.
//   - file secret: IsFile, resolve to Dest (Mode, 0600 default) and set $Var=Dest.
type Entry struct {
	Var      string      // env var name
	Backend  string      // age | age-offline | bw; "" resolves as age (back-compat)
	File     string      // age source: base name under sensitive/, without .secret.age
	Item     string      // bw source: Bitwarden item name/id
	Field    string      // bw source: field within the item
	IsFile   bool        // true for a file secret
	Dest     string      // materialization path (~ expanded); only when IsFile
	Mode     os.FileMode // file permissions (0 → 0600 default); only when IsFile
	Validate string      // optional liveness-check key carried from the registry (sync ci)
}

// SourceID is the entry's underlying-secret identity: the backend tag plus the
// source fields that backend actually populates. Consumers asking "are these two
// entries the same secret?" must compare this, never a single source field —
// Entry.File is "" for every bw entry, so a File comparison silently reports all
// of them as one secret (it defeated render's duplicate-var guard and collapsed
// the PAT check's dedupe, REFACTOR-012). The value is an opaque comparison key
// and never a display string; it carries no secret material, only source
// coordinates already present in the registry.
func (e Entry) SourceID() string {
	switch e.Backend {
	case BackendBW:
		return BackendBW + ":" + e.Item + "/" + e.Field
	default:
		// age, age-offline and the empty back-compat tag all source from the
		// age store, and the same base name under two age backends IS the same
		// blob — the tag selects the plane, not the file.
		return "age:" + e.File
	}
}

// SourceDisplay names the entry's source for a human reading an error: the age
// base name, or the bw item/field. Kept separate from SourceID because the
// comparison key's prefixes are noise in a message, and because an error that
// printed "" for a bw source (as the pre-REFACTOR-012 guard did) is unactionable.
func (e Entry) SourceDisplay() string {
	if e.Backend == BackendBW {
		if e.Field == "" {
			return e.Item
		}
		return e.Item + "/" + e.Field
	}
	return e.File
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
