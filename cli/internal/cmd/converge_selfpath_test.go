package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeDotf places an executable named dotf in a new directory and returns it.
func fakeDotf(t *testing.T) string { return fakeBinary(t, "dotf") }

func fakeBinary(t *testing.T, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func pathOf(dirs ...string) string { return strings.Join(dirs, string(os.PathListSeparator)) }

// A from-zero install.sh execs converge by absolute path on a PATH without
// ~/.local/bin, and compile-harness.sh then finds no dotf (#2688). Each row is
// the dotf PATH resolves before the run, and whether converge must put its own
// directory first.
func TestEnsureSelfOnPath(t *testing.T) {
	self := fakeDotf(t)
	other := fakeDotf(t)
	link := filepath.Join(t.TempDir(), filepath.Base(self))
	symlinked := os.Symlink(self, link) == nil
	empty := t.TempDir()

	cases := []struct {
		name    string
		path    string
		prepend bool
	}{
		{"no dotf on PATH", pathOf(empty), true},
		{"empty PATH", "", true},
		{"PATH resolves another dotf first", pathOf(filepath.Dir(other), filepath.Dir(self)), true},
		{"PATH resolves the running dotf", pathOf(empty, filepath.Dir(self)), false},
	}
	if symlinked {
		cases = append(cases, struct {
			name    string
			path    string
			prepend bool
		}{"PATH resolves a link to the running dotf", pathOf(filepath.Dir(link)), false})
	}
	// A dotf renamed for a test run (dotf-dev) is not what its children find
	// by name, so its directory would only lengthen PATH.
	t.Run("a binary not named dotf", func(t *testing.T) {
		t.Setenv("PATH", empty)
		ensureSelfOnPath(fakeBinary(t, "dotf-dev"))
		if got := os.Getenv("PATH"); got != empty {
			t.Errorf("PATH = %q, want it unchanged (%q)", got, empty)
		}
	})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("PATH", c.path)
			ensureSelfOnPath(self)
			got := os.Getenv("PATH")
			want := c.path
			if c.prepend {
				want = filepath.Dir(self)
				if c.path != "" {
					want = pathOf(want, c.path)
				}
			}
			if got != want {
				t.Errorf("PATH = %q, want %q", got, want)
			}
		})
	}
}

// The wiring: a converge started on a PATH that cannot find dotf leaves the
// running binary's directory first on PATH, which every child inherits.
func TestRunConverge_PutsTheRunningDotfFirstOnPath(t *testing.T) {
	repo, _ := convergeFixture(t)
	empty := t.TempDir()
	t.Setenv("PATH", empty)
	self := fakeDotf(t)
	saved := executable
	executable = func() (string, error) { return self, nil }
	t.Cleanup(func() { executable = saved })
	_ = runConverge(&strings.Builder{}, repo, true, []string{"checkout"})
	want := pathOf(filepath.Dir(self), empty)
	if got := os.Getenv("PATH"); got != want {
		t.Errorf("PATH after converge = %q, want %q", got, want)
	}
}
