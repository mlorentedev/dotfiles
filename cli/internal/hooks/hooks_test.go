package hooks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeGit stands in for the git binary. Every case but one drives this rather
// than the real thing: `git config --global` has machine-wide blast radius, and
// a test suite that can rewire the developer's global config is not a test
// suite. The single integration test at the bottom is what proves this fake
// speaks the real binary's dialect.
type fakeGit struct {
	hooksPath string // what `--get core.hooksPath` returns; "" means unset
	setErr    error
	writes    []string
	reads     int
}

func (f *fakeGit) run(_ context.Context, args ...string) ([]byte, error) {
	joined := strings.Join(args, " ")
	switch {
	case strings.HasPrefix(joined, "config --global --get core.hooksPath"):
		f.reads++
		if f.hooksPath == "" {
			// git exits 1 on an unset key and prints nothing. Treating that as
			// an error rather than as empty output is the dialect detail the
			// integration test exists to confirm.
			return nil, errors.New("exit status 1")
		}
		return []byte(f.hooksPath + "\n"), nil
	case strings.HasPrefix(joined, "config --global core.hooksPath"):
		if f.setErr != nil {
			return nil, f.setErr
		}
		f.writes = append(f.writes, joined)
		f.hooksPath = args[len(args)-1]
		return nil, nil
	}
	return nil, fmt.Errorf("fakeGit: unexpected args %q", joined)
}

// dispatcherTree writes a structurally valid GUARD-001 dispatcher: the
// pre-commit entrypoint AND the memory-sink guard it execs. isGuardDispatcher is
// structural for the reason recorded in BUG-040 — a marker grep would call a
// dispatcher valid whose guard script is missing, so it could not run.
func dispatcherTree(t *testing.T, dir string, extra map[string]string) string {
	t.Helper()
	files := map[string]string{
		"pre-commit":               "#!/usr/bin/env bash\nexec \"$(dirname \"$0\")/lib/memory-sink-guard.sh\"\n",
		"commit-msg":               "#!/usr/bin/env bash\n",
		"prepare-commit-msg":       "#!/usr/bin/env bash\n",
		"pre-push":                 "#!/usr/bin/env bash\n",
		"post-checkout":            "#!/usr/bin/env bash\n",
		"lib/memory-sink-guard.sh": "#!/usr/bin/env bash\nexit 0\n",
	}
	for k, v := range extra {
		files[k] = v
	}
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func opts(src, dotfiles string, out *bytes.Buffer) Options {
	return Options{Source: src, DotfilesDir: dotfiles, Out: out}
}

// --- Destination refusals -------------------------------------------------
//
// First, because these are what stand between a misconfigured DOTFILES_DIR and
// the clean mirror's rm -rf, and because they are where the two shell suites
// disagreed most: the drive-root case existed only in Pester, the $HOME and
// root cases only in bats.

func TestInstallRefusesUnsafeDestinations(t *testing.T) {
	home := t.TempDir()
	src := dispatcherTree(t, t.TempDir(), nil)

	cases := []struct {
		name        string
		dotfilesDir string
		wantSubstr  string
		windowsOnly bool
	}{
		// Each case asserts its OWN diagnosis, not merely that something was
		// refused. Asserting a shared "unsafe" made the empty-dir guard
		// undetectable by mutation: remove it and filepath.Clean("") == "."
		// falls into the root branch, which refuses too — correctly, but while
		// telling a user with an unset DOTFILES_DIR that "." is a filesystem
		// root. A guard is worth what its message is worth.
		{name: "empty dotfiles dir", dotfilesDir: "", wantSubstr: "empty"},
		{name: "filesystem root", dotfilesDir: string(filepath.Separator), wantSubstr: "filesystem root"},
		{name: "the home directory itself", dotfilesDir: home, wantSubstr: "home directory"},
		{
			// Pester covered this and bats did not: on Windows a drive root is
			// not "/" and the POSIX check alone would let C:\git-hooks through
			// to the rm -rf.
			name: "a drive root", dotfilesDir: "C:\\", wantSubstr: "filesystem root", windowsOnly: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.windowsOnly && runtime.GOOS != "windows" {
				t.Skip("drive roots only exist on Windows")
			}
			var buf bytes.Buffer
			o := opts(src, tc.dotfilesDir, &buf)
			o.homeDir = home
			err := install(context.Background(), (&fakeGit{}).run, o)
			if err == nil {
				t.Fatalf("want a refusal for %q, got nil — this guard is what stops rm -rf", tc.dotfilesDir)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("error %q should say %q", err, tc.wantSubstr)
			}
		})
	}
}

