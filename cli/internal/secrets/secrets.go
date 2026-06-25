// Package secrets implements the on-demand (JIT) secrets path of ADR-028: decrypt
// the age-mapped secrets and inject them into a single child process, never the
// ambient shell. It ports the resolution logic of scripts/load-secrets.sh into Go
// behind the `dotf secrets run` facade (#493), over the existing age store with
// no migration.
package secrets

import (
	"bufio"
	"io"
	"strings"
)

// Entry is one parsed env-mapping.conf line. Two forms:
//   - VAR=file         → an env secret: decrypt sensitive/<file>.secret.age into $VAR.
//   - @VAR=file>dest    → a file secret: decrypt to dest (0600) and set $VAR=dest.
type Entry struct {
	Var    string // env var name (the @ prefix stripped)
	File   string // base name under sensitive/, without the .secret.age suffix
	IsFile bool   // true for the @VAR=file>dest form
	Dest   string // materialization path (~ expanded); only when IsFile
}

// ParseMapping parses env-mapping.conf content into entries. Comment (#) and blank
// lines are skipped; a line with no '=' is skipped rather than fatal (lenient,
// matching load-secrets.sh). home expands a leading ~ in file-secret destinations.
func ParseMapping(r io.Reader, home string) ([]Entry, error) {
	var entries []Entry
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, val, _ := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" || val == "" {
			continue
		}

		if strings.HasPrefix(key, "@") {
			file, dest, _ := strings.Cut(val, ">")
			entries = append(entries, Entry{
				Var:    strings.TrimPrefix(key, "@"),
				File:   strings.TrimSpace(file),
				IsFile: true,
				Dest:   expandHome(strings.TrimSpace(dest), home),
			})
			continue
		}
		entries = append(entries, Entry{Var: key, File: val})
	}
	return entries, sc.Err()
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
