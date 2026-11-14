package lane

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	// OutOfLane: cwd is in an active repository but not in a lane, such as its
	// main checkout or a worktree made by hand.
	OutOfLane
)

// Position is Where's answer. Lane and Branch are set only InLane, and Branch
// is "" for a lane parked on a detached HEAD.
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
	// `git branch --show-current` prints nothing on a detached HEAD, which is
	// the state of a parked lane, and exits 0.
	out, err := exec.Command("git", "-C", cwd, "branch", "--show-current").Output()
	if err != nil {
		return Position{}, fmt.Errorf("lane where: reading the branch of %s: %w", cwd, err)
	}
	return Position{Standing: InLane, Lane: name, Branch: strings.TrimSpace(string(out))}, nil
}

// Line is the one line a session is told, or "" when there is nothing to say.
func (p Position) Line() string {
	switch p.Standing {
	case InLane:
		branch := p.Branch
		if branch == "" {
			branch = "(detached)"
		}
		return fmt.Sprintf("lane `%s` ACTIVE on `%s`", p.Lane, branch)
	case OutOfLane:
		return "this repository runs lanes: start with `dotf lane switch <branch>`"
	}
	return ""
}
