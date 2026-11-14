package lane

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

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

// Warning renders the finding as something to act on before it bites: the
// same shape as String, and no effect on the audit's verdict.
func (f Finding) Warning() string {
	return "WARN: " + f.Subject + ": " + f.Problem + ".\nFIX: " + f.Fix
}

// Audit is what AuditLanes answers. Managed is false when this machine never
// activated the repository's lanes (no activation record): the guard enforces
// nothing there, whatever marker a working tree carries, so there is nothing to
// compare. Initialised is false when the marker is present but no declared
// lane exists on this machine: the marker is committed and travels to every
// clone, the lanes are per machine, so that state is not drift. Warnings are
// what is not drift yet: states the guard will refuse, or sweep act on.
type Audit struct {
	Managed     bool
	Initialised bool
	Findings    []Finding
	Warnings    []Finding
}

// AuditLanes compares the marker of repoRoot with the lanes under
// worktreeRoot and reports every incoherence. A marker finding stops the
// audit: lanes cannot drift from a list that is not a list.
func AuditLanes(repoRoot, worktreeRoot string) (Audit, error) {
	absRepo, absWTRoot, err := auditRoots(repoRoot, worktreeRoot)
	if err != nil {
		return Audit{}, err
	}
	commonDir, err := resolveGitCommonDir(absRepo)
	if err != nil {
		return Audit{}, err
	}
	// The guard arms from the activation record and from nothing else (#3437),
	// so this is what makes a repository managed for the audit too.
	if _, err := os.Stat(filepath.Join(commonDir, ActivationFile)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Audit{}, nil
		}
		return Audit{}, fmt.Errorf("reading %s: %w", ActivationFile, err)
	}
	raw, err := os.ReadFile(filepath.Join(absRepo, filepath.FromSlash(LanesMarkerFile)))
	if errors.Is(err, fs.ErrNotExist) {
		// Active, but the main checkout sits on a commit that predates the
		// marker: the lanes were activated from the default branch and there is
		// no declared list here to compare them with.
		return Audit{Managed: true, Initialised: true, Warnings: []Finding{{LanesMarkerFile,
			"is absent from the main checkout, so the lanes were not compared with it",
			"switch the main checkout to the default branch (git switch main && git pull), then run dotf lane doctor again"}}}, nil
	}
	if err != nil {
		return Audit{}, fmt.Errorf("reading %s: %w", LanesMarkerFile, err)
	}
	a := Audit{Managed: true}
	marker, findings := parseMarker(raw)
	if len(findings) > 0 {
		a.Findings = findings
		return a, nil
	}
	records, unreadable, err := laneRecords(commonDir)
	if err != nil {
		return a, err
	}
	a.Findings = append(a.Findings, unreadable...)
	laneDir := filepath.Join(absWTRoot, filepath.Base(absRepo))
	a.Initialised = anyLaneExists(marker.CanonicalLanes, laneDir, records)
	if a.Initialised {
		for _, name := range marker.CanonicalLanes {
			a.Findings = append(a.Findings, auditLane(absRepo, commonDir, name, filepath.Join(laneDir, name), records[name])...)
		}
	}
	a.Findings = append(a.Findings, undeclaredRecords(marker.CanonicalLanes, records)...)
	a.Warnings, err = leaseWarnings(absRepo, records, time.Now())
	return a, err
}

