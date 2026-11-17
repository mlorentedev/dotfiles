package cmd

import (
	"fmt"
	"io"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// laneHint is the line `worktree add` prints after it has made a worktree in a
// repository that runs lanes (HARNESS-195 AC4), or "" when it runs none. It
// lives here and not in package worktree because lane imports worktree: only
// this layer sees both. It is not in AddWithRunner either, so `spec review`'s
// detached worktrees, which never go through add, cannot print it.
//
// A directory Where cannot read is reported on errOut and prints no hint: add
// has already succeeded, and a hint must not turn that into a failure.
func laneHint(cwd string, errOut io.Writer) string {
	pos, err := lane.Where(cwd)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "lane hint skipped: %v\n", err)
		return ""
	}
	if pos.Standing == lane.Inactive {
		return ""
	}
	return "this repository runs lanes: start work with `dotf lane switch <branch>`, not a hand-made worktree"
}
