// Package gitconfig converges what the machine's global git configuration
// needs from the dotfiles, without owning the file git and gh also write
// (#2207).
//
// ~/.gitconfig is co-owned: `git config --global`, `gh auth setup-git` and
// `dotf hooks install` all write it, so a deploy that replaces it erases them
// (lesson 366). The repo's settings live in their own file, deployed by
// `dotf deploy` to IncludeFile, and ~/.gitconfig only gains an include of it,
// written through git itself. The include file is not a path git reads on its
// own: were it ~/.config/git/config, a `git config --global` on a machine with
// no ~/.gitconfig would write into the deployed file, and the next deploy
// would erase that write.
//
// The credential helper for GitHub is gh's to write. A bare `!gh auth
// git-credential` resolves gh through PATH, which a GUI app, launchd, cron or
// a scheduled task does not share with the interactive shell; a literal path
// is right on one OS only (audit-008 C42). `gh auth setup-git` writes the
// absolute path of the gh that runs it, on every OS.
package gitconfig

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// IncludePath is how ~/.gitconfig names the deployed file. git expands a
// leading ~/ in include.path on every OS, so one spelling serves all of them.
const IncludePath = "~/.config/git/dotfiles.gitconfig"

// IncludeFile is IncludePath under home: the `gitconfig` entry's destination
// in ai/deploy.json.
func IncludeFile(home string) string {
	return filepath.Join(home, ".config", "git", "dotfiles.gitconfig")
}

// CredentialHosts are the hosts `gh auth setup-git` configures.
var CredentialHosts = []string{"https://github.com", "https://gist.github.com"}

// Runner runs a command and returns its stdout. The error keeps the command's
// exit status (an error with an ExitCode method), which is how git says a key
// is absent.
type Runner func(name string, args ...string) ([]byte, error)

// ExecRunner is the production Runner: the command on PATH, its stdout.
func ExecRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output() //nolint:gosec // git or gh, fixed arguments
}

// envTokens are the variables gh reads a token from before its stored login.
var envTokens = []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}

// WithoutEnvTokens is environ minus the variables gh would take a token from,
// so gh answers for the login it stored, which is the one a GUI app, launchd,
// cron or a scheduled task can use (#2319).
func WithoutEnvTokens(environ []string) []string {
	out := make([]string, 0, len(environ))
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if !slices.Contains(envTokens, name) {
			out = append(out, kv)
		}
	}
	return out
}

// StoredLoginRunner is the production Machine.StoredAuth: the command on PATH
// with WithoutEnvTokens(os.Environ()), its stdout.
func StoredLoginRunner(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...) //nolint:gosec // gh, fixed arguments
	cmd.Env = WithoutEnvTokens(os.Environ())
	return cmd.Output()
}

// Machine is what Inspect and Apply need: the runner and the questions they
// ask the filesystem and PATH. Every field is a seam for tests (lesson 335).
type Machine struct {
	Home   string
	Run    Runner
	OnPath func(name string) bool
	Exists func(path string) bool
	// StoredAuth runs `gh auth status` with the environment's tokens removed
	// (StoredLoginRunner in production). A GH_TOKEN in this process is not a
	// login the helper can use from a GUI app or a scheduled task, and the
	// two callers did not agree on it: CI's setup step has no token and saw a
	// blocked helper, its doctor step has one and saw a repairable helper
	// nobody repaired (#2319). Nil means Run.
	StoredAuth Runner
}

// State is the global git configuration as far as the dotfiles care.
type State struct {
	// IncludeMissing: ~/.gitconfig does not include the deployed file.
	IncludeMissing bool
	// TargetMissing: the deployed file itself is absent, so an include reads
	// nothing. `dotf deploy` writes it; Apply does not.
	TargetMissing bool
	// BadHelpers names each host whose helper is not gh's absolute form, with
	// the helper git would run.
	BadHelpers []string
	// Blocked says why the helpers cannot be repaired here, or "": gh is not on
	// PATH, or not logged in to github.com.
	Blocked string
}

// Converged reports whether git reads the dotfiles' settings and runs gh's
// absolute helper. A blocked helper or an undeployed file is not converged
// either: Apply cannot fix them, and the caller names the remedy.
func (s State) Converged() bool {
	return !s.IncludeMissing && !s.TargetMissing && len(s.BadHelpers) == 0
}

// Repairable is what Apply would change on this machine now.
func (s State) Repairable() int {
	n := 0
	if s.IncludeMissing {
		n++
	}
	if len(s.BadHelpers) > 0 && s.Blocked == "" {
		n++
	}
	return n
}