// leaseWarnings walks the linked worktrees once and warns of the two states
// the guard or sweep will act on before anyone looks: a detached worktree with
// no lease, which guard rule 6 refuses and nothing in its path would have
// adopted (#3436), and a lease that has run out. Lanes are audited on their own
// for the leaseless case, so a worktree recorded as one is left out of it; a
// worktree whose lease cannot be read is a different defect and is left out of
// both.
func leaseWarnings(absRepo string, records map[string][]laneRecord, now time.Time) ([]Finding, error) {
	out, err := exec.Command("git", "-C", absRepo, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("listing the worktrees of %s: %w", absRepo, err)
	}
	linked, err := worktree.LinkedWorktrees(string(out))
	if err != nil {
		return nil, fmt.Errorf("reading the worktree list of %s: %w", absRepo, err)
	}
	var lanes []string
	for _, rs := range records {
		for _, r := range rs {
			lanes = append(lanes, r.path)
		}
	}
	var warns []Finding
	for _, w := range linked {
		if w.Prunable {
			continue
		}
		meta, lerr := worktree.LoadMetadata(w.Path)
		switch {
		case lerr != nil:
		case meta == nil:
			if w.Detached && !slices.ContainsFunc(lanes, func(p string) bool { return samePath(p, w.Path) }) {
				warns = append(warns, Finding{w.Path, "is detached and has no lease (" + worktree.MetadataFileName + "), so the guard refuses commits there while the lanes are active",
					"dotf worktree adopt " + w.Path + ", or run git switch -c <type>/<issue>-<slug> inside it"})
			}
		case leaseExpired(meta, now):
			warns = append(warns, expiredLease(w.Path, meta))
		}
	}
	return warns, nil
}

// leaseExpired is true for a lease that carries an expiry and whose expiry has
// passed. A parked lane's lease carries none (it is not a claim on anything),
// and a lease without one has not expired.
func leaseExpired(meta *worktree.Metadata, now time.Time) bool {
	return !meta.LeaseExpiresAt.IsZero() && !meta.LeaseExpiresAt.After(now) &&
		!strings.EqualFold(meta.State, string(worktree.StateParked))
}

// expiredLease names the command that ends the work the lease claimed: park
// for a lane, which stays where it is, done for any other worktree. There is no
// command that renews a lease; the work either ended or this is a new claim.
func expiredLease(path string, meta *worktree.Metadata) Finding {
	problem := "lease expired " + meta.LeaseExpiresAt.Format(time.RFC3339) + ", so sweep may reap or park it under whoever works there"
	if meta.Lane != "" {
		return Finding{path, problem, "dotf lane park " + meta.Lane + " once its branch landed, or dotf lane switch to claim the lane again"}
	}
	return Finding{path, problem, "dotf worktree done " + path + " once its branch landed"}
}

// auditRoots makes both roots absolute. An empty worktree root would resolve
// to the cwd and every FIX line would then point at the wrong path.
func auditRoots(repoRoot, worktreeRoot string) (absRepo, absWTRoot string, err error) {
	if strings.TrimSpace(worktreeRoot) == "" {
		return "", "", fmt.Errorf("WORKTREE_ROOT cannot be empty")
	}
	if absRepo, err = filepath.Abs(repoRoot); err == nil {
		absWTRoot, err = filepath.Abs(worktreeRoot)
	}
	return absRepo, absWTRoot, err
}

// anyLaneExists is the "initialised here" test: a declared lane either exists
// at its canonical path or is recorded by git somewhere.
func anyLaneExists(declared []string, laneDir string, records map[string][]laneRecord) bool {
	for _, name := range declared {
		if _, err := os.Stat(filepath.Join(laneDir, name)); err == nil || len(records[name]) > 0 {
			return true
		}
	}
	return false
}

// undeclaredRecords reports every worktree whose record names a lane the
// marker does not declare, sorted by lane name. Both repairs are legitimate,
// so the FIX names both and decides neither.
func undeclaredRecords(declared []string, records map[string][]laneRecord) []Finding {
	var out []Finding
	for _, name := range slices.Sorted(maps.Keys(records)) {
		if slices.Contains(declared, name) {
			continue
		}
		for _, r := range records[name] {
			out = append(out, Finding{name,
				fmt.Sprintf("the worktree at %s is recorded as this lane, which the marker does not declare", r.path),
				fmt.Sprintf("if it is not a lane, delete %s; if it is, declare it and run dotf lane init", filepath.Join(r.adminDir, LaneIdentityFile))})
		}
	}
	return out
}

