package sshtarget

import (
	"os"
	"os/exec"
)

// Sudo is Root through sudo. Ready reuses a cached credential when sudo has
// one, else asks for the password once on the terminal; each command then
// runs with `sudo -n`, which fails rather than prompt again. With no terminal
// and no cached credential, Ready returns ErrNoTerminal and nothing runs.
type Sudo struct {
	// Terminal is the terminal to ask on, ok false when there is none.
	Terminal func() (tty *os.File, ok bool)
}

func (s Sudo) Ready() error {
	if exec.Command("sudo", "-n", "true").Run() == nil {
		return nil
	}
	tty, ok := s.Terminal()
	if !ok {
		return ErrNoTerminal
	}
	c := exec.Command("sudo", "-v", "-p", "dotf converge needs root for workstation-target. Password for %u: ")
	c.Stdin, c.Stdout, c.Stderr = tty, tty, tty
	return c.Run()
}

func (Sudo) Run(argv ...string) ([]byte, error) {
	return exec.Command("sudo", append([]string{"-n"}, argv...)...).CombinedOutput() //nolint:gosec // the target's fixed commands
}

// Exec runs a command as the current user and returns its combined output.
func Exec(argv ...string) ([]byte, error) {
	return exec.Command(argv[0], argv[1:]...).CombinedOutput() //nolint:gosec // the target's fixed commands
}
