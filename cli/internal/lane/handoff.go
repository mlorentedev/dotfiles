package lane

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// Options is one hand-off: which repo, which queue, which agent.
type Options struct {
	Agent    string // empty: the agent of the calling pane
	DryRun   bool   // decide and print, change nothing
	Config   Config
	Template string // the prompt template's text
	Parent   string // the repository's parent checkout
	// PromptDir is where the prompt is left when there is no worktree to put it
	// in yet (no backend): a person reads the path from the printed commands.
	PromptDir string
	// WorktreeRoot is where `dotf worktree add` and herdr put a branch's
	// worktree. Only the commands printed for a person need it: a backend
	// decides the path itself.
	WorktreeRoot string
	Fetch        func() error
	// Dotf is the command word the successor's pane runs `lane exec` with. The
	// caller passes the path of the binary that decided, so the successor is the
	// same dotf and not whichever release `dotf` resolves to on the pane's PATH.
	Dotf string
	// GitDir is the repository's git common directory: the lease file is
	// excluded in its info/exclude, which every worktree of the repo reads.
	GitDir string
	Now    func() time.Time // the lease's clock; nil is time.Now
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

// plainWord is the one shape the binary's path may take in the command typed
// into the successor's pane: that line is read by zsh, bash, PowerShell or cmd,
// and a word of these characters means the same in all four. Anything else
// would need quoting rules per shell, which is what lane exec exists to avoid.
// A tilde is allowed after the first character: Windows short names (RUNNER~1)
// carry one, and no shell expands it mid-word. A backslash is a path separator
// only on Windows; zsh and bash read it as an escape, so it is a plain
// character there only on Windows.
var plainWord = plainWordFor(runtime.GOOS)

func plainWordFor(goos string) *regexp.Regexp {
	if goos == "windows" {
		return regexp.MustCompile(`^[A-Za-z0-9_./:\\-][A-Za-z0-9_./:\\~-]*$`)
	}
	return regexp.MustCompile(`^[A-Za-z0-9_./:-][A-Za-z0-9_./:~-]*$`)
}

// Handoff starts the successor of a finished lane item. With a nil backend it
// starts nothing and returns the commands that would. It never removes a
// worktree, and running it twice for the same item starts one agent, because
// an item whose branch is already shown by a workspace is left to that one.
// Every worktree it starts an agent in, or finds already shown, is leased
// first, so sweep can reap it once its PR lands.
func Handoff(o Options, b Backend, src IssueSource) (Outcome, error) {
	issue, ok, err := Next(o.Config, src)
	if err != nil {
		return Outcome{}, err
	}
	if !ok {
		return Outcome{Note: "every queued issue is closed or held; nothing to hand off"}, nil
	}
	out := Outcome{Issue: issue, Branch: o.Config.BranchPrefix + "/" + Slug(issue.Title)}

	p, err := prepare(o, b, &out)
	if err != nil {
		return out, err
	}
	if b == nil {
		return noBackend(o, out, p)
	}
	wt, found, err := b.Find(o.Parent, out.Branch)
	if err != nil {
		return out, err
	}
	if found && wt.Open {
		out.Note = "a workspace already shows " + out.Branch + "; no second agent started"
		if o.DryRun {
			return out, nil
		}
		return out, ensureLease(o, wt.Path, issue.Number, out.Agent)
	}
	if o.DryRun {
		out.Commands = []string{"(write the prompt to " + PromptFile + " in the new worktree)", "(" + describe(found) + " " + out.Branch + ")", p.exec(PromptFile)}
		out.Note = p.note
		return out, nil
	}
	if err := launch(o, b, wt, found, out, p); err != nil {
		return out, err
	}
	out.Started, out.Note = true, p.note
	return out, nil
}

// PromptFile is the name the prompt is left under, inside the successor's
// worktree. The command typed into the pane names it relative to that pane's
// directory, so the line carries no path anyone chose: a git directory, a temp
// directory or a home with a space in it cannot reach the pane's shell.
const PromptFile = ".dotf-lane-prompt.md"

// launch starts the branch's worktree, leases it, leaves the prompt in it, and
// only then types the command: the lease must exist before anything runs there.
func launch(o Options, b Backend, wt Worktree, found bool, out Outcome, p plan) error {
	st, err := start(o, b, wt, found, out.Branch, out.Issue)
	if err != nil {
		return err
	}
	if err := ensureLease(o, st.Worktree.Path, out.Issue.Number, out.Agent); err != nil {
		return err
	}
	if err := excludeFile(o.GitDir, PromptFile); err != nil {
		return err
	}
	if err := writePrompt(o, filepath.Join(st.Worktree.Path, PromptFile), p.prompt); err != nil {
		return err
	}
	return b.Type(st.Pane, p.exec(PromptFile))
}

// plan is what a hand-off hands the successor: its prompt, the dotf its pane
// runs, and a note when that is not quite what the caller asked for.
type plan struct{ prompt, dotf, agent, note string }

// exec is the command line typed into the successor's pane, reading its prompt
// from file.
func (p plan) exec(file string) string {
	return fmt.Sprintf("%s lane exec --agent %s --prompt-file %s", p.dotf, p.agent, file)
}

// prepare renders the prompt, settles the agent and builds the command line.
// out.Agent is set as a side effect, so a refusal still names it.
func prepare(o Options, b Backend, out *Outcome) (plan, error) {
	prompt, err := RenderPrompt(o.Template, PromptData{
		Number: out.Issue.Number, Title: out.Issue.Title, Branch: out.Branch, Remaining: remaining(o.Config.Queue, out.Issue.Number),
	})
	if err != nil {
		return plan{}, err
	}
	if out.Agent, err = resolveAgent(o.Agent, b); err != nil {
		return plan{}, err
	}
	if out.Agent != agentPlaceholder() {
		if _, err := LaunchArgv(out.Agent, prompt); err != nil {
			return plan{}, err
		}
	}
	dotf, note := o.Dotf, ""
	if dotf == "" || !plainWord.MatchString(dotf) {
		if dotf != "" {
			note = fmt.Sprintf("the successor runs `dotf` from its pane's PATH: %q is not a path every shell reads alike", dotf)
		}
		dotf = "dotf"
	}
	return plan{prompt, dotf, out.Agent, note}, nil
}

// humanPath is a path as a person pastes it into their own shell: bare when it
// is a plain word, else in double quotes, which bash, zsh, PowerShell and cmd
// all read as one word. A character that is live inside double quotes in one
// of them ($ and backtick in bash and zsh, ! for their history expansion, % for
// cmd's variables) is refused, since no quoting of it means the same in all four.
func humanPath(p string) (string, error) {
	if plainWord.MatchString(p) {
		return p, nil
	}
	if strings.ContainsAny(p, "\"$`!%") {
		return "", fmt.Errorf("lane: %q has a character no shell quotes alike; move the repository to a path without a quote, $, backtick, ! or %%", p)
	}
	return `"` + p + `"`, nil
}

// noBackend answers a hand-off with nothing to start the successor in: the
// commands a person runs, and the prompt they read. When no agent was named it
// cannot be guessed (no pane to ask, and no harness is privileged), so the
// printed line carries a placeholder for the person to fill.
func noBackend(o Options, out Outcome, p plan) (Outcome, error) {
	file := filepath.Join(o.PromptDir, strconv.Itoa(out.Issue.Number)+".md")
	quoted, err := humanPath(file)
	if err != nil {
		return out, err
	}
	cmds, err := manualCommands(o, out, p.exec(quoted))
	if err != nil {
		return out, err
	}
	out.Commands = cmds
	notes := []string{"no session backend found; run these commands"}
	if out.Agent == agentPlaceholder() {
		notes = append(notes, "replace "+out.Agent+" with the agent to start")
	}
	if p.note != "" {
		notes = append(notes, p.note)
	}
	out.Note = strings.Join(notes, "; ")
	return out, writePrompt(o, file, p.prompt)
}

// start opens the worktree that exists, or creates the branch's off the base,
// and answers with the pane to run the successor in. An opened worktree's path
// is the one Find reported when the open answer leaves it out.
func start(o Options, b Backend, wt Worktree, found bool, branch string, issue Issue) (Started, error) {
	if found {
		st, err := b.Open(o.Parent, wt)
		if err == nil && st.Worktree.Path == "" {
			st.Worktree.Path = wt.Path
		}
		return st, err
	}
	if o.Fetch != nil {
		if err := o.Fetch(); err != nil {
			return Started{}, err
		}
	}
	return b.Create(o.Parent, branch, o.Config.Base, fmt.Sprintf("#%d %s", issue.Number, Slug(issue.Title)))
}

func describe(found bool) string {
	if found {
		return "open the existing worktree of"
	}
	return "create a worktree for"
}

// agentPlaceholder stands in for the agent in the commands printed when there
// is no backend to ask and no --agent: a default would privilege one harness.
func agentPlaceholder() string { return "<" + strings.Join(Agents(), "|") + ">" }

// resolveAgent is the explicit agent, else the calling pane's, else (with no
// backend to ask) the placeholder. LaunchArgv then refuses one the table lacks:
// a pane running something else is a reason to say so, not to start a
// different kind of successor.
func resolveAgent(want string, b Backend) (string, error) {
	agent := want
	if agent == "" {
		if b == nil {
			return agentPlaceholder(), nil
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

// manualCommands is what a person runs to do what a backend would. The worktree
// goes where `dotf worktree add` and herdr put it, so `dotf worktree done` and
// the sweep find it by its branch.
func manualCommands(o Options, out Outcome, exec string) ([]string, error) {
	if o.WorktreeRoot == "" {
		return nil, errors.New("lane: WORKTREE_ROOT does not resolve, so the worktree's path cannot be printed; run `dotf doctor`")
	}
	dir, err := humanPath(worktree.ResolveWorktreePath(o.WorktreeRoot, filepath.Base(o.Parent), out.Branch))
	if err != nil {
		return nil, err
	}
	parent, err := humanPath(o.Parent)
	if err != nil {
		return nil, err
	}
	remote, _ := o.Config.Remote()
	return []string{
		"git -C " + parent + " fetch " + remote,
		"git -C " + parent + " worktree add -b " + out.Branch + " " + dir + " " + o.Config.Base,
		"cd " + dir,
		exec,
	}, nil
}
