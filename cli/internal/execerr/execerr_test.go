package execerr

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// failing runs a real command that writes msg to stderr and exits 3, so the
// test exercises the *exec.ExitError that Output produces, not a hand-built one.
func failing(t *testing.T, msg string) error {
	t.Helper()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "echo "+msg+" 1>&2 & exit 3")
	} else {
		cmd = exec.Command("sh", "-c", "echo '"+msg+"' >&2; exit 3")
	}
	_, err := cmd.Output()
	if err == nil {
		t.Fatal("the command was meant to fail")
	}
	return err
}

func TestWithStderrKeepsTheCommandsOwnReason(t *testing.T) {
	err := fmt.Errorf("gh pr update-branch: %w", WithStderr(failing(t, "GraphQL: rate limited")))
	if !strings.Contains(err.Error(), "GraphQL: rate limited") {
		t.Errorf("error lost the command's stderr: %q", err)
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 3 {
		t.Errorf("error no longer unwraps to the *exec.ExitError with code 3: %v", err)
	}
}

func TestWithStderrLeavesOtherErrorsAlone(t *testing.T) {
	plain := errors.New("not a process")
	if got := WithStderr(plain); got != plain {
		t.Errorf("WithStderr changed a non-exit error: %v", got)
	}
	if WithStderr(nil) != nil {
		t.Error("WithStderr(nil) is not nil")
	}
}

func TestWithStderrLeavesASilentExitAlone(t *testing.T) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "exit 3")
	} else {
		cmd = exec.Command("sh", "-c", "exit 3")
	}
	_, err := cmd.Output()
	if got := WithStderr(err); got != err {
		t.Errorf("WithStderr wrapped an exit with nothing on stderr: %q", got)
	}
}
