package lane

import (
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
// it, and only its existence counts: nothing parses the content.
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
	record := filepath.Join(commonDir, ActivationFile)
	act := Activation{Active: true, Ref: ref}
	if _, err := os.Stat(record); err != nil {
		act.Changes++
	}
	if dryRun {
		act.Changes += missingLanes(mainRoot, worktreeRoot)
		return act, nil
	}
	created, err := InitLanes(mainRoot, worktreeRoot)
	if err != nil {
		return act, err
	}
	act.Changes += created
	if _, err := os.Stat(record); err == nil {
		return act, nil
	}
	body := "dotf lane activation record: written by `dotf converge`, read by the lane guard; existence is the signal\n"
	if err := os.WriteFile(record, []byte(body), 0o644); err != nil {
		return act, fmt.Errorf("lane activate: writing %s: %w", record, err)
	}
	return act, nil
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

// missingLanes counts the canonical lanes InitLanes would have to create: no
// worktree at the lane path, or one that carries no lease.
func missingLanes(mainRoot, worktreeRoot string) int {
	laneDir := filepath.Join(worktreeRoot, filepath.Base(filepath.Clean(mainRoot)))
	missing := 0
	for _, name := range CanonicalLanes {
		if meta, err := worktree.LoadMetadata(filepath.Join(laneDir, name)); err != nil || meta == nil {
			missing++
		}
	}
	return missing
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
