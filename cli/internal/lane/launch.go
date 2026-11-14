// Package lane hands a work lane from the session that finished its item to a
// successor: it picks the next queued issue, and starts the same kind of agent
// on it in a fresh worktree. The launch argv is built here, in Go, and the
// prompt reaches the agent as one argument, never through a shell, so the same
// code serves zsh, bash and PowerShell panes (CLI-119).
package lane

import (
	"fmt"
	"slices"
	"strings"
)

// launchers is the interactive launch argv per agent. An agent is added only
// after a real launch of it was observed (specs/CLI-119-lane-next/verification.md):
// a guessed flag would start the successor in the wrong mode, silently.
var launchers = map[string]func(prompt string) []string{
	"claude": func(p string) []string { return []string{"claude", p} },
	// pi's keys are injected by a function the interactive shell defines
	// (bash, zsh and PowerShell each have one); exec skips functions, so the
	// bare binary would start the successor without them.
	"pi": func(p string) []string {
		return []string{"dotf", "secrets", "run", "--only", "NAN_API_KEY,OPENROUTER_API_KEY", "--", "pi", p}
	},
	"agy": func(p string) []string { return []string{"agy", "-i", p} },
	// copilot's -i starts interactive mode on the prompt; -p would run it once and exit.
	"copilot": func(p string) []string { return []string{"copilot", "-i", p} },
	// opencode's key comes from a shell function too, so it goes through secrets run
	// as pi does. --prompt pre-fills the input; the observed launch decides whether
	// it also submits (verification.md).
	"opencode": func(p string) []string {
		return []string{"dotf", "secrets", "run", "--only", "NAN_API_KEY", "--", "opencode", "--prompt", p}
	},
}

// LaunchEnv is the environment a launch must start with, beyond the caller's own:
// the agent's name for herdr, when the launch goes through `dotf secrets run`.
// herdr names a pane's agent from the foreground process, and the wrapper makes
// that process dotf, so the pane reads agent: null (#2813; observed for
// opencode, #3430). The value is read from the environment the process was
// started with, so it has to be there when the wrapper starts: set later it is
// invisible to herdr (lesson 493). A launch that starts the agent itself needs
// nothing, and gets nothing.
func LaunchEnv(argv []string) []string {
	if len(argv) < 4 || argv[0] != "dotf" || argv[1] != "secrets" || argv[2] != "run" {
		return nil
	}
	dash := slices.Index(argv, "--")
	if dash < 0 || dash+1 >= len(argv) {
		return nil
	}
	name := argv[dash+1]
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".exe"), ".EXE")
	if name == "" {
		return nil
	}
	return []string{"HERDR_AGENT=" + name}
}

// Agents lists the agents LaunchArgv knows, sorted.
func Agents() []string {
	names := make([]string, 0, len(launchers))
	for name := range launchers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// LaunchArgv is the command that starts agent on prompt, interactively. The
// prompt is refused when it would not reach the agent as one: empty, a leading
// dash (read as a flag) or a leading @ (pi attaches the file it names).
func LaunchArgv(agent, prompt string) ([]string, error) {
	build, ok := launchers[agent]
	if !ok {
		return nil, fmt.Errorf("lane: no launch command for agent %q (known: %s)", agent, strings.Join(Agents(), ", "))
	}
	if p := strings.TrimSpace(prompt); p == "" || p[0] == '-' || p[0] == '@' {
		return nil, fmt.Errorf("lane: the prompt must start with text, not %q", firstWord(prompt))
	}
	return build(prompt), nil
}

func firstWord(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return ""
}
