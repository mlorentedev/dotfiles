package lane

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// ActivationFile is the record that a machine runs this repository's lanes. It
// lives in the repository's git common dir: not committed and in no working
// tree, so `git clean -fdx` cannot remove it (lesson 597). Only Activate writes
// it. Its existence is what arms the guard; its content is the marker as the
// default branch carries it, because the guard reads the declared third-party
// roots from the record and from no working tree (#3437).
const ActivationFile = "dotf-lanes-active"

// defaultBranchRefs is the order the remote's default branch is looked for in.
// The first that resolves is the default branch; a later one is never a
// fallback for a marker the first lacks, and a local branch is never one.
var defaultBranchRefs = []string{"origin/HEAD", "origin/main", "origin/master"}

// Activation is what Activate found, and did or, under a dry run, would do.
type Activation struct {
	Active  bool   // the default branch carries the marker
	Ref     string // the ref the marker was read from, when Active
	Changes int    // lanes created plus the record written
}

// Activate makes the repository of repoRoot run lanes on this machine when its
// default branch carries the committed marker: it creates the three lanes
// (InitLanes) and writes the activation record. A marker anywhere else, in a
// working tree or on a commit the remote has not got, activates nothing.
// Idempotent: a second run reports Changes == 0. It reads the remote-tracking
// ref as it stands and fetches nothing, so a plan never touches the network.
func Activate(repoRoot, worktreeRoot string, dryRun bool) (Activation, error) {
	mainRoot, err := worktree.ResolveMainRepoRoot(repoRoot)
	if err != nil {
		return Activation{}, fmt.Errorf("lane activate: %w", err)
	}
	ref := committedMarkerRef(mainRoot)
	if ref == "" {
		return Activation{}, nil
	}
	commonDir, err := resolveGitCommonDir(mainRoot)
	if err != nil {
		return Activation{}, err
	}
	absWT, err := filepath.Abs(worktreeRoot)
	if err != nil || strings.TrimSpace(worktreeRoot) == "" {
		return Activation{}, fmt.Errorf("lane activate: WORKTREE_ROOT cannot be empty")
	}
	act := Activation{Active: true, Ref: ref}
	// Refuse before counting or writing anything, in a plan as in an apply:
	// the same refusal InitLanes would make, reported before it makes a lane.
	laneDir := filepath.Join(absWT, filepath.Base(filepath.Clean(mainRoot)))
	for _, name := range CanonicalLanes {
		if err := refuseStranger(mainRoot, commonDir, filepath.Join(laneDir, name)); err != nil {
			return act, &RefusedError{Err: err}
		}
	}
	body, err := committedMarker(mainRoot, ref)
	if err != nil {
		return act, err
	}
	record := filepath.Join(commonDir, ActivationFile)
	stale := recordStale(record, body)
	if stale {
		act.Changes++
	}
	if dryRun {
		act.Changes += pendingLanes(laneDir)
		return act, nil
	}
	created, err := InitLanes(mainRoot, absWT)
	if err != nil {
		return act, err
	}
	act.Changes += created
	if !stale {
		return act, nil
	}
	if err := os.WriteFile(record, body, 0o644); err != nil {
		return act, fmt.Errorf("lane activate: writing %s: %w", record, err)
	}
	return act, nil
}

// committedMarker reads the marker as ref carries it.
func committedMarker(mainRoot, ref string) ([]byte, error) {
	body, err := exec.Command("git", "-C", mainRoot, "show", ref+":"+LanesMarkerFile).Output()
	if err != nil {
		return nil, fmt.Errorf("lane activate: reading %s from %s: %w", LanesMarkerFile, ref, err)
	}
	return body, nil
}

// recordStale is true when the record is missing, or holds something other than
// body: a marker edited on the default branch, or a record an earlier build
// wrote without one.
func recordStale(record string, body []byte) bool {
	have, err := os.ReadFile(record)
	return err != nil || !bytes.Equal(have, body)
}

// committedMarkerRef names the default-branch ref whose tree carries the
// marker, or "" when the default branch does not, or there is none.
func committedMarkerRef(mainRoot string) string {
	for _, ref := range defaultBranchRefs {
		if exec.Command("git", "-C", mainRoot, "rev-parse", "--verify", "--quiet", ref).Run() != nil {
			continue
		}
		if exec.Command("git", "-C", mainRoot, "cat-file", "-e", ref+":"+LanesMarkerFile).Run() == nil {
			return ref
		}
		return ""
	}
	return ""
}

// pendingLanes counts the canonical lanes InitLanes would write to, by the
// same conditions provisionLane writes on, so a plan predicts its apply.
func pendingLanes(laneDir string) int {
	pending := 0
	for _, name := range CanonicalLanes {
		if laneNeedsWork(filepath.Join(laneDir, name), name) {
			pending++
		}
	}
	return pending
}

// WriteLanesMarker writes .dotf/lanes.json into repo, for the one commit that
// declares the repository a lane repository. It arms nothing by itself: a
// machine activates through Activate, from the committed marker.
func WriteLanesMarker(repo string) (bool, error) {
	abs, err := filepath.Abs(strings.TrimSpace(repo))
	if err != nil {
		return false, err
	}
	return writeLanesMarker(abs)
}

// RefusedError says the lanes cannot be provisioned here because something
// else occupies a lane path (#3318). It is one machine's collision, not a
// failure of Activate, so a caller can report it and carry on.
type RefusedError struct{ Err error }

func (e *RefusedError) Error() string { return e.Err.Error() }
func (e *RefusedError) Unwrap() error { return e.Err }