// Inspect reads the global configuration and changes nothing.
func Inspect(m Machine) (State, error) {
	var st State
	includes, err := getAll(m.Run, "include.path")
	if err != nil {
		return st, err
	}
	st.IncludeMissing = !hasInclude(includes, m.Home)
	st.TargetMissing = !m.Exists(IncludeFile(m.Home))

	for _, host := range CredentialHosts {
		helpers, err := getAll(m.Run, "credential."+host+".helper")
		if err != nil {
			return st, err
		}
		eff := effective(helpers)
		if len(eff) == 1 && IsAbsoluteGHHelper(eff[0], m.Exists) {
			continue
		}
		st.BadHelpers = append(st.BadHelpers, host+": "+describe(eff))
	}
	if len(st.BadHelpers) > 0 {
		st.Blocked = ghBlocked(m)
	}
	return st, nil
}

// Apply converges what st reports and m can repair: the include through
// `git config --global --add`, the helpers through `gh auth setup-git`. A
// blocked helper is left as it is; Inspect's Blocked names why.
func Apply(m Machine, st State) error {
	if st.IncludeMissing {
		if _, err := m.Run("git", "config", "--global", "--add", "include.path", IncludePath); err != nil {
			return fmt.Errorf("add include.path %s: %w", IncludePath, err)
		}
	}
	if len(st.BadHelpers) > 0 && st.Blocked == "" {
		if _, err := m.Run("gh", "auth", "setup-git"); err != nil {
			return fmt.Errorf("gh auth setup-git: %w", err)
		}
	}
	return nil
}

// IsAbsoluteGHHelper reports whether a credential helper value runs gh by an
// absolute path to a file that exists: `!<path> auth git-credential`, with the
// path optionally quoted (gh quotes it when it holds a space, as on Windows).
// That is the form a process with no shell PATH can still run.
func IsAbsoluteGHHelper(value string, exists func(string) bool) bool {
	rest, ok := strings.CutPrefix(strings.TrimSpace(value), "!")
	if !ok {
		return false
	}
	path, args := firstWord(strings.TrimSpace(rest))
	if strings.Join(strings.Fields(args), " ") != "auth git-credential" {
		return false
	}
	// Either separator: gh on Windows writes a backslash path, and this is
	// also checked from other OSes' tests.
	base := strings.TrimSuffix(strings.ToLower(path[strings.LastIndexAny(path, `/\`)+1:]), ".exe")
	return base == "gh" && isAbs(path) && exists(path)
}

// isAbs accepts a Windows drive path on every OS: the value is what gh wrote
// on the machine it runs on, and a test on one OS checks the others' forms.
func isAbs(p string) bool {
	if filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
		return true
	}
	return len(p) > 2 && p[1] == ':' && (p[2] == '\\' || p[2] == '/')
}

// firstWord splits off a leading word, honouring one level of single or
// double quotes.
func firstWord(s string) (word, rest string) {
	if s != "" && (s[0] == '\'' || s[0] == '"') {
		if end := strings.IndexByte(s[1:], s[0]); end >= 0 {
			return s[1 : end+1], s[end+2:]
		}
	}
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i], s[i:]
	}
	return s, ""
}

// effective is the helper list git uses: an empty value resets the list, so
// only what follows the last empty one counts.
func effective(values []string) []string {
	start := 0
	for i, v := range values {
		if strings.TrimSpace(v) == "" {
			start = i + 1
		}
	}
	return values[start:]
}

func describe(helpers []string) string {
	if len(helpers) == 0 {
		return "no helper"
	}
	return strings.Join(helpers, "; ")
}

func hasInclude(paths []string, home string) bool {
	want := filepath.Clean(IncludeFile(home))
	for _, p := range paths {
		if p == IncludePath {
			return true
		}
		if rest, ok := strings.CutPrefix(p, "~/"); ok {
			p = filepath.Join(home, rest)
		}
		if filepath.Clean(filepath.FromSlash(p)) == want {
			return true
		}
	}
	return false
}

func ghBlocked(m Machine) string {
	if !m.OnPath("gh") {
		return "gh is not on PATH"
	}
	// Only the exit status is read: `gh auth status` masks the token, and its
	// output is discarded either way.
	stored := m.StoredAuth
	if stored == nil {
		stored = m.Run
	}
	if _, err := stored("gh", "auth", "status", "--hostname", "github.com"); err != nil {
		return "gh is not logged in to github.com (run: gh auth login; a token in the environment does not count, since a GUI app or scheduled task does not inherit it)"
	}
	return ""
}

// getAll returns every value of a key in the global configuration. git exits 1
// when the key is absent; that is an empty list, not an error.
func getAll(run Runner, key string) ([]string, error) {
	out, err := run("git", "config", "--global", "--get-all", key)
	if err != nil {
		var ec interface{ ExitCode() int }
		if errors.As(err, &ec) && ec.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("git config --global --get-all %s: %w", key, err)
	}
	// Drop only the final newline: a last value that is empty (a reset) prints
	// as a blank last line, and trimming every trailing newline would lose it.
	var vals []string
	for _, line := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
		vals = append(vals, strings.TrimRight(line, "\r"))
	}
	return vals, nil
}
