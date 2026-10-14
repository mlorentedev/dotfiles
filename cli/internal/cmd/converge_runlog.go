package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// dotfVersion is the running binary's version, for the transcript's header.
var dotfVersion = "dev"

// consoleStdout is the process's real stdout while a converge run captures
// stdout into its transcript; nil otherwise.
var consoleStdout *os.File

// consoleOut is the stdout a person is looking at. A terminal check made
// during a captured run reads it, not os.Stdout, which is then a pipe.
func consoleOut() *os.File {
	if consoleStdout != nil {
		return consoleStdout
	}
	return os.Stdout
}

// drainTimeout bounds how long the end of a run waits for the capture to
// drain. A child that outlived the run (a daemon it started) keeps the pipe
// open; the run must still exit.
const drainTimeout = 2 * time.Second

// secretNameShape is a variable name that holds a credential by convention:
// one that ends like one, plus Bitwarden's session. Anchored on the suffix,
// because a name that merely contains SESSION or CREDENTIAL holds ordinary
// values (XDG_SESSION_TYPE=wayland, a credentials file's path) that masking
// would shred throughout the transcript.
var secretNameShape = regexp.MustCompile(`(?i)((^|_)(TOKEN|SECRET|SECRET_KEY|PASSWORD|PASSWD|PASSPHRASE|API_?KEY|PRIVATE_KEY|ACCESS_KEY|PAT)$|^BW_SESSION$)`)

// runLogSecrets is what a transcript masks by value: every variable the
// checkout's secrets registry exposes, and every variable whose name says it
// holds a credential, as set in this process right now.
func runLogSecrets(repo string) map[string]string {
	names := map[string]bool{}
	if data, err := os.ReadFile(filepath.Join(repo, "secrets", "registry.yaml")); err == nil {
		if reg, _, perr := secrets.ParseRegistryPartial(data); perr == nil && reg != nil {
			for _, s := range reg.Secrets {
				for _, v := range s.Expose.Env.Vars {
					names[v.Name] = true
				}
			}
		}
	}
	out := map[string]string{}
	for _, kv := range os.Environ() {
		name, value, ok := strings.Cut(kv, "=")
		if ok && value != "" && (names[name] || secretNameShape.MatchString(name)) {
			out[name] = value
		}
	}
	return out
}

// runCapture tees the process's stdout and stderr into a run's transcript
// while they still reach the console. It swaps the os.Stdout and os.Stderr
// files, so what Go code prints and what every child inherits is captured
// alike, on every OS, with no per-reconciler plumbing. A child then writes to
// a pipe, not a terminal; where the console is a terminal the colour hints
// below keep tools that honour them coloured.
type runCapture struct {
	out, errOut  *os.File // the originals
	wOut, wErr   *os.File
	drained      sync.WaitGroup
	restoreColor func()
}

func startRunCapture(rl *converge.RunLog) (*runCapture, error) {
	rOut, wOut, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		_ = rOut.Close()
		_ = wOut.Close()
		return nil, err
	}
	c := &runCapture{out: os.Stdout, errOut: os.Stderr, wOut: wOut, wErr: wErr}
	c.restoreColor = forceColorFor(term.IsTerminal(int(c.out.Fd())))
	c.pump(rOut, c.out, rl.Stream())
	c.pump(rErr, c.errOut, rl.Stream())
	os.Stdout, os.Stderr, consoleStdout = wOut, wErr, c.out
	return c, nil
}

// pump copies one pipe to the console and the transcript. A failing console
// write never stops the transcript, nor the other way round.
func (c *runCapture) pump(r *os.File, console *os.File, log io.Writer) {
	c.drained.Add(1)
	go func() {
		defer c.drained.Done()
		defer func() { _ = r.Close() }()
		buf := make([]byte, 32*1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				_, _ = console.Write(buf[:n])
				_, _ = log.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
}

// stop restores stdout and stderr and waits, bounded, for the pipes to drain.
func (c *runCapture) stop() {
	os.Stdout, os.Stderr, consoleStdout = c.out, c.errOut, nil
	_ = c.wOut.Close()
	_ = c.wErr.Close()
	c.restoreColor()
	done := make(chan struct{})
	go func() { c.drained.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(drainTimeout):
	}
}

// colorHints are the variables that make a child colour a pipe: FORCE_COLOR
// for Node and uv, CLICOLOR_FORCE for most BSD-style and Rust CLIs, and
// HOMEBREW_COLOR, because brew honours neither. None of them colours the
// output dotf parses: measured on darwin-arm64, 38 mise tools' version probes
// and brew's list, info and --prefix stay escape-free with all three set.
var colorHints = []string{"FORCE_COLOR", "CLICOLOR_FORCE", "HOMEBREW_COLOR"}

// forceColorFor sets colorHints when the console is a terminal, so capturing
// the children's output does not strip their colour, and returns what undoes
// that. A hint the person already set is left alone.
func forceColorFor(tty bool) func() {
	if !tty {
		return func() {}
	}
	var set []string
	for _, k := range colorHints {
		if _, ok := os.LookupEnv(k); !ok {
			_ = os.Setenv(k, "1")
			set = append(set, k)
		}
	}
	return func() {
		for _, k := range set {
			_ = os.Unsetenv(k)
		}
	}
}

// openConvergeRunLog starts an applied run's transcript, or returns nil and
// says why: a transcript that cannot be written never stops the converge.
func openConvergeRunLog(w io.Writer, repo string, args []string) *converge.RunLog {
	state, err := env.StateDir()
	if err == nil {
		header := fmt.Sprintf("dotf %s on %s/%s: %s", dotfVersion, runtime.GOOS, runtime.GOARCH, strings.Join(args, " "))
		var rl *converge.RunLog
		if rl, err = converge.OpenRunLog(state, time.Now(), runLogSecrets(repo), header); err == nil {
			return rl
		}
	}
	_, _ = fmt.Fprintf(w, "warning: this run keeps no transcript: %v\n", err)
	return nil
}

// loggedRunner records a command whose output the run keeps to itself (mise's
// installs return it instead of printing it) in the transcript as well.
func loggedRunner(run tools.Runner, log io.Writer) tools.Runner {
	if run == nil {
		return nil
	}
	return func(name string, args ...string) ([]byte, error) {
		out, err := run(name, args...)
		_, _ = fmt.Fprintf(log, "$ %s\n%s", strings.Join(append([]string{name}, args...), " "), out)
		if len(out) > 0 && out[len(out)-1] != '\n' {
			_, _ = io.WriteString(log, "\n")
		}
		if err != nil {
			_, _ = fmt.Fprintf(log, "(exit: %v)\n", err)
		}
		return out, err
	}
}

// teeWriter writes to each writer and ignores their errors: a console that
// went away must not cost the transcript the report, nor the other way round.
type teeWriter []io.Writer

func (t teeWriter) Write(p []byte) (int, error) {
	for _, w := range t {
		_, _ = w.Write(p)
	}
	return len(p), nil
}
