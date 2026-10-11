package hooks

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// inspect runs Inspect for the host's OS: the executable bit is part of a
// current mirror only where chmod takes effect.
func inspect(t *testing.T, g *fakeGit, src, dotfiles, goos string) State {
	t.Helper()
	st, err := Inspect(context.Background(), g.run, opts(src, dotfiles, nil), goos)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	return st
}

func apply(t *testing.T, g *fakeGit, src, dotfiles string) {
	t.Helper()
	var buf bytes.Buffer
	if err := Apply(context.Background(), g.run, opts(src, dotfiles, &buf)); err != nil {
		t.Fatalf("Apply: %v\n%s", err, buf.String())
	}
}

// A fresh machine has two changes to make, and Inspect makes neither.
func TestInspect_AFreshMachineNeedsTheMirrorAndTheWiring(t *testing.T) {
	src := dispatcherTree(t, t.TempDir(), nil)
	dotfiles := t.TempDir()
	g := &fakeGit{}

	st := inspect(t, g, src, dotfiles, runtime.GOOS)
	if !st.MirrorStale || st.Wiring != WiringUnset || st.Changes() != 2 {
		t.Errorf("got %+v (changes %d), want a stale mirror, unset wiring, 2 changes", st, st.Changes())
	}
	if _, err := os.Stat(st.Dest); !os.IsNotExist(err) {
		t.Errorf("Inspect created the mirror: %v", err)
	}
	if len(g.writes) != 0 {
		t.Errorf("Inspect wrote git config: %v", g.writes)
	}
}

// After an Apply there is nothing left, including for a CRLF source the mirror
// holds with LF: what counts is what a deploy would write.
func TestInspect_AnAppliedMachineIsConverged(t *testing.T) {
	src := dispatcherTree(t, t.TempDir(), map[string]string{"lib/crlf.sh": "#!/usr/bin/env bash\r\necho hi\r\n"})
	dotfiles := t.TempDir()
	g := &fakeGit{}
	apply(t, g, src, dotfiles)

	st := inspect(t, g, src, dotfiles, runtime.GOOS)
	if st.Changes() != 0 || st.Wiring != WiringMirror {
		t.Errorf("got %+v (changes %d), want converged and wired to the mirror", st, st.Changes())
	}
}

// Each way a mirror goes stale: a file edited, a file the source no longer
// has, a file the source gained, and an entrypoint that lost its bit.
func TestInspect_FindsAStaleMirror(t *testing.T) {
	cases := map[string]func(t *testing.T, src, dest string){
		"edited": func(t *testing.T, _, dest string) {
			writeTestFile(t, filepath.Join(dest, "pre-push"), "#!/usr/bin/env bash\nexit 1\n")
		},
		"extra in the mirror": func(t *testing.T, _, dest string) {
			writeTestFile(t, filepath.Join(dest, "lib", "removed-upstream.sh"), "#!/usr/bin/env bash\n")
		},
		"new in the source": func(t *testing.T, src, _ string) {
			writeTestFile(t, filepath.Join(src, "lib", "new.sh"), "#!/usr/bin/env bash\n")
		},
		"bit lost": func(t *testing.T, _, dest string) {
			if runtime.GOOS == "windows" {
				t.Skip("chmod is inert on Windows")
			}
			if err := os.Chmod(filepath.Join(dest, "pre-commit"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			src := dispatcherTree(t, t.TempDir(), nil)
			dotfiles := t.TempDir()
			g := &fakeGit{}
			apply(t, g, src, dotfiles)
			mutate(t, src, filepath.Join(dotfiles, "git-hooks"))

			st := inspect(t, g, src, dotfiles, runtime.GOOS)
			if !st.MirrorStale || st.Changes() != 1 {
				t.Errorf("got %+v (changes %d), want a stale mirror and 1 change", st, st.Changes())
			}
		})
	}
}

// On Windows the bit is not part of a current mirror: chmod cannot set it, so
// counting it would report a change on every run.
func TestInspect_IgnoresTheBitOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a filesystem where chmod takes effect")
	}
	src := dispatcherTree(t, t.TempDir(), nil)
	dotfiles := t.TempDir()
	g := &fakeGit{}
	apply(t, g, src, dotfiles)
	if err := os.Chmod(filepath.Join(dotfiles, "git-hooks", "pre-commit"), 0o644); err != nil {
		t.Fatal(err)
	}
	if st := inspect(t, g, src, dotfiles, "windows"); st.MirrorStale {
		t.Errorf("got a stale mirror on windows from the bit alone: %+v", st)
	}
}

func TestInspect_ClassifiesTheWiring(t *testing.T) {
	src := dispatcherTree(t, t.TempDir(), nil)
	dotfiles := t.TempDir()
	dest := filepath.Join(dotfiles, "git-hooks")
	foreign := t.TempDir()
	cases := []struct {
		name, hooksPath string
		want            Wiring
	}{
		{"unset", "", WiringUnset},
		{"the mirror, before it exists", dest, WiringMirror},
		{"the mirror with a trailing slash", dest + string(filepath.Separator), WiringMirror},
		{"a checkout's dispatcher", src, WiringEquivalent},
		{"someone else's hooks", foreign, WiringForeign},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := inspect(t, &fakeGit{hooksPath: tc.hooksPath}, src, dotfiles, runtime.GOOS)
			if st.Wiring != tc.want {
				t.Errorf("hooksPath %q: got wiring %d, want %d", tc.hooksPath, st.Wiring, tc.want)
			}
			if tc.want == WiringForeign && st.Changes() != 1 {
				t.Errorf("a foreign hooksPath is left as it is, so only the mirror counts; got %d", st.Changes())
			}
		})
	}
}

// Inspect refuses what Install refuses, before it reads anything.
func TestInspect_RefusesWhatInstallRefuses(t *testing.T) {
	src := dispatcherTree(t, t.TempDir(), nil)
	if _, err := Inspect(context.Background(), (&fakeGit{}).run, opts(src, "/", nil), "linux"); err == nil {
		t.Error("a filesystem root as the dotfiles dir must be refused")
	}
	if _, err := Inspect(context.Background(), (&fakeGit{}).run, opts(t.TempDir(), t.TempDir(), nil), "linux"); err == nil {
		t.Error("a source with no pre-commit must be refused")
	}
}

func writeTestFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A read that fails for any reason but "no such key" is an error, in Inspect
// and in the install: read as unset, it would wire over a hooksPath nobody
// could see.
func TestAFailedHooksPathReadIsNotAnUnsetOne(t *testing.T) {
	src := dispatcherTree(t, t.TempDir(), nil)
	for _, readErr := range []error{gitExit(128), errors.New("git: executable file not found")} {
		g := &fakeGit{getErr: readErr}
		if _, err := Inspect(context.Background(), g.run, opts(src, t.TempDir(), nil), runtime.GOOS); err == nil {
			t.Errorf("Inspect with read error %v: want an error", readErr)
		}
		var buf bytes.Buffer
		if err := Apply(context.Background(), g.run, opts(src, t.TempDir(), &buf)); err == nil || len(g.writes) != 0 {
			t.Errorf("Apply with read error %v: err %v, writes %v; want an error and no write", readErr, err, g.writes)
		}
	}
}
