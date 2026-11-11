package lane

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// Finding is one incoherence between the marker and the lanes: what is wrong,
// about which field or lane, and the command that repairs it. The audit never
// repairs anything itself.
type Finding struct {
	Subject string
	Problem string
	Fix     string
}

// String renders the finding in the ERROR/FIX form the guard and every
// agent-facing message of this CLI use.
func (f Finding) String() string {
	return "ERROR: " + f.Subject + ": " + f.Problem + ".\nFIX: " + f.Fix
}

// Audit is what AuditLanes answers. Managed is false when the repository has
// no marker. Initialised is false when the marker is present but no declared
// lane exists on this machine: the marker is committed and travels to every
// clone, the lanes are per machine, so that state is not drift.
type Audit struct {
	Managed     bool
	Initialised bool
	Findings    []Finding
}

// AuditLanes compares the marker of repoRoot with the lanes under
// worktreeRoot and reports every incoherence. A marker finding stops the
// audit: lanes cannot drift from a list that is not a list.
func AuditLanes(repoRoot, worktreeRoot string) (Audit, error) {
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return Audit{}, err
	}
	// An empty root would resolve to the cwd and every FIX line would point
	// at the wrong path.
	if strings.TrimSpace(worktreeRoot) == "" {
		return Audit{}, fmt.Errorf("lane doctor: WORKTREE_ROOT cannot be empty")
	}
	absWTRoot, err := filepath.Abs(worktreeRoot)
	if err != nil {
		return Audit{}, err
	}
	raw, err := os.ReadFile(filepath.Join(absRepo, filepath.FromSlash(LanesMarkerFile)))
	if errors.Is(err, fs.ErrNotExist) {
		return Audit{}, nil
	}
	if err != nil {
		return Audit{}, fmt.Errorf("lane doctor: reading %s: %w", LanesMarkerFile, err)
	}
	a := Audit{Managed: true}
	marker, findings := parseMarker(raw)
	if len(findings) > 0 {
		a.Findings = findings
		return a, nil
	}
	commonDir, err := resolveGitCommonDir(absRepo)
	if err != nil {
		return a, err
	}
	records, unreadable, err := laneRecords(commonDir)
	if err != nil {
		return a, err
	}
	a.Findings = append(a.Findings, unreadable...)
	laneDir := filepath.Join(absWTRoot, filepath.Base(absRepo))
	for _, name := range marker.CanonicalLanes {
		if _, serr := os.Stat(filepath.Join(laneDir, name)); serr == nil || len(records[name]) > 0 {
			a.Initialised = true
		}
	}
	if a.Initialised {
		for _, name := range marker.CanonicalLanes {
			a.Findings = append(a.Findings, auditLane(absRepo, commonDir, name, filepath.Join(laneDir, name), records[name])...)
		}
	}
	names := make([]string, 0, len(records))
	for name := range records {
		if !slices.Contains(marker.CanonicalLanes, name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		for _, r := range records[name] {
			a.Findings = append(a.Findings, Finding{name,
				fmt.Sprintf("the worktree at %s is recorded as this lane, which the marker does not declare", r.path),
				fmt.Sprintf("if it is not a lane, delete %s; if it is, declare it and run dotf lane init", filepath.Join(r.adminDir, LaneIdentityFile))})
		}
	}
	return a, nil
}

// parseMarker validates the marker's shape and content, naming the field of
// every problem. A decode failure is the only finding it reports, since the
// rest would be read from a document that did not parse.
func parseMarker(raw []byte) (lanesMarker, []Finding) {
	var m lanesMarker
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return m, []Finding{{LanesMarkerFile, "does not parse: " + err.Error(), "run dotf lane init to rewrite it, then commit the marker"}}
	}
	var out []Finding
	if m.Version != 1 {
		out = append(out, Finding{"version", fmt.Sprintf("is %d, expected 1", m.Version), "run dotf lane init to rewrite the marker"})
	}
	lanes := validateNames("canonical_lanes", m.CanonicalLanes, "run dotf lane init to rewrite the marker")
	if len(lanes) == 0 && !sameSet(m.CanonicalLanes, CanonicalLanes) {
		lanes = append(lanes, Finding{"canonical_lanes", fmt.Sprintf("declares %v, which differs from the lanes this dotf provisions %v", m.CanonicalLanes, CanonicalLanes), "run dotf lane init to rewrite the marker"})
	}
	out = append(out, lanes...)
	out = append(out, validateNames("third_party_roots", m.ThirdPartyRoots, "edit "+LanesMarkerFile+" by hand (directory names only), then run dotf lane init")...)
	return m, out
}