// parseMarker validates the marker's shape and content, naming the field of
// every problem. A decode failure is the only finding it reports, since the
// rest would be read from a document that did not parse.
func parseMarker(raw []byte) (lanesMarker, []Finding) {
	var m lanesMarker
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return m, []Finding{{LanesMarkerFile, "does not parse: " + err.Error(), "run dotf lane init --write-marker to rewrite it, then commit the marker"}}
	}
	if dec.More() {
		return m, []Finding{{LanesMarkerFile, "carries trailing data after the object", "run dotf lane init --write-marker to rewrite it, then commit the marker"}}
	}
	var out []Finding
	if m.Version != 1 {
		out = append(out, Finding{"version", fmt.Sprintf("is %d, expected 1", m.Version), "run dotf lane init --write-marker to rewrite the marker"})
	}
	lanes := validateNames("canonical_lanes", m.CanonicalLanes, "run dotf lane init --write-marker to rewrite the marker")
	if len(lanes) == 0 && !sameSet(m.CanonicalLanes, CanonicalLanes) {
		lanes = append(lanes, Finding{"canonical_lanes", fmt.Sprintf("declares %v, which differs from the lanes this dotf provisions %v", m.CanonicalLanes, CanonicalLanes), "run dotf lane init --write-marker to rewrite the marker"})
	}
	out = append(out, lanes...)
	out = append(out, validateNames("third_party_roots", m.ThirdPartyRoots, "edit "+LanesMarkerFile+" by hand (directory names only), then commit it")...)
	return m, out
}

// validateNames checks a marker list of directory names: non-empty, no blank,
// padded, dot or duplicate entry, no separator and no drive prefix.
// Separators are the literal `/` and `\` because the marker is
// OS-independent: filepath.IsAbs would answer for the OS the audit runs on.
func validateNames(field string, names []string, fix string) []Finding {
	if len(names) == 0 {
		return []Finding{{field, "is empty", fix}}
	}
	var out []Finding
	seen := map[string]bool{}
	for _, n := range names {
		if problem := nameProblem(n, seen[n]); problem != "" {
			out = append(out, Finding{field, problem, fix})
		}
		seen[n] = true
	}
	return out
}

