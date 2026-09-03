// Package hooks installs the GUARD-001 memory-sink dispatcher and wires git's
// global core.hooksPath at it.
//
// It replaces scripts/install-git-hooks.{sh,ps1} (CLI-072 / #1460). The pair had
// drifted where nothing was watching: 13 bats cases against 9 Pester ones for the
// same behaviours, with the #695 self-mirror guard, the dispatcher-equivalence
// probe and the trailing-slash path comparison verified on Linux only. The guards
// existed on both sides; only their verification did not — and test drift shows
// no symptom until the untested guard is the one that fails.
//
// ADR-020 C7 does not cover this. C7 keeps "the step that provisions the tooling
// itself" in shell; git hooks are downstream configuration, installed when `dotf`
// already exists.
package hooks

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitRunner runs a git subcommand and returns its stdout. It is a seam because
// `git config --global` has machine-wide blast radius: a suite that drove the
// real binary could rewire the developer's ~/.gitconfig, so all but one test
// drives a fake. The exception is TestInstallAgainstRealGit, which exists to
// prove the fake speaks the real dialect.
type gitRunner func(ctx context.Context, args ...string) ([]byte, error)

func execGit(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "git", args...).Output()
}

var execLookPath = exec.LookPath

// entrypoints are the hooks git execs directly. lib/*.sh is handled separately:
// those are sourced or exec'd by the entrypoints, and they need the bit too.
var entrypoints = []string{"pre-commit", "commit-msg", "prepare-commit-msg", "pre-push", "post-checkout"}

// Options configures an install.
type Options struct {
	// Source is the dispatcher tree to mirror, i.e. the checkout's git-hooks/.
	Source string
	// DotfilesDir is the deploy root; the mirror lands at DotfilesDir/git-hooks.
	DotfilesDir string
	// Out receives progress. Never nil after Install fills it in.
	Out io.Writer

	// homeDir is the user's home, used only by the unsafe-destination check.
	// Unexported so callers cannot vary it in production; tests set it because
	// t.TempDir() is not the real home and the check must still fire.
	homeDir string
}

// Install deploys the dispatcher and wires core.hooksPath.
func Install(ctx context.Context, o Options) error {
	if o.homeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home directory: %w", err)
		}
		o.homeDir = home
	}
	return install(ctx, execGit, o)
}

func install(ctx context.Context, run gitRunner, o Options) error {
	if o.Out == nil {
		o.Out = io.Discard
	}
	if err := checkDotfilesDir(o.DotfilesDir, o.homeDir); err != nil {
		return err
	}
	dest := filepath.Join(o.DotfilesDir, "git-hooks")
	if err := deployHooks(o.Source, dest, o.Out); err != nil {
		return err
	}
	return wireHooksPath(ctx, run, dest, o.Out)
}

// checkDotfilesDir refuses a deploy root that would turn the clean mirror's
// removal loose. Empty, a filesystem or drive root, and the home directory
// itself are all rejected: `rm -rf $HOME/git-hooks` is survivable, but the value
// arriving empty or as "/" is how a misconfigured DOTFILES_DIR becomes a
// destructive command.
//
// The root test is `filepath.Dir(p) == p`, which is true of "/" on POSIX and of
// "C:\" on Windows — one check rather than a POSIX one that lets drive roots
// through, which is the asymmetry the two shell suites had.
func checkDotfilesDir(dir, home string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("unsafe dotfiles dir: empty")
	}
	clean := filepath.Clean(dir)
	if filepath.Dir(clean) == clean {
		return fmt.Errorf("unsafe dotfiles dir: %q is a filesystem root", dir)
	}
	if home != "" && clean == filepath.Clean(home) {
		return fmt.Errorf("unsafe dotfiles dir: %q is the home directory itself", dir)
	}
	return nil
}

// deployHooks clean-mirrors the dispatcher tree into dest and makes the
// entrypoints executable.
//
// A clean mirror rather than a copy-over: a hook removed upstream must stop
// firing, and a stale security hook is worse than no hook because it is trusted.
func deployHooks(src, dest string, out io.Writer) error {
	if fi, err := os.Stat(src); err != nil || !fi.IsDir() {
		return fmt.Errorf("source dispatcher dir not found: %s", src)
	}
	if _, err := os.Stat(filepath.Join(src, "pre-commit")); err != nil {
		return fmt.Errorf("source %s has no pre-commit dispatcher — refusing to deploy", src)
	}
	// Defence in depth for a direct caller: install only ever builds
	// <dotfilesDir>/git-hooks, but this function removes a tree and must be
	// safe on its own terms.
	if filepath.Base(filepath.Clean(dest)) != "git-hooks" {
		return fmt.Errorf("refusing to mirror to %q: the destination must be a git-hooks directory", dest)
	}

	// #695: with src == dest, "remove then copy" empties the dispatcher and
	// copies nothing back, while still reporting success. os.SameFile rather
	// than a string compare, because a cleaned-string equality passes the
	// trailing-slash case and still misses a symlinked mirror.
	same, err := sameDir(src, dest)
	if err != nil {
		return err
	}
	if same {
		_, _ = fmt.Fprintf(out, "[INFO] src and dest are the same directory (%s) — skipping mirror\n", dest)
		return markExecutable(dest)
	}

	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("clear %s: %w", dest, err)
	}
	if err := copyTree(src, dest); err != nil {
		return err
	}
	if err := markExecutable(dest); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "[OK] GUARD dispatcher deployed to %s\n", dest)
	return nil
}