// validateNames checks a marker list of directory names: non-empty, no blank
// or duplicate entry, no separator and no drive prefix. Separators are the
// literal `/` and `\` because the marker is OS-independent: filepath.IsAbs
// would answer for the OS the audit runs on.
func validateNames(field string, names []string, fix string) []Finding {
	if len(names) == 0 {
		return []Finding{{field, "is empty", fix}}
	}
	var out []Finding
	seen := map[string]bool{}
	for _, n := range names {
		switch {
		case strings.TrimSpace(n) == "":
			out = append(out, Finding{field, "carries an empty entry", fix})
		case len(n) >= 2 && n[1] == ':' && (n[0]|0x20 >= 'a' && n[0]|0x20 <= 'z'):
			out = append(out, Finding{field, fmt.Sprintf("entry %q starts with a drive prefix", n), fix})
		case strings.ContainsAny(n, `/\`):
			out = append(out, Finding{field, fmt.Sprintf("entry %q carries a path separator", n), fix})
		case seen[n]:
			out = append(out, Finding{field, fmt.Sprintf("entry %q is a duplicate", n), fix})
		}
		seen[n] = true
	}
	return out
}

func sameSet(a, b []string) bool {
	sa, sb := slices.Clone(a), slices.Clone(b)
	sort.Strings(sa)
	sort.Strings(sb)
	return slices.Equal(sa, sb)
}

// laneRecord is one worktree admin dir under the common git dir that carries
// an identity record, with the worktree path git registered for it.
type laneRecord struct {
	adminDir string
	path     string
}

// laneRecords reads git's own registry of linked worktrees
// (<common dir>/worktrees/*/), keyed by recorded lane name, and reports a
// record that exists but cannot be read as a finding, as the guard fails
// closed on it. This is the direction from disk to marker: a lane moved by
// hand keeps its admin dir, its record and a gitdir pointing at the old path,
// which `git worktree list` only reports as prunable and a path lookup would
// report as absent. gitdir is relative to the admin dir under
// worktree.useRelativePaths (git 2.48+), absolute otherwise.
func laneRecords(commonDir string) (map[string][]laneRecord, []Finding, error) {
	entries, err := os.ReadDir(filepath.Join(commonDir, "worktrees"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("lane doctor: reading the worktree registry of %s: %w", commonDir, err)
	}
	out := map[string][]laneRecord{}
	var unreadable []Finding
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		adminDir := filepath.Join(commonDir, "worktrees", e.Name())
		name, ok, rerr := readIdentityRecord(adminDir)
		if rerr != nil {
			unreadable = append(unreadable, Finding{adminDir, "the identity record there is not readable, so the guard refuses commits in that worktree", "restore it with dotf lane init"})
			continue
		}
		gitdir, gerr := os.ReadFile(filepath.Join(adminDir, "gitdir"))
		if !ok || gerr != nil {
			continue
		}
		p := filepath.FromSlash(strings.TrimSpace(string(gitdir)))
		if !filepath.IsAbs(p) {
			p = filepath.Join(adminDir, p)
		}
		out[name] = append(out[name], laneRecord{adminDir: adminDir, path: filepath.Dir(filepath.Clean(p))})
	}
	return out, unreadable, nil
}

// auditLane checks one declared lane: where git registered it, whether that is
// its canonical path, and whether the worktree there is this repository's and
// carries its record and lease. One finding per lane, the first that applies.
func auditLane(absRepo, commonDir, name, expected string, records []laneRecord) []Finding {
	switch {
	case len(records) > 1:
		paths := make([]string, 0, len(records))
		for _, r := range records {
			paths = append(paths, r.path)
		}
		return []Finding{{name, "is recorded in more than one worktree: " + strings.Join(paths, ", "), "keep the one at " + expected + " and remove the others with git worktree remove <path>"}}
	case len(records) == 1:
		r := records[0]
		if st, err := os.Stat(r.path); err != nil || !st.IsDir() {
			return []Finding{{name, fmt.Sprintf("is registered by git at %s, which no longer exists (moved or deleted by hand)", r.path),
				"git worktree repair <new path> if it was moved, or git worktree prune if it was deleted; then run dotf lane doctor again"}}
		}
		if !samePath(r.path, expected) {
			return []Finding{{name, fmt.Sprintf("is at %s, not at %s", r.path, expected), fmt.Sprintf("git worktree move %s %s", r.path, expected)}}
		}
		return auditLease(name, expected)
	}
	if _, err := os.Stat(expected); err != nil {
		return []Finding{{name, "is absent at " + expected, "dotf lane init"}}
	}
	if err := assertWorktreeOf(absRepo, commonDir, expected); err != nil {
		return []Finding{{name, err.Error(), "move the directory aside, then dotf lane init"}}
	}
	return []Finding{{name, "the worktree at " + expected + " carries no identity record, so guard rule 5b does not fire there", "dotf lane init"}}
}

func auditLease(name, path string) []Finding {
	meta, err := worktree.LoadMetadata(path)
	switch {
	case err != nil:
		return []Finding{{name, "lease unreadable: " + err.Error(), "dotf lane init"}}
	case meta == nil:
		return []Finding{{name, "has no lease (" + worktree.MetadataFileName + "), so the guard refuses commits there", "dotf lane init"}}
	case meta.Lane != name:
		return []Finding{{name, fmt.Sprintf("lease names lane %q", meta.Lane), "dotf lane init"}}
	case !strings.EqualFold(meta.State, string(worktree.StateActive)) && !strings.EqualFold(meta.State, string(worktree.StateParked)):
		return []Finding{{name, fmt.Sprintf("lease state %q is neither ACTIVE nor PARKED", meta.State), "dotf lane park " + name + ", or dotf lane init"}}
	}
	return nil
}
