package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The lint reads what CI's checkout would hold once this tree is committed:
// tracked and untracked-unignored PowerShell files, not ignored ones, and not
// a tracked file deleted from the working tree.
func TestLintFiles_SeesWhatACommitWouldShip(t *testing.T) {
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q")
	write(".gitignore", "ignored.ps1\n")
	write("tracked.ps1", "")
	write("gone.ps1", "")
	write("scripts/mod.psm1", "")
	write("setup.sh", "")
	git("add", ".")
	write("scripts/new.ps1", "")
	write("ignored.ps1", "")
	if err := os.Remove(filepath.Join(repo, "gone.ps1")); err != nil {
		t.Fatal(err)
	}

	got, err := lintFiles(repo)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"scripts/mod.psm1", "scripts/new.ps1", "tracked.ps1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnalyzerPin(t *testing.T) {
	if v, err := analyzerPin([]byte("# pwsh-module: PSScriptAnalyzer\nPSSCRIPTANALYZER_VERSION=1.25.0\n")); err != nil || v != "1.25.0" {
		t.Errorf("got %q, %v", v, err)
	}
	if _, err := analyzerPin([]byte("PSSCRIPTANALYZER_VERSION=1.25.0\n")); err == nil {
		t.Error("an unmarked analyzer pin must be an error: nothing would install it")
	}
}
