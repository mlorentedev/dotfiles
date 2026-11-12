package converge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/diskfree"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/filelock"
	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// cacheGC keeps the developer caches of a workstation bounded (#3367). Once a
// week it runs each present tool's own garbage collection, and only the kind
// that keeps what is in use: an age filter, or entries nothing refers to. It
// never clears a cache outright, because a cold cache costs a rebuild or a
// re-download that a disk-space saving does not repay.
//
// It rides the daily `dotf update` of the self-deploy role, so it has no
// trigger of its own; a stamp under the state dir holds the weekly cadence.
// Elsewhere it runs whenever someone runs `dotf converge`.
//
// Left out on purpose: Go's build cache trims itself (entries unused for five
// days), Homebrew is cleaned by system-upgrade, tagged Docker images are a
// re-pull away, and browser or application caches belong to their apps.
type cacheGC struct {
	run      tools.Runner                 // ExecManager in production; nil: the step is skipped
	has      func(string) bool            // is a command on PATH
	free     func(string) (uint64, error) // diskfree.Free in production; nil: no delta is reported
	now      func() time.Time             // time.Now when unset
	stateDir func(Env) (string, error)    // where the stamp lives; dotf's state dir when unset
	tryLock  func(string) (func(), error) // filelock.TryLock when unset
}

// gcPruner is one tool's safe collection. A pruner whose tool is not on PATH
// is left out; one whose ready check gives a reason (a daemon that is not
// running, a cache another process holds) is skipped and named.
type gcPruner struct {
	tool  string
	ready func(cacheGC) string // nil: always ready
	cmds  [][]string
}

// gcPruners run in this order: Docker frees blocks inside the colima VM, and
// fstrim, last, hands them back to the host's disk file.
var gcPruners = []gcPruner{
	{tool: "uv", ready: uvCacheFree, cmds: [][]string{{"uv", "cache", "prune"}}},
	{tool: "pre-commit", cmds: [][]string{{"pre-commit", "gc"}}},
	{tool: "npm", cmds: [][]string{{"npm", "cache", "verify"}}},
	{tool: "docker", ready: answers("docker", "info"), cmds: [][]string{
		{"docker", "builder", "prune", "-f", "--filter", "until=168h"},
		{"docker", "image", "prune", "-f", "--filter", "until=168h"},
	}},
	// --keep-referred keeps the images an instance or template was built
	// from; without it the next instance downloads its base image again.
	{tool: "limactl", cmds: [][]string{{"limactl", "prune", "--keep-referred"}}},
	{tool: "colima", ready: answers("colima", "status"), cmds: [][]string{{"colima", "ssh", "--", "sudo", "fstrim", "-a"}}},
}

// gcCadence is a week less an hour, so a daily trigger that fires a little
// earlier than last week still finds the step due on the seventh day.
const gcCadence = 7*24*time.Hour - time.Hour

func (cacheGC) Name() string        { return "cache-gc" }
func (cacheGC) Platforms() []string { return nil }

func (r cacheGC) Reconcile(e Env, dryRun bool) (Result, error) {
	if r.run == nil {
		return Result{Skip: "no command runner is wired into this registry"}, nil
	}
	present := r.present()
	if len(present) == 0 {
		return Result{Skip: "none of uv, pre-commit, npm, docker, limactl or colima is on PATH"}, nil
	}
	stamp, err := r.stampPath(e)
	if err != nil {
		return Result{}, err
	}
	before, freeErr := r.freeSpace(e.Home)
	why, due := r.due(stamp, before, freeErr)
	if !due {
		return Result{Detail: why}, nil
	}
	if dryRun {
		return Result{Opaque: true, Detail: "due (" + why + "): would run " + strings.Join(plannedCommands(present), ", ")}, nil
	}
	return r.prune(stamp, present, before, freeErr, e.Home)
}

// due says whether the step runs now, and why. A home volume under
// diskfree.Low makes it due whatever the stamp says: a disk filling up is the
// reason the step exists, and it must not wait out the week. That also makes
// `dotf converge --only cache-gc` the remedy doctor can name for a low disk.
func (r cacheGC) due(stamp string, free uint64, freeErr error) (string, bool) {
	if freeErr == nil && free < diskfree.Low {
		return "free space " + gib(free) + " is under " + gib(diskfree.Low), true
	}
	last, ok := r.lastRun(stamp)
	if !ok {
		return "no previous run recorded", true
	}
	if r.clock().Sub(last) >= gcCadence {
		return "last run " + last.Format(time.DateOnly), true
	}
	return fmt.Sprintf("not due: last run %s, next after %s",
		last.Format(time.DateOnly), last.Add(gcCadence).Format(time.DateOnly)), false
}

