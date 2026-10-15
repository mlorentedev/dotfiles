package lane

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Options is one hand-off: which repo, which queue, which agent.
type Options struct {
	Agent     string // empty: the agent of the calling pane
	DryRun    bool   // decide and print, change nothing
	Config    Config
	Template  string // the prompt template's text
	Parent    string // the repository's parent checkout
	PromptDir string // where the rendered prompt is left for the successor
	Fetch     func() error
	// Dotf is the command word the successor's pane runs `lane exec` with. The
	// caller passes the path of the binary that decided, so the successor is the
	// same dotf and not whichever release `dotf` resolves to on the pane's PATH.
	Dotf string
}

// Outcome is what a hand-off decided and did.
type Outcome struct {
	Issue    Issue
	Branch   string
	Agent    string
	Started  bool     // a successor agent was started
	Note     string   // why nothing was started, or what was reused
	Commands []string // what a person runs when nothing could be run for them
}

// plainWord is the one shape a path may take in the command typed into the
// successor's pane: that line is read by zsh, bash, PowerShell or cmd, and a
// word of these characters means the same in all four. Anything else would need
// quoting rules per shell, which is what lane exec exists to avoid. A tilde is
// allowed after the first character: Windows short names (RUNNER~1) carry one,
// and no shell expands it mid-word.
var plainWord = regexp.MustCompile(`^[A-Za-z0-9_./:\\-][A-Za-z0-9_./:\\~-]*$`)

// Handoff starts the successor of a finished lane item. With a nil backend it
// starts nothing and returns the commands that would. It never removes a
// worktree, and running it twice for the same item starts one agent, because
// an item whose branch is already shown by a workspace is left to that one.
func Handoff(o Options, b Backend, src IssueSource) (Outcome, error) {
	issue, ok, err := Next(o.Config, src)
	if err != nil {
		return Outcome{}, err
	}
	if !ok {
		return Outcome{Note: "every queued issue is closed or held; nothing to hand off"}, nil
	}
	out := Outcome{Issue: issue, Branch: o.Config.BranchPrefix + "/" + Slug(issue.Title)}

	prompt, err := RenderPrompt(o.Template, PromptData{
		Number: issue.Number, Title: issue.Title, Branch: out.Branch, Remaining: remaining(o.Config.Queue, issue.Number),
	})
	if err != nil {
		return out, err
	}
	if out.Agent, err = resolveAgent(o.Agent, b); err != nil {
		return out, err
	}
	if _, err := LaunchArgv(out.Agent, prompt); err != nil {
		return out, err
	}
	promptFile := filepath.Join(o.PromptDir, strconv.Itoa(issue.Number)+".md")
	if !plainWord.MatchString(promptFile) {
		return out, fmt.Errorf("lane: the prompt path %q has characters a pane's shell would read differently; use a PromptDir of letters, digits and _ . / : \\ ~ -", promptFile)
	}
	dotf := o.Dotf
	if dotf == "" || !plainWord.MatchString(dotf) {
		dotf = "dotf"
	}
	exec := fmt.Sprintf("%s lane exec --agent %s --prompt-file %s", dotf, out.Agent, promptFile)

	if b == nil {
		out.Commands = manualCommands(o, out, exec)
		out.Note = "no session backend found; run these commands"
		return out, writePrompt(o, promptFile, prompt)
	}
	wt, found, err := b.Find(o.Parent, out.Branch)
	if err != nil {
		return out, err
	}
	if found && wt.Open {
		out.Note = "a workspace already shows " + out.Branch + "; no second agent started"
		return out, nil
	}
	if o.DryRun {
		out.Commands = []string{"(write the prompt to " + promptFile + ")", "(" + describe(found) + " " + out.Branch + ")", exec}
		return out, nil
	}
	if err := writePrompt(o, promptFile, prompt); err != nil {
		return out, err
	}
	var st Started
	if found {
		st, err = b.Open(o.Parent, wt)
	} else {
		if o.Fetch != nil {
			if err := o.Fetch(); err != nil {
				return out, err
			}
		}
		st, err = b.Create(o.Parent, out.Branch, o.Config.Base, fmt.Sprintf("#%d %s", issue.Number, Slug(issue.Title)))
	}
	if err != nil {
		return out, err
	}
	if err := b.Type(st.Pane, exec); err != nil {
		return out, err
	}
	out.Started = true
	return out, nil
}

func describe(found bool) string {
	if found {
		return "open the existing worktree of"
	}
	return "create a worktree for"
}

// resolveAgent is the explicit agent, else the calling pane's. LaunchArgv then
// refuses one the table lacks: a pane running something else is a reason to
// say so, not to start a different kind of successor.
func resolveAgent(want string, b Backend) (string, error) {
	agent := want
	if agent == "" {
		if b == nil {
			return "", fmt.Errorf("lane: no --agent given and no session backend to ask (known: %s)", strings.Join(Agents(), ", "))
		}
		got, err := b.CallingAgent()
		if err != nil {
			return "", err
		}
		agent = got
	}
	return agent, nil
}

func remaining(queue []int, current int) []int {
	i := slices.Index(queue, current)
	if i < 0 {
		return nil
	}
	return queue[i+1:]
}

func writePrompt(o Options, path, prompt string) error {
	if o.DryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("lane: %w", err)
	}
	return os.WriteFile(path, []byte(prompt), 0o600)
}

func manualCommands(o Options, out Outcome, exec string) []string {
	dir := filepath.Join(filepath.Dir(o.Parent), "worktrees", filepath.Base(o.Parent), Slug(out.Issue.Title))
	base := o.Config.Base
	return []string{
		"git -C " + o.Parent + " fetch " + strings.SplitN(base, "/", 2)[0],
		"git -C " + o.Parent + " worktree add -b " + out.Branch + " " + dir + " " + base,
		"cd " + dir,
		exec,
	}
}
