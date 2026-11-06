package env

import (
	"fmt"
	"strings"
)

// UserPathStore reads and writes the user scope's Path value as stored, never
// expanded: on Windows the HKCU\Environment `Path`, which a process started
// with no profile (a Scheduled Task, Explorer, `pwsh -NoProfile`) inherits
// ahead of nothing else of ours. A write keeps the value's REG_EXPAND_SZ type,
// so an entry like `%USERPROFILE%\go\bin` stays a variable reference.
type UserPathStore interface {
	ReadPath() (string, error)
	WritePath(value string) error
}

// ManagedPathMarker names the Path entries dotf added, joined by ";". Path is
// the user's value: dotf owns entries inside it, not the value, so a retired
// entry is removed only when this record says dotf put it there, the same
// bound ManagedMarker gives the variables (CLI-065).
const ManagedPathMarker = "DOTF_MANAGED_PATH"

// PathPlan is what PersistPath would write, or wrote.
type PathPlan struct {
	Next        string // the Path value after the merge
	Changed     bool   // Next differs from the stored Path, entry for entry
	MarkerStale bool   // the marker does not name want
}

// MergePath puts want at the head of current, in order, and removes the
// entries of owned that want no longer names. Every other entry keeps its
// position and spelling. Entries compare as Windows compares directories:
// case-insensitive, a trailing separator ignored, and after expand, so a
// stored `%USERPROFILE%\.local\bin` is the same entry as its expansion; a
// matching entry keeps its stored spelling when it moves to the head.
// changed is false when the result has the same entries as current, so empty
// segments alone never cause a write.
func MergePath(current string, want, owned []string, expand func(string) string) (string, bool) {
	key := func(s string) string {
		return strings.ToLower(strings.TrimRight(expand(strings.TrimSpace(s)), `\/`))
	}
	wanted := make(map[string]int, len(want))
	for i, w := range want {
		if _, dup := wanted[key(w)]; !dup {
			wanted[key(w)] = i
		}
	}
	retired := map[string]bool{}
	for _, o := range owned {
		if _, still := wanted[key(o)]; !still {
			retired[key(o)] = true
		}
	}

	head := make([]string, len(want))
	copy(head, want)
	placed := map[string]bool{}
	var rest, entries []string
	for _, e := range strings.Split(current, ";") {
		k := key(e)
		if k == "" {
			continue
		}
		entries = append(entries, e)
		if i, ok := wanted[k]; ok {
			if !placed[k] {
				head[i] = e
				placed[k] = true
			}
			continue
		}
		if !retired[k] {
			rest = append(rest, e)
		}
	}
	next := make([]string, 0, len(want)+len(rest))
	seen := map[string]bool{}
	for _, h := range head {
		if !seen[key(h)] {
			seen[key(h)] = true
			next = append(next, h)
		}
	}
	next = append(next, rest...)
	if strings.Join(next, ";") == strings.Join(entries, ";") {
		return current, false
	}
	return strings.Join(next, ";"), true
}

// PlanPath reports what PersistPath would write, and writes nothing.
func PlanPath(want []string, reader UserEnvReader, path UserPathStore, expand func(string) string) (PathPlan, error) {
	marker, _, err := reader.Get(ManagedPathMarker)
	if err != nil {
		return PathPlan{}, fmt.Errorf("read persisted %s: %w", ManagedPathMarker, err)
	}
	current, err := path.ReadPath()
	if err != nil {
		return PathPlan{}, fmt.Errorf("read the user Path: %w", err)
	}
	next, changed := MergePath(current, want, ParseMarker(marker), expand)
	return PathPlan{Next: next, Changed: changed, MarkerStale: marker != strings.Join(want, markerSep)}, nil
}

// PersistPath merges want into the user Path and records it as owned. The
// Path is written before the marker: a failure between them leaves a marker
// naming the old entries, which the next run still reads correctly.
func PersistPath(want []string, store UserEnvStore, path UserPathStore, expand func(string) string) (PathPlan, error) {
	plan, err := PlanPath(want, store, path, expand)
	if err != nil {
		return plan, err
	}
	if plan.Changed {
		if err := path.WritePath(plan.Next); err != nil {
			return plan, fmt.Errorf("write the user Path: %w", err)
		}
	}
	if plan.MarkerStale {
		if err := store.Set(ManagedPathMarker, strings.Join(want, markerSep)); err != nil {
			return plan, fmt.Errorf("persist %s: %w", ManagedPathMarker, err)
		}
	}
	return plan, nil
}