// prune runs every present pruner, and writes the stamp only when none failed,
// so a failure is retried by the next daily run instead of a week later.
func (r cacheGC) prune(stamp string, present []gcPruner, before uint64, beforeErr error, home string) (Result, error) {
	var ran, skipped []string
	var failed []error
	for _, p := range present {
		if p.ready != nil {
			if reason := p.ready(r); reason != "" {
				skipped = append(skipped, p.tool+": "+reason)
				continue
			}
		}
		for _, argv := range p.cmds {
			cmd := strings.Join(argv, " ")
			if _, err := r.run(argv[0], argv[1:]...); err != nil {
				failed = append(failed, fmt.Errorf("%s: %w", cmd, err))
				continue
			}
			ran = append(ran, cmd)
		}
	}
	detail := "ran " + orNone(ran)
	if len(skipped) > 0 {
		detail += "; skipped " + strings.Join(skipped, ", ")
	}
	if after, err := r.freeSpace(home); err == nil && beforeErr == nil {
		detail += "; free on the home volume " + freeDelta(before, after)
	}
	if len(failed) > 0 {
		return Result{Opaque: true, Detail: detail}, errors.Join(failed...)
	}
	if err := writeGCStamp(stamp, r.clock()); err != nil {
		return Result{Opaque: true, Detail: detail}, err
	}
	return Result{Opaque: true, Detail: detail}, nil
}

// Probe holds the post-condition of an apply: the stamp says the step is not
// due. A step with nothing to run, or no runner, has nothing to hold.
func (r cacheGC) Probe(e Env) error {
	if r.run == nil || len(r.present()) == 0 {
		return nil
	}
	stamp, err := r.stampPath(e)
	if err != nil {
		return err
	}
	if last, ok := r.lastRun(stamp); !ok || r.clock().Sub(last) >= gcCadence {
		return fmt.Errorf("cache-gc is still due after it ran: %s was not written", stamp)
	}
	return nil
}

// answers is ready when the command succeeds: a daemon that does not answer is
// an ordinary state on a workstation, so its pruners skip instead of failing.
func answers(name string, args ...string) func(cacheGC) string {
	return func(r cacheGC) string {
		if _, err := r.run(name, args...); err != nil {
			return "not running"
		}
		return ""
	}
}

// uvCacheFree is ready when no uv process holds the cache. `uv cache prune`
// needs the cache's lock to itself, and every running `uvx` (an MCP server an
// agent session started, say) holds it shared for as long as it runs, often
// days: without this check the prune waits five minutes and fails every day.
// A cache in use is a skip; the step tries again on its next run (#3370).
func uvCacheFree(r cacheGC) string {
	out, err := r.run("uv", "cache", "dir")
	dir := strings.TrimSpace(string(out))
	if err != nil || dir == "" {
		return "" // the prune itself reports what is wrong
	}
	lock := r.tryLock
	if lock == nil {
		lock = filelock.TryLock
	}
	release, err := lock(filepath.Join(dir, ".lock"))
	if errors.Is(err, filelock.ErrLocked) {
		return "cache in use by a running uv process"
	}
	if err == nil {
		release()
	}
	return ""
}

func (r cacheGC) present() []gcPruner {
	var out []gcPruner
	for _, p := range gcPruners {
		if r.has(p.tool) {
			out = append(out, p)
		}
	}
	return out
}

func (r cacheGC) clock() time.Time {
	if r.now == nil {
		return time.Now()
	}
	return r.now()
}

func (r cacheGC) stampPath(e Env) (string, error) {
	dir := r.stateDir
	if dir == nil {
		dir = func(e Env) (string, error) { return env.StateDirFor(os.Getenv, e.Home) }
	}
	state, err := dir(e)
	if err != nil {
		return "", err
	}
	return filepath.Join(state, "cache-gc", "last-run"), nil
}

// lastRun reads the stamp. A missing or unreadable one reports false, which
// makes the step due: a corrupt file must not switch maintenance off for good.
func (r cacheGC) lastRun(stamp string) (time.Time, bool) {
	b, err := os.ReadFile(stamp) //nolint:gosec // dotf's own state file
	if err != nil {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(b)))
	return t, err == nil
}

func (r cacheGC) freeSpace(home string) (uint64, error) {
	if r.free == nil {
		return 0, errors.New("no free-space probe")
	}
	return r.free(home)
}

func writeGCStamp(stamp string, at time.Time) error {
	if err := os.MkdirAll(filepath.Dir(stamp), 0o700); err != nil {
		return err
	}
	return os.WriteFile(stamp, []byte(at.UTC().Format(time.RFC3339)+"\n"), 0o600)
}

func plannedCommands(present []gcPruner) []string {
	var out []string
	for _, p := range present {
		for _, argv := range p.cmds {
			out = append(out, strings.Join(argv, " "))
		}
	}
	return out
}

func orNone(s []string) string {
	if len(s) == 0 {
		return "nothing"
	}
	return strings.Join(s, ", ")
}

// freeDelta is the free space before and after, and the change when it shows
// at one decimal: a prune that freed nothing reads "unchanged", not "-0.0".
func freeDelta(before, after uint64) string {
	d := (float64(after) - float64(before)) / (1 << 30)
	if d > -0.05 && d < 0.05 {
		return gib(after) + ", unchanged"
	}
	return fmt.Sprintf("%s → %s (%+.1f GiB)", gib(before), gib(after), d)
}

func gib(b uint64) string { return fmt.Sprintf("%.1f GiB", float64(b)/(1<<30)) }
