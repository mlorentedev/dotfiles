package cmd

import (
	"fmt"
	"io"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// laneLine is the line `mem resume` and `mem session-start` print to say where
// cwd stands in a repository that runs lanes (HARNESS-195), or "" when it runs
// none. A directory Where cannot read is reported on errOut and prints no line:
// an unanswerable question is never worded as "no lanes here", and a session
// start must not fail over it.
func laneLine(cwd string, errOut io.Writer) string {
	pos, err := lane.Where(cwd)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "lane line skipped: %v\n", err)
		return ""
	}
	return pos.Line()
}