// TestDeployRefusesADestinationOutsideGitHooks pins the shape check both twins
// carry: the mirror target must be a */git-hooks path. It is defence in depth —
// `install` only ever builds `<dotfilesDir>/git-hooks`, so this guard exists for
// a direct caller, and it is tested at that level rather than through a seam
// invented to reach it.
func TestDeployRefusesADestinationOutsideGitHooks(t *testing.T) {
	src := dispatcherTree(t, t.TempDir(), nil)
	var buf bytes.Buffer
	err := deployHooks(src, filepath.Join(t.TempDir(), "not-hooks"), &buf)
	if err == nil || !strings.Contains(err.Error(), "git-hooks") {
		t.Fatalf("want a refusal naming the required */git-hooks shape, got %v", err)
	}
}

// --- The #695 self-mirror case --------------------------------------------

// TestInstallDoesNotDestroyItsOwnSource is #695: rm -rf dest followed by
// cp src/. dest empties the dispatcher and copies nothing back when the two are
// the same directory — while still reporting success. Comparing cleaned strings
// would pass a trailing-slash variant of this test and still miss a symlinked
// mirror, so the check is os.SameFile.
func TestInstallDoesNotDestroyItsOwnSource(t *testing.T) {
	dotfiles := t.TempDir()
	dest := filepath.Join(dotfiles, "git-hooks")
	dispatcherTree(t, dest, nil)

	var buf bytes.Buffer
	// Source IS the destination — the exact input #695 was filed for.
	if err := install(context.Background(), (&fakeGit{}).run, opts(dest, dotfiles, &buf)); err != nil {
		t.Fatalf("install: %v", err)
	}

	for _, want := range []string{"pre-commit", filepath.Join("lib", "memory-sink-guard.sh")} {
		if _, err := os.Stat(filepath.Join(dest, want)); err != nil {
			t.Fatalf("%s is gone — the mirror deleted its own source: %v", want, err)
		}
	}
}

