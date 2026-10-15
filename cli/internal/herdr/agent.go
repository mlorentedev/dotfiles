package herdr

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// Kinds are the agent kinds `herdr agent start --kind` takes, as herdr 0.9.3
// lists them. A dotfiles harness name is its kind as is; the integration
// target map above (agy -> antigravity-cli) does not apply here.
var Kinds = []string{
	"pi", "claude", "codex", "gemini", "cursor", "devin", "agy", "cline",
	"omp", "mastracode", "opencode", "copilot", "kimi", "kiro", "droid", "amp",
	"grok", "hermes", "kilo", "qodercli", "qwen", "letta", "maki", "muse",
}

// agentNameShape is herdr's rule for a live agent's name.
var agentNameShape = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// AgentName derives the herdr agent name for a worktree from its directory
// name: anything outside [a-z0-9_-] becomes "-", a name that does not start
// with a letter gets a "wt-" prefix, and it is cut to herdr's 32 characters.
func AgentName(dir string) string {
	name := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '-'
	}, strings.ToLower(dir))
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		name = "wt-" + name
	}
	if len(name) > 32 {
		name = name[:32]
	}
	return strings.TrimRight(name, "-_")
}

// herdrError is the JSON herdr prints for a server error, with exit status 1.
type herdrError struct {
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// errorCode reads herdr's error JSON from out. err is the runner's: when
// stdout carries no message (herdr crashed, was killed, or wrote only to
// stderr), it is the message, so a refusal never ends in an empty reason.
func errorCode(out []byte, err error) (code, message string) {
	var e herdrError
	if json.Unmarshal(out, &e) == nil && e.Error != nil {
		return e.Error.Code, e.Error.Message
	}
	if message = strings.TrimSpace(string(out)); message == "" && err != nil {
		message = err.Error()
	}
	return "", message
}

// Preflight refuses, naming the state, before add creates anything: a harness
// herdr cannot start, herdr absent, its server down, or an agent of the same
// name already live.
func Preflight(run tools.Runner, harness, name string) error {
	if !slices.Contains(Kinds, harness) {
		return fmt.Errorf("--agent %s is not an agent kind herdr can start (%s)", harness, strings.Join(Kinds, ", "))
	}
	out, err := run("herdr", "status", "server", "--json")
	if errors.Is(err, exec.ErrNotFound) {
		return errors.New("herdr is not installed: --agent needs it (`dotf tools sync`)")
	}
	if err != nil {
		return fmt.Errorf("herdr status server: %w: %s", err, strings.TrimSpace(string(out)))
	}
	var st struct {
		Running bool `json:"running"`
	}
	if err := json.Unmarshal(out, &st); err != nil {
		return fmt.Errorf("herdr status server: unreadable answer %q", strings.TrimSpace(string(out)))
	}
	if !st.Running {
		return errors.New("no herdr server is running: start it with `herdr`, then retry")
	}
	out, err = run("herdr", "agent", "get", name)
	if err == nil {
		// herdr's agent names are per server, not per repository: the live one
		// may be another repository's worktree on the same branch slug, so the
		// remedy is a different branch here, never renaming that agent.
		return fmt.Errorf("an agent named %s is already live in herdr, perhaps another repository's worktree on the same branch: add this worktree under another branch, or finish that agent first", name)
	}
	if code, msg := errorCode(out, err); code != "agent_not_found" {
		return fmt.Errorf("herdr agent get %s: %s", name, msg)
	}
	return nil
}

// LaunchOptions names the worktree to open and the agent to start in it.
type LaunchOptions struct {
	Repo, Worktree, Harness, Name, Prompt string
}

// Launch opens the worktree as a herdr workspace, starts the agent in its root
// pane and, with a prompt, submits it. It never passes --wait: add returns
// once the prompt is sent, and the agent works on its own. An agent that is
// not ready, or waits at a dialog, fails the launch with the prompt unsent.
func Launch(run tools.Runner, o LaunchOptions) error {
	out, err := run("herdr", "worktree", "open", "--cwd", o.Repo, "--path", o.Worktree, "--no-focus")
	if err != nil {
		_, msg := errorCode(out, err)
		return fmt.Errorf("herdr worktree open %s: %s", o.Worktree, msg)
	}
	var opened struct {
		Result struct {
			RootPane struct {
				PaneID string `json:"pane_id"`
			} `json:"root_pane"`
		} `json:"result"`
	}
	if json.Unmarshal(out, &opened) != nil || opened.Result.RootPane.PaneID == "" {
		return fmt.Errorf("herdr worktree open: no root pane in %q", strings.TrimSpace(string(out)))
	}
	pane := opened.Result.RootPane.PaneID
	if out, err := run("herdr", "agent", "start", o.Name, "--kind", o.Harness, "--pane", pane); err != nil {
		code, msg := errorCode(out, err)
		return fmt.Errorf("herdr agent start %s in %s: %s (%s); the prompt was not sent: inspect it with `herdr agent read %s`", o.Name, pane, msg, code, o.Name)
	}
	if o.Prompt == "" {
		return nil
	}
	if out, err := run("herdr", "agent", "prompt", o.Name, o.Prompt); err != nil {
		code, msg := errorCode(out, err)
		return fmt.Errorf("herdr agent prompt %s: %s (%s); the prompt was not sent: inspect it with `herdr agent read %s`", o.Name, msg, code, o.Name)
	}
	return nil
}
