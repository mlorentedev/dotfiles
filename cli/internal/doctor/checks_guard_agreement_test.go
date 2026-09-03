package doctor

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/hooks"
)

// TestDoctorAgreesWithTheInstallerOnTheDispatcherPath is CLI-072's AC6.
//
// `dotf hooks install` writes the dispatcher to <DotfilesDir>/git-hooks and
// checkGuardHooks looks for it at <DotfilesDir>/git-hooks. Today those are two
// independent expressions in two packages that happen to agree — and doctor
// checking a path the installer does not write is a check that passes while the
// guard is absent, or fails while it is present.
//
// So the agreement is asserted by RUNNING both: install into a temp deploy root,
// then point doctor at the same root and require it to be satisfied. Two
// constants compared to each other would pass even if both were wrong together;
// this cannot.
func TestDoctorAgreesWithTheInstallerOnTheDispatcherPath(t *testing.T) {
	home := t.TempDir()
	dotfiles := filepath.Join(home, ".dotfiles")

	// A structurally valid dispatcher: the entrypoint and the guard it execs.
	src := t.TempDir()
	for name, body := range map[string]string{
		"pre-commit":               "#!/usr/bin/env bash\nexec \"$(dirname \"$0\")/lib/memory-sink-guard.sh\"\n",
		"lib/memory-sink-guard.sh": "#!/usr/bin/env bash\nexit 0\n",
	} {
		p := filepath.Join(src, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Real git against a throwaway global config: the installer wires
	// core.hooksPath for real, and nothing touches the developer's ~/.gitconfig.
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	var installOut bytes.Buffer
	if err := hooks.Install(context.Background(), hooks.Options{
		Source: src, DotfilesDir: dotfiles, Out: &installOut,
	}); err != nil {
		t.Fatalf("hooks.Install: %v\n%s", err, installOut.String())
	}

	// Now ask doctor about the same deploy root, with its git probe answering
	// what the installer just wrote.
	wired := filepath.Join(dotfiles, "git-hooks")
	sys := newSys(map[string]string{"HOME": home}, []string{"git"},
		map[string]string{"git config --global --get core.hooksPath": wired})

	var buf bytes.Buffer
	rep := capture(&buf)
	checkGuardHooks(sys, &Config{DotfilesDir: dotfiles}, rep, false)

	if rep.Failures() != 0 {
		t.Fatalf("doctor reports a failure against a tree the installer just deployed — "+
			"the two disagree about where the dispatcher lives:\n%s", buf.String())
	}
}
