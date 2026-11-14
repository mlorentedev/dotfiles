package lane

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// Standing is where a session's working directory stands relative to a
// repository's lanes.
type Standing int

const (
	// Inactive: the repository does not run lanes on this machine, or cwd is in
	// no repository. A session is told nothing.
	Inactive Standing = iota
	// InLane: cwd is in a persistent named lane of an active repository.
	InLane
	// ParkedLane: cwd is in a persistent lane nobody holds, whose lease says
	// PARKED or whose HEAD is detached. It is not a place to work from.
	ParkedLane
	// OutOfLane: cwd is in an active repository but not in a lane, such as its
	// main checkout or a worktree made by hand.
	OutOfLane
)

// Position is Where's answer. Lane is set InLane and ParkedLane, Branch only
// InLane.
type Position struct {
	Standing Standing
	Lane     string
	Branch   string
}

// Where reports where cwd stands. A repository is active when the activation
// record Activate writes exists in its git common dir; nothing else counts, as
// for the guard, so a lane with no record behind it reads as Inactive.
func Where(cwd string) (Position, error) {
	_, commonDir, err := worktreeGitDirs(cwd)
	if err != nil {
		return Position{}, err
	}
	if commonDir == "" {
		return Position{}, nil
	}
	if _, err := os.Stat(filepath.Join(commonDir, ActivationFile)); err != nil {
		if os.IsNotExist(err) {
			return Position{}, nil
		}
		return Position{}, fmt.Errorf("lane where: inspecting the activation record: %w", err)
	}
	name, ok, err := Identity(cwd)
	if err != nil {
		return Position{}, err
	}
	if !ok {
		return Position{Standing: OutOfLane}, nil
	}
	// `git branch --show-current` prints nothing on a detached HEAD and exits 0.
	out, err := exec.Command("git", "-C", cwd, "branch", "--show-current").Output()
	if err != nil {
		return Position{}, fmt.Errorf("lane where: reading the branch of %s: %w", cwd, err)
	}
	branch := strings.TrimSpace(string(out))
	meta, err := worktree.LoadMetadata(cwd)
	if err != nil {
		return Position{}, fmt.Errorf("lane where: reading the lease of %s: %w", cwd, err)
	}
	// A lane is held only while it has a branch and its lease says so: a lease
	// still PARKED beside a branch is a lane nobody claimed.
	if branch == "" || (meta != nil && strings.EqualFold(meta.State, string(worktree.StateParked))) {
		return Position{Standing: ParkedLane, Lane: name}, nil
	}
	return Position{Standing: InLane, Lane: name, Branch: branch}, nil
}

// Line is the one line a session is told, or "" when there is nothing to say.
func (p Position) Line() string {
	switch p.Standing {
	case InLane:
		return fmt.Sprintf("lane `%s` ACTIVE on `%s`", p.Lane, p.Branch)
	case ParkedLane:
		return fmt.Sprintf("lane `%s` is PARKED: start with `dotf lane switch <branch>`", p.Lane)
	case OutOfLane:
		return "this repository runs lanes: start with `dotf lane switch <branch>`"
	}
	return ""
}
