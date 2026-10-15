package lane

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Runner runs a command and returns its stdout. It is the one seam over the
// outside world (herdr, gh, git) so every parser and every sequence is tested
// against recorded output instead of a live tool.
type Runner func(name string, args ...string) (string, error)

// ExecRunner is the production Runner. A failure carries the tool's stderr, or
// its stdout when stderr is empty: herdr reports its errors as JSON on stdout.
func ExecRunner(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return stdout.String(), fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, msg)
	}
	return stdout.String(), nil
}
