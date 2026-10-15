package lane

import (
	"encoding/json"
	"fmt"
)

// Worktree is a worktree the backend knows.
type Worktree struct {
	Path   string
	Branch string
	Open   bool // a workspace already shows it, so an agent may be working there
}

// Started is a worktree the backend just made available, with the pane to run
// the successor in.
type Started struct {
	Worktree Worktree
	Pane     string
}

// Backend is where a successor session lives. herdr is the first; the
// interface is the seam a tmux backend would fill. It has no way to remove a
// worktree: the caller's own is never ours to close (a removed launch worktree
// strands its session).
type Backend interface {
	// CallingAgent is the agent of the pane the command is run from.
	CallingAgent() (string, error)
	// Find is the worktree of repo's branch, if one exists.
	Find(repo, branch string) (Worktree, bool, error)
	// Create makes a worktree on a new branch off base and opens it.
	Create(repo, branch, base, label string) (Started, error)
	// Open shows an existing worktree that nothing displays yet.
	Open(repo string, wt Worktree) (Started, error)
	// Type runs a command line in a pane's shell.
	Type(pane, command string) error
}

// Herdr is the herdr backend. Its worktree commands refuse a linked worktree
// as the source, so every one of them is given the repository's parent
// checkout, whichever checkout the caller stands in.
type Herdr struct{ Run Runner }

func (h Herdr) herdr(args ...string) (string, error) { return h.Run("herdr", args...) }

// Available reports whether this process runs inside a herdr pane: the
// calling pane is the one thing no other terminal can answer.
func (h Herdr) Available() bool {
	_, err := h.herdr("pane", "current")
	return err == nil
}

func (h Herdr) CallingAgent() (string, error) {
	out, err := h.herdr("pane", "current")
	if err != nil {
		return "", err
	}
	var r struct {
		Result struct {
			Pane struct {
				Agent string `json:"agent"`
			} `json:"pane"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return "", fmt.Errorf("herdr pane current: %w", err)
	}
	return r.Result.Pane.Agent, nil
}

type herdrWorktree struct {
	Branch          string `json:"branch"`
	Path            string `json:"path"`
	OpenWorkspaceID string `json:"open_workspace_id"`
}

func (w herdrWorktree) worktree() Worktree {
	return Worktree{Path: w.Path, Branch: w.Branch, Open: w.OpenWorkspaceID != ""}
}

func (h Herdr) Find(repo, branch string) (Worktree, bool, error) {
	out, err := h.herdr("worktree", "list", "--cwd", repo)
	if err != nil {
		return Worktree{}, false, err
	}
	var r struct {
		Result struct {
			Worktrees []herdrWorktree `json:"worktrees"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return Worktree{}, false, fmt.Errorf("herdr worktree list: %w", err)
	}
	for _, w := range r.Result.Worktrees {
		if w.Branch == branch {
			return w.worktree(), true, nil
		}
	}
	return Worktree{}, false, nil
}

func (h Herdr) Create(repo, branch, base, label string) (Started, error) {
	out, err := h.herdr("worktree", "create", "--cwd", repo, "--branch", branch, "--base", base, "--label", label, "--no-focus")
	if err != nil {
		return Started{}, err
	}
	return parseStarted(out, "worktree create")
}

func (h Herdr) Open(repo string, wt Worktree) (Started, error) {
	out, err := h.herdr("worktree", "open", "--cwd", repo, "--path", wt.Path, "--no-focus")
	if err != nil {
		return Started{}, err
	}
	return parseStarted(out, "worktree open")
}

func parseStarted(out, what string) (Started, error) {
	var r struct {
		Result struct {
			RootPane struct {
				PaneID string `json:"pane_id"`
			} `json:"root_pane"`
			Worktree herdrWorktree `json:"worktree"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return Started{}, fmt.Errorf("herdr %s: %w", what, err)
	}
	if r.Result.RootPane.PaneID == "" {
		return Started{}, fmt.Errorf("herdr %s: no root pane in the answer", what)
	}
	return Started{Worktree: r.Result.Worktree.worktree(), Pane: r.Result.RootPane.PaneID}, nil
}

func (h Herdr) Type(pane, command string) error {
	_, err := h.herdr("pane", "run", pane, command)
	return err
}
