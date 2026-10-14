// Package herdr drives herdr's built-in agent integrations: the hook or
// plugin each agent runs so herdr can show which pane holds which session.
// herdr owns their content and their version (`herdr integration install`
// overwrites its own files and merges its own hook entries); dotfiles only
// decides where they go, which is every installed harness (#2013 H6).
package herdr

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// targets maps a dotfiles harness name to herdr's integration name where the
// two differ.
var targets = map[string]string{"agy": "antigravity-cli"}

// Target is herdr's integration name for a dotfiles harness.
func Target(harness string) string {
	if t, ok := targets[harness]; ok {
		return t
	}
	return harness
}

// ParseStatus reads `herdr integration status`, one `<target>: <state>
// (<path>)` line per integration, into target -> state. A target marked
// `(experimental)` is keyed by its bare name.
func ParseStatus(out []byte) map[string]string {
	states := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		name, rest, ok := strings.Cut(strings.TrimSpace(line), ": ")
		if !ok {
			continue
		}
		name = strings.TrimSuffix(name, " (experimental)")
		// The path is the trailing parenthesised group. An absolute POSIX path
		// starts with "/", so the first " (/" opens it even when the path
		// holds " (" itself; any other path is cut at the last " (".
		if i := strings.Index(rest, " (/"); i >= 0 {
			rest = rest[:i]
		} else if i := strings.LastIndex(rest, " ("); i >= 0 && strings.HasSuffix(rest, ")") {
			rest = rest[:i]
		}
		states[name] = rest
	}
	return states
}

// Current reports whether a state says the integration is up to date. It is
// read on the positive: an outdated integration, or a state a later herdr
// words differently, is reinstalled, never trusted.
func Current(state string) bool { return strings.HasPrefix(state, "current") }

// Report is one pass over the harnesses, each named by its dotfiles name.
type Report struct {
	Installed []string // installed by this pass, or to install under a plan
	Current   []string // already current
	Absent    []string // harness not installed here: never touched
	Unknown   []string // installed, but herdr has no integration for it
}

// Apply installs herdr's integration for every harness in harnesses that has
// reports installed and whose integration is not current. A harness that is
// not installed is left alone, because herdr would create its config
// directory. Under dryRun nothing is installed. One failed install does not
// stop the rest; every failure is returned, joined.
func Apply(run tools.Runner, harnesses []string, has func(string) bool, dryRun bool) (Report, error) {
	var rep Report
	out, err := run("herdr", "integration", "status")
	if err != nil {
		return rep, fmt.Errorf("herdr integration status: %w: %s", err, strings.TrimSpace(string(out)))
	}
	states := ParseStatus(out)
	var errs []error
	for _, h := range slices.Sorted(slices.Values(harnesses)) {
		state, known := states[Target(h)]
		switch {
		case !has(h):
			rep.Absent = append(rep.Absent, h)
		case !known:
			rep.Unknown = append(rep.Unknown, h)
		case Current(state):
			rep.Current = append(rep.Current, h)
		case dryRun:
			rep.Installed = append(rep.Installed, h)
		default:
			if out, err := run("herdr", "integration", "install", Target(h)); err != nil {
				errs = append(errs, fmt.Errorf("herdr integration install %s: %w: %s", Target(h), err, strings.TrimSpace(string(out))))
				continue
			}
			rep.Installed = append(rep.Installed, h)
		}
	}
	return rep, errors.Join(errs...)
}
