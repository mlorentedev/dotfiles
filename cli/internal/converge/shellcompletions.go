package converge

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// completionTools are the CLIs whose zsh completion the machine gets: dotf and
// herdr print theirs (`<tool> completion zsh`), and neither Homebrew nor mise
// installs it as an _file on fpath (DX-005, #2645).
var completionTools = []string{"dotf", "herdr"}

// shellCompletions writes ~/.zsh/completions/_<tool> for each completionTools
// entry on PATH. .zshrc puts that directory on fpath ahead of compinit and
// rebuilds its dump when a file there is newer, so a written completion is live
// in the next shell. Compared by content, not mtime: a tool upgrade that
// changes its completion changes the file, and a second run changes nothing.
// A tool that is not installed is skipped by name and its file left alone.
// Linux and macOS: zsh is the only shell this serves.
type shellCompletions struct {
	run tools.Runner      // ExecStdout in production
	has func(string) bool // is a command on PATH
}

func (shellCompletions) Name() string        { return "shell-completions" }
func (shellCompletions) Platforms() []string { return []string{"linux", "darwin"} }

// ExecStdout runs name and returns its stdout alone: unlike tools.ExecRunner it
// does not append stderr, which would put a tool's warning inside the file.
func ExecStdout(name string, args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	cmd := exec.Command(name, args...) //nolint:gosec // name is one of completionTools
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.Bytes(), err
}

type completionFile struct {
	tool, path string
	want       []byte
}

// plan reads each installed tool's completion and returns the files that differ
// from it, and the tools that were not installed.
func (r shellCompletions) plan(env Env) (todo []completionFile, skipped []string, err error) {
	dir := filepath.Join(env.Home, ".zsh", "completions")
	for _, tool := range completionTools {
		if !r.has(tool) {
			skipped = append(skipped, tool)
			continue
		}
		want, err := r.run(tool, "completion", "zsh")
		if err != nil {
			return nil, nil, fmt.Errorf("%s completion zsh: %w", tool, err)
		}
		if len(bytes.TrimSpace(want)) == 0 {
			return nil, nil, fmt.Errorf("%s completion zsh printed nothing", tool)
		}
		path := filepath.Join(dir, "_"+tool)
		have, err := os.ReadFile(path) //nolint:gosec // fixed name under the user's home
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, nil, err
		}
		if !bytes.Equal(have, want) {
			todo = append(todo, completionFile{tool: tool, path: path, want: want})
		}
	}
	return todo, skipped, nil
}

func (r shellCompletions) Reconcile(env Env, dryRun bool) (Result, error) {
	if r.run == nil {
		return Result{Skip: "no runner is wired into this registry"}, nil
	}
	if !r.has("zsh") {
		return Result{Skip: "zsh is not installed"}, nil
	}
	todo, skipped, err := r.plan(env)
	if err != nil {
		return Result{}, err
	}
	if len(skipped) == len(completionTools) {
		return Result{Skip: strings.Join(skipped, " and ") + " not on PATH"}, nil
	}
	res := Result{Changes: len(todo), Detail: completionsDetail(todo, skipped, dryRun)}
	if dryRun {
		return res, nil
	}
	for _, f := range todo {
		if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil { //nolint:gosec // a completions dir is world-readable
			return Result{}, err
		}
		if err := secrets.AtomicWriteMode(f.path, f.want, 0o644); err != nil {
			return Result{}, err
		}
	}
	return res, nil
}

// Probe holds the post-condition: every installed tool's file is its completion.
func (r shellCompletions) Probe(env Env) error {
	todo, _, err := r.plan(env)
	if err != nil {
		return err
	}
	if len(todo) > 0 {
		return fmt.Errorf("%s still differs from the tool's completion", todo[0].path)
	}
	return nil
}

func completionsDetail(todo []completionFile, skipped []string, dryRun bool) string {
	var parts []string
	if len(todo) > 0 {
		verb := "wrote "
		if dryRun {
			verb = "to write "
		}
		names := make([]string, len(todo))
		for i, f := range todo {
			names[i] = "_" + f.tool
		}
		parts = append(parts, verb+strings.Join(names, ", "))
	} else {
		parts = append(parts, "completions up to date")
	}
	if len(skipped) > 0 {
		parts = append(parts, "skipped "+strings.Join(skipped, ", ")+" (not on PATH)")
	}
	return strings.Join(parts, "; ")
}