// TestInstallHandlesAFirstInstall covers the case os.SameFile cannot: on a first
// install the destination does not exist, so stat fails and the self-mirror
// check must short-circuit to "not the same" rather than surface the error. The
// bats test for #695 assumes both paths exist and would not catch this.
func TestInstallHandlesAFirstInstall(t *testing.T) {
	src := dispatcherTree(t, t.TempDir(), nil)
	dotfiles := t.TempDir()

	var buf bytes.Buffer
	if err := install(context.Background(), (&fakeGit{}).run, opts(src, dotfiles, &buf)); err != nil {
		t.Fatalf("first install must succeed against a non-existent destination: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dotfiles, "git-hooks", "pre-commit")); err != nil {
		t.Fatalf("dispatcher not deployed: %v", err)
	}
}

// --- Source refusals -------------------------------------------------------

func TestInstallRefusesABadSource(t *testing.T) {
	t.Run("missing source directory", func(t *testing.T) {
		var buf bytes.Buffer
		err := install(context.Background(), (&fakeGit{}).run,
			opts(filepath.Join(t.TempDir(), "nope"), t.TempDir(), &buf))
		if err == nil || !strings.Contains(err.Error(), "source") {
			t.Fatalf("want a refusal naming the source, got %v", err)
		}
	})

	t.Run("source without a pre-commit dispatcher", func(t *testing.T) {
		src := t.TempDir()
		if err := os.WriteFile(filepath.Join(src, "commit-msg"), []byte("#!/bin/sh\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		err := install(context.Background(), (&fakeGit{}).run, opts(src, t.TempDir(), &buf))
		if err == nil || !strings.Contains(err.Error(), "pre-commit") {
			t.Fatalf("want a refusal naming pre-commit, got %v", err)
		}
	})
}

// --- The mirror ------------------------------------------------------------

// TestInstallCleanMirrorsRatherThanMerging pins why this is not a bare copy: a
// hook removed upstream must stop firing. A stale security hook is worse than
// no hook, because it is trusted.
func TestInstallCleanMirrorsRatherThanMerging(t *testing.T) {
	src := dispatcherTree(t, t.TempDir(), nil)
	dotfiles := t.TempDir()
	dest := filepath.Join(dotfiles, "git-hooks")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dest, "pre-rebase")
	if err := os.WriteFile(stale, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := install(context.Background(), (&fakeGit{}).run, opts(src, dotfiles, &buf)); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("a hook removed upstream survived the mirror and keeps firing (err=%v)", err)
	}
}

// TestInstallNormalisesCRLF is BUG-068: a copy is byte-verbatim, so a
// CRLF-tainted checkout propagates "#!/usr/bin/env bash\r" into the mirror, bash
// resolves the interpreter "bash\r", and every hook dies "No such file or
// directory". The .gitattributes eol=lf rule keeps the SOURCE clean; this keeps
// the DEPLOYED copy clean whatever the source looks like.
func TestInstallNormalisesCRLF(t *testing.T) {
	src := dispatcherTree(t, t.TempDir(), map[string]string{
		"pre-commit": "#!/usr/bin/env bash\r\nexec guard\r\n",
	})
	dotfiles := t.TempDir()

	var buf bytes.Buffer
	if err := install(context.Background(), (&fakeGit{}).run, opts(src, dotfiles, &buf)); err != nil {
		t.Fatalf("install: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dotfiles, "git-hooks", "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got, []byte("\r")) {
		t.Fatalf("deployed pre-commit still carries CR: %q", got)
	}
}

// TestInstallMakesEntrypointsExecutable — git execs these directly. The bit is
// inert on Windows, so only the Linux leg can regress silently, which is exactly
// why it is asserted rather than assumed.
func TestInstallMakesEntrypointsExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no executable bit on Windows; git-for-windows runs the dispatchers through sh")
	}
	src := dispatcherTree(t, t.TempDir(), nil)
	dotfiles := t.TempDir()

	var buf bytes.Buffer
	if err := install(context.Background(), (&fakeGit{}).run, opts(src, dotfiles, &buf)); err != nil {
		t.Fatalf("install: %v", err)
	}
	for _, name := range []string{"pre-commit", "commit-msg", "prepare-commit-msg", "pre-push", "post-checkout",
		filepath.Join("lib", "memory-sink-guard.sh")} {
		fi, err := os.Stat(filepath.Join(dotfiles, "git-hooks", filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if fi.Mode()&0o111 == 0 {
			t.Errorf("%s is not executable (mode %v) — git will refuse to run it", name, fi.Mode())
		}
	}
}

// --- core.hooksPath wiring -------------------------------------------------

func TestInstallWiresHooksPath(t *testing.T) {
	src := dispatcherTree(t, t.TempDir(), nil)

	t.Run("unset gets wired to the deployed dispatcher", func(t *testing.T) {
		dotfiles := t.TempDir()
		git := &fakeGit{}
		var buf bytes.Buffer
		if err := install(context.Background(), git.run, opts(src, dotfiles, &buf)); err != nil {
			t.Fatalf("install: %v", err)
		}
		want := filepath.Join(dotfiles, "git-hooks")
		if len(git.writes) != 1 || !strings.HasSuffix(git.writes[0], want) {
			t.Fatalf("want one write of %q, got %v", want, git.writes)
		}
	})

	t.Run("already wired is a no-op, not a rewrite", func(t *testing.T) {
		dotfiles := t.TempDir()
		git := &fakeGit{hooksPath: filepath.Join(dotfiles, "git-hooks")}
		var buf bytes.Buffer
		if err := install(context.Background(), git.run, opts(src, dotfiles, &buf)); err != nil {
			t.Fatalf("install: %v", err)
		}
		if len(git.writes) != 0 {
			t.Fatalf("an already-correct value must not be rewritten, got %v", git.writes)
		}
	})

	t.Run("a trailing-slash variant counts as already wired", func(t *testing.T) {
		dotfiles := t.TempDir()
		git := &fakeGit{hooksPath: filepath.Join(dotfiles, "git-hooks") + string(filepath.Separator)}
		var buf bytes.Buffer
		if err := install(context.Background(), git.run, opts(src, dotfiles, &buf)); err != nil {
			t.Fatalf("install: %v", err)
		}
		if len(git.writes) != 0 {
			t.Fatalf("git treats these as the same path; it must not be rewritten: %v", git.writes)
		}
		// Asserting only "no write" cannot see this guard: under a byte
		// comparison the trailing-slash value falls through to the
		// equivalent-dispatcher branch, which ALSO declines to write — the
		// deployed tree is a valid dispatcher, just reached by another spelling.
		// The two outcomes differ only in what the user is told, so that is what
		// the test reads.
		if out := buf.String(); !strings.Contains(out, "already wired") {
			t.Errorf("want this recognised as the same path, not as a foreign dispatcher; got:\n%s", out)
		}
	})

	t.Run("an equivalent dispatcher elsewhere is active, not INACTIVE", func(t *testing.T) {
		// BUG-040: developing the hooks from a checkout points hooksPath at an
		// equivalent dispatcher. That guard RUNS, so reporting it inactive was
		// wrong on every run.
		elsewhere := dispatcherTree(t, t.TempDir(), nil)
		dotfiles := t.TempDir()
		git := &fakeGit{hooksPath: elsewhere}
		var buf bytes.Buffer
		if err := install(context.Background(), git.run, opts(src, dotfiles, &buf)); err != nil {
			t.Fatalf("install: %v", err)
		}
		if len(git.writes) != 0 {
			t.Fatalf("an active equivalent dispatcher must be preserved, got %v", git.writes)
		}
		if out := buf.String(); !strings.Contains(out, "active") || strings.Contains(out, "INACTIVE") {
			t.Errorf("want the equivalent dispatcher reported active, got:\n%s", out)
		}
	})

	t.Run("an unrelated value is preserved and warned about", func(t *testing.T) {
		// Machine-wide blast radius: clobbering someone's hooksPath is worse
		// than leaving the guard inactive, so this warns and does not write.
		unrelated := t.TempDir()
		dotfiles := t.TempDir()
		git := &fakeGit{hooksPath: unrelated}
		var buf bytes.Buffer
		if err := install(context.Background(), git.run, opts(src, dotfiles, &buf)); err != nil {
			t.Fatalf("install: %v", err)
		}
		if len(git.writes) != 0 {
			t.Fatalf("an unrelated core.hooksPath must never be clobbered, got %v", git.writes)
		}
		if out := buf.String(); !strings.Contains(out, "INACTIVE") {
			t.Errorf("want the user told the guard is inactive, got:\n%s", out)
		}
	})
}

// TestInstallReportsAWiringFailure — a hooksPath that could not be set means the
// guard does not run, and reporting success there is the "cannot compute reads
// as nothing pending" failure in another costume.
func TestInstallReportsAWiringFailure(t *testing.T) {
	src := dispatcherTree(t, t.TempDir(), nil)
	git := &fakeGit{setErr: errors.New("exit status 128")}
	var buf bytes.Buffer
	err := install(context.Background(), git.run, opts(src, t.TempDir(), &buf))
	if err == nil {
		t.Fatal("a failed core.hooksPath write must surface, not be swallowed")
	}
}

// --- The one integration test ---------------------------------------------

// TestInstallAgainstRealGit is the only case that runs the real binary, and it
// exists to prove the fake speaks its dialect — specifically that `--get` on an
// unset key exits non-zero rather than printing nothing. CLI-071 shipped a fake
// that agreed with itself about a login spelling; the same trap applies here.
//
// GIT_CONFIG_GLOBAL points at a throwaway file, so this cannot touch the
// developer's real ~/.gitconfig.
func TestInstallAgainstRealGit(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	cfg := filepath.Join(t.TempDir(), "gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	src := dispatcherTree(t, t.TempDir(), nil)
	dotfiles := t.TempDir()

	var buf bytes.Buffer
	if err := Install(context.Background(), opts(src, dotfiles, &buf)); err != nil {
		t.Fatalf("Install against real git: %v\n%s", err, buf.String())
	}

	// Read the value back THROUGH git rather than out of the file. The config
	// format is git's, not ours: on Windows it stores
	// `C:\\Users\\...\\git-hooks` with the backslashes escaped, so grepping the
	// raw text for the path finds nothing while the wiring is perfectly correct.
	// That is what this assertion originally did, and it failed on the Windows
	// leg while passing on Linux — measuring the file's spelling instead of the
	// property that matters.
	//
	// Asking git closes the round trip: whatever escaping it applied on write,
	// it undoes on read, which is exactly what git does for the hook it later
	// executes.
	out, err := execGit(context.Background(), "config", "--global", "--get", "core.hooksPath")
	if err != nil {
		raw, _ := os.ReadFile(cfg)
		t.Fatalf("git did not record core.hooksPath (%v); config file holds:\n%s", err, raw)
	}
	got := strings.TrimSpace(string(out))
	want := filepath.Join(dotfiles, "git-hooks")
	if !samePath(got, want) {
		t.Fatalf("core.hooksPath = %q, want %q", got, want)
	}
}