// nameProblem is what disqualifies n as a directory name in the marker, or
// empty when nothing does.
func nameProblem(n string, duplicate bool) string {
	switch {
	case strings.TrimSpace(n) == "":
		return "carries an empty entry"
	case strings.TrimSpace(n) != n:
		return fmt.Sprintf("entry %q carries leading or trailing whitespace", n)
	case n == "." || n == "..":
		return fmt.Sprintf("entry %q is not a directory name", n)
	case hasDrivePrefix(n):
		return fmt.Sprintf("entry %q starts with a drive prefix", n)
	case strings.ContainsAny(n, `/\`):
		return fmt.Sprintf("entry %q carries a path separator", n)
	case duplicate:
		return fmt.Sprintf("entry %q is a duplicate", n)
	}
	return ""
}

// hasDrivePrefix is true for `C:` and `c:` openings, checked before the
// separator rule so `C:\hosts` is reported as the drive it names.
func hasDrivePrefix(n string) bool {
	return len(n) >= 2 && n[1] == ':' && n[0]|0x20 >= 'a' && n[0]|0x20 <= 'z'
}

func sameSet(a, b []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(a)), slices.Sorted(slices.Values(b)))
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
		return nil, nil, fmt.Errorf("reading the worktree registry of %s: %w", commonDir, err)
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
		if !ok {
			continue
		}
		// The record reads fine but git's own pointer to the worktree does
		// not: without this the lane fell through to "no identity record",
		// a diagnosis of the wrong file (#3337). The empty path marks the
		// entry as reported so auditLane does not diagnose it again.
		gitdir, gerr := os.ReadFile(filepath.Join(adminDir, "gitdir"))
		if gerr != nil {
			unreadable = append(unreadable, Finding{adminDir, "the gitdir file there is missing or not readable, so git cannot locate the worktree of lane " + name,
				"git worktree prune if the worktree is gone, or git worktree repair <path> if it was moved; then run dotf lane doctor again"})
			out[name] = append(out[name], laneRecord{adminDir: adminDir})
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
	records, broken := located(records)
	switch {
	case len(records) == 0 && broken:
		return nil // already reported by laneRecords: the admin dir's gitdir is unreadable
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
		// A lane wiped and recreated by hand keeps its record and its
		// registration; without this check it would read as a missing lease
		// and the FIX would name a command that refuses the directory.
		if err := assertWorktreeOf(absRepo, commonDir, expected); err != nil {
			return []Finding{{name, err.Error(), strangerFix(err, r.adminDir)}}
		}
		return auditLease(name, expected)
	}
	if _, err := os.Stat(expected); err != nil {
		return []Finding{{name, "is absent at " + expected, "dotf lane init"}}
	}
	if err := assertWorktreeOf(absRepo, commonDir, expected); err != nil {
		return []Finding{{name, err.Error(), strangerFix(err, "")}}
	}
	return []Finding{{name, "the worktree at " + expected + " carries no identity record, so guard rule 5b does not fire there", "dotf lane init"}}
}

// located drops the registrations laneRecords already reported (an unreadable
// gitdir leaves no path to audit) and says whether it dropped any, so a broken
// twin neither pads the duplicate list with a blank path nor gets diagnosed
// twice.
func located(records []laneRecord) ([]laneRecord, bool) {
	kept := records[:0:0]
	for _, r := range records {
		if r.path != "" {
			kept = append(kept, r)
		}
	}
	return kept, len(kept) != len(records)
}

// strangerDirFix repairs a directory at a lane path that is not this
// repository's worktree. The prune is not optional: git keeps the lane
// registered after its directory is wiped, and `git worktree add` refuses a
// registered path, so `dotf lane init` alone fails after the move (measured
// 2026-11-11).
const strangerDirFix = "move the directory aside and run git worktree prune, then dotf lane init"

// foreignLaneFix is the repair when the directory is a worktree that is not
// this repository's: nothing there is ours to move, prune or repair. Measured
// 2026-11-11: from the repository whose registration is stale, `git worktree
// prune` keeps the entry while the other's .git file exists, `remove --force`
// refuses, and `git worktree repair` rewrites the other repository's .git
// file to point here, hijacking its worktree. Only the stale admin dir is
// ours, and only a hand delete drops it.
const foreignLaneFix = "the worktree there is not this repository's: leave it to its owner, and give this clone a different directory name before dotf lane init (#3318)"

// strangerFix picks the repair for an assertWorktreeOf refusal: the move and
// prune only for a plain directory; adminDir, when set, is this repository's
// stale registration of the path.
func strangerFix(err error, adminDir string) string {
	if errors.Is(err, errNotAWorktree) {
		return strangerDirFix
	}
	if adminDir != "" {
		return "delete this repository's stale registration " + adminDir + " by hand; " + foreignLaneFix
	}
	return foreignLaneFix
}

func auditLease(name, path string) []Finding {
	meta, err := worktree.LoadMetadata(path)
	switch {
	case err != nil:
		return []Finding{{name, "lease unreadable: " + err.Error(), "dotf lane init"}}
	case meta == nil:
		return []Finding{{name, "has no lease (" + worktree.LeaseFileName + " in its git admin dir), so the guard refuses commits there", "dotf lane init"}}
	case meta.Lane != name:
		return []Finding{{name, fmt.Sprintf("lease names lane %q", meta.Lane), "dotf lane init"}}
	case !strings.EqualFold(meta.State, string(worktree.StateActive)) && !strings.EqualFold(meta.State, string(worktree.StateParked)):
		return []Finding{{name, fmt.Sprintf("lease state %q is neither ACTIVE nor PARKED", meta.State), "dotf lane park " + name + ", or dotf lane init"}}
	}
	return nil
}