// sameDir reports whether two paths are the same directory. A destination that
// does not exist yet is not an error and is not the same directory — that is the
// first-install case, which stat cannot answer.
func sameDir(a, b string) (bool, error) {
	fa, err := os.Stat(a)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", a, err)
	}
	fb, err := os.Stat(b)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", b, err)
	}
	return os.SameFile(fa, fb), nil
}

// copyTree mirrors src into dest, stripping CR from text files as it goes.
//
// BUG-068: a byte-verbatim copy propagates "#!/usr/bin/env bash\r" from a
// CRLF-tainted checkout into the mirror; bash then resolves the interpreter
// "bash\r" and every hook dies with "No such file or directory". The
// .gitattributes eol=lf rule keeps the source clean — this keeps the deployed
// copy clean whatever the source looks like.
func copyTree(src, dest string) error {
	return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if isText(body) {
			body = stripCR(body)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
}

// isText mirrors the shell's `grep -I`: a NUL byte means binary, and a binary
// file must not have its CRs stripped.
func isText(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return false
		}
	}
	return true
}

func stripCR(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c != '\r' {
			out = append(out, c)
		}
	}
	return out
}

// markExecutable sets the bit on the hooks git execs and on the lib helpers they
// call. Inert on Windows, where git-for-windows runs the dispatchers through sh —
// which is why only the Linux leg can regress here, and why it is asserted.
func markExecutable(dest string) error {
	for _, name := range entrypoints {
		p := filepath.Join(dest, name)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := os.Chmod(p, 0o755); err != nil {
			return fmt.Errorf("chmod %s: %w", p, err)
		}
	}
	libs, err := filepath.Glob(filepath.Join(dest, "lib", "*.sh"))
	if err != nil {
		return err
	}
	for _, p := range libs {
		if err := os.Chmod(p, 0o755); err != nil {
			return fmt.Errorf("chmod %s: %w", p, err)
		}
	}
	return nil
}

// isGuardDispatcher reports whether dir holds a GUARD-001 dispatcher.
//
// Structural, not a marker grep (BUG-040): pre-commit execs
// lib/memory-sink-guard.sh, so without that script the guard cannot run whatever
// the files claim about themselves.
func isGuardDispatcher(dir string) bool {
	if dir == "" {
		return false
	}
	for _, f := range []string{"pre-commit", filepath.Join("lib", "memory-sink-guard.sh")} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			return false
		}
	}
	return true
}

// wireHooksPath points git's global core.hooksPath at the deployed dispatcher,
// but only when it is unset.
//
// An unrelated value is preserved and surfaced, never clobbered: a global
// hooksPath has machine-wide blast radius, and taking someone's over is worse
// than leaving the guard inactive and saying so.
//
// "Correct" means the guard RUNS, not that the path equals the mirror (BUG-040).
// Developing the hooks from a checkout points hooksPath at an equivalent
// dispatcher, which is active and used to be reported INACTIVE on every run.
func wireHooksPath(ctx context.Context, run gitRunner, target string, out io.Writer) error {
	current := ""
	if b, err := run(ctx, "config", "--global", "--get", "core.hooksPath"); err == nil {
		current = strings.TrimSpace(string(b))
	}
	// An unset key makes git exit non-zero with no output, which is not an
	// error here — it is the case this function exists to fix.

	switch {
	case current == "":
		if _, err := run(ctx, "config", "--global", "core.hooksPath", target); err != nil {
			return fmt.Errorf("set core.hooksPath: %w", err)
		}
		_, _ = fmt.Fprintf(out, "[OK] core.hooksPath wired to the GUARD dispatcher (%s)\n", target)
	case samePath(current, target):
		_, _ = fmt.Fprintln(out, "[INFO] core.hooksPath already wired to the GUARD dispatcher")
	case isGuardDispatcher(current):
		_, _ = fmt.Fprintf(out, "[OK] GUARD dispatcher active via %s (not the deploy mirror %s)\n", current, target)
	default:
		_, _ = fmt.Fprintf(out, "[WARN] core.hooksPath is %q (not the GUARD dispatcher) — preserving it; "+
			"the memory-sink guard is INACTIVE. Point it at %s, or chain the dispatcher from your hooks, manually.\n",
			current, target)
	}
	return nil
}

// samePath compares two hooksPath values as directories rather than as bytes.
// git treats a trailing separator as the same directory, and so must this: a
// byte comparison would rewrite a config that is already correct.
//
// Identity on disk is the single mechanism, deliberately. A cleaned-string
// fast path was written first and removed: it made the common case two syscalls
// cheaper and no test could tell whether it was there, because wiring always
// runs after the deploy, so both paths exist and os.SameFile already answers.
// A branch nothing can distinguish is a branch that will rot.
func samePath(a, b string) bool {
	same, err := sameDir(a, b)
	return err == nil && same
}
