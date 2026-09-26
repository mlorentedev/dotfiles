package pi

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// SlowAfter is the default verbosity threshold: a call that takes longer has
// its output printed even on success. It gates what is printed, never the
// call, so being wrong costs log lines and not an install. Sized from the
// Windows runner's distribution (#1486): normal installs at 35-345 s, the
// anomaly at 421 s, with no clean gap, so it errs toward printing too much.
const SlowAfter = 120 * time.Second

// Runner runs the pi binary with args and returns its combined output.
type Runner func(bin string, args ...string) (string, error)

// ExecRunner runs pi for real.
func ExecRunner(bin string, args ...string) (string, error) {
	out, err := exec.Command(bin, args...).CombinedOutput() // #nosec G204 -- the resolved pi binary and manifest sources
	return string(out), err
}

// Options configure an apply.
type Options struct {
	PiBin string
	Log   io.Writer
	// SlowAfter overrides the verbosity threshold; zero means the default, and
	// a negative value prints every call's output.
	SlowAfter time.Duration
}

// Result counts what an apply did.
type Result struct {
	Removed, Installed, Failed int
}

// Changed is the number of changes that took effect.
func (r Result) Changed() int { return r.Removed + r.Installed }

// Apply carries out a plan in its order: removals, then installs. A failed
// call is counted and logged, and the rest of the plan still runs.
func Apply(p Plan, opt Options, run Runner) Result {
	var res Result
	for _, src := range p.Remove {
		if call(opt, run, "remove", src) {
			res.Removed++
		} else {
			res.Failed++
		}
	}
	for _, src := range p.Install {
		if call(opt, run, "install", src) {
			res.Installed++
		} else {
			res.Failed++
		}
	}
	return res
}

// call runs one pi verb, logs its elapsed time on every outcome, and prints its
// captured output inside a fence when it failed or was slow. The fence makes
// empty output legible: "the install printed nothing" is a finding.
func call(opt Options, run Runner, verb, src string) bool {
	_, _ = fmt.Fprintf(opt.Log, "[INFO] pi %s %s ...\n", verb, src)
	started := time.Now()
	out, err := run(opt.PiBin, verb, src)
	elapsed := time.Since(started).Round(time.Second)
	threshold := opt.SlowAfter
	if threshold == 0 {
		threshold = SlowAfter
	}
	slow := elapsed >= threshold // a negative threshold makes every call verbose
	switch {
	case err != nil:
		_, _ = fmt.Fprintf(opt.Log, "[WARN] pi %s %s failed after %s (exit %d) — output follows\n", verb, src, elapsed, exitCode(err))
	case slow:
		_, _ = fmt.Fprintf(opt.Log, "[WARN] pi %s %s took %s — over the %s diagnostic threshold, output follows\n", verb, src, elapsed, threshold)
	default:
		_, _ = fmt.Fprintf(opt.Log, "[OK] %s %s in %s\n", src, done[verb], elapsed)
		return true
	}
	_, _ = fmt.Fprintf(opt.Log, "--- pi %s %s (exit %d, %s) ---\n%s\n--- end pi %s %s ---\n", verb, src, exitCode(err), elapsed, out, verb, src)
	return err == nil
}

var done = map[string]string{"install": "installed", "remove": "removed"}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 1
}
