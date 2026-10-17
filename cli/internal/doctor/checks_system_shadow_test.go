package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// shadowCatalog has a system entry on every OS (eza), one apt has no name for
// (tea), and an npm tool, which is checkShadowedCatalogTools' to report. The
// npm entry carries the system keys too, so only its type keeps it out.
var shadowCatalog = tools.Catalog{Tools: []tools.Tool{
	{Name: "eza", Source: tools.Source{Type: "system", Apt: "eza", Brew: "eza", Command: "eza"}},
	{Name: "tea", Source: tools.Source{Type: "system", Brew: "tea", Command: "tea"}},
	{Name: "opencode", Source: tools.Source{Type: "npm", Package: "opencode-ai", Apt: "opencode", Command: "opencode"}},
}}

type shadowBox struct {
	sys      *System
	localBin string
	managed  string // apt's eza, "" when the box has none
}

// newShadowBox is a linux box whose PATH lists ~/.local/bin ahead of the
// manager's directory, as the profile files do. runs says whether the manager's
// eza answers --version.
func newShadowBox(t *testing.T, withManaged, runs bool) shadowBox {
	t.Helper()
	home := t.TempDir()
	localBin := filepath.Join(home, ".local", "bin")
	usrBin := filepath.Join(t.TempDir(), "usr", "bin")
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(usrBin, 0o755); err != nil {
		t.Fatal(err)
	}
	b := shadowBox{localBin: localBin}
	cmds := map[string]string{}
	if withManaged {
		b.managed = filepath.Join(usrBin, "eza")
		writeExec(t, b.managed)
		if runs {
			cmds[b.managed+" --version"] = "eza - A modern, maintained replacement for ls\nv0.18.2"
		}
	}
	env := map[string]string{"HOME": home, "PATH": localBin + string(os.PathListSeparator) + usrBin}
	b.sys = newSys(env, nil, cmds)
	b.sys.GOOS = "linux"
	return b
}

func (b shadowBox) run(t *testing.T, fix bool) string {
	t.Helper()
	var buf bytes.Buffer
	checkShadowingSystemCopies(b.sys, shadowCatalog, capture(&buf), fix)
	return buf.String()
}

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

// setup-linux.sh downloaded eza into ~/.local/bin until #2013 W2; that copy
// sits ahead of apt's on PATH and no pin governs it (#2320).
func TestCheckShadowingSystemCopies_ACopyAheadOfTheManagersWarnsNamingBoth(t *testing.T) {
	b := newShadowBox(t, true, true)
	stale := filepath.Join(b.localBin, "eza")
	writeExec(t, stale)
	out := b.run(t, false)
	if !strings.Contains(out, "[WARN]") || !strings.Contains(out, stale) || !strings.Contains(out, b.managed) {
		t.Fatalf("want a WARN naming %s and %s\n%s", stale, b.managed, out)
	}
	if !strings.Contains(out, "dotf doctor --fix") {
		t.Errorf("want the remedy named\n%s", out)
	}
	if !exists(stale) {
		t.Error("without --fix nothing is removed")
	}
}

func TestCheckShadowingSystemCopies_FixRemovesTheCopyOnceTheManagersRuns(t *testing.T) {
	b := newShadowBox(t, true, true)
	stale := filepath.Join(b.localBin, "eza")
	writeExec(t, stale)
	out := b.run(t, true)
	if exists(stale) {
		t.Fatalf("--fix must remove %s\n%s", stale, out)
	}
	if !strings.Contains(out, "removed "+stale) {
		t.Errorf("want the removal reported\n%s", out)
	}
	if again := b.run(t, true); strings.Contains(again, "eza") {
		t.Errorf("a second run must find nothing\n%s", again)
	}
}

// The fix never trades a running tool for a broken one.
func TestCheckShadowingSystemCopies_FixKeepsTheCopyWhenTheManagersDoesNotRun(t *testing.T) {
	b := newShadowBox(t, true, false)
	stale := filepath.Join(b.localBin, "eza")
	writeExec(t, stale)
	out := b.run(t, true)
	if !exists(stale) {
		t.Fatalf("the copy must stay while %s does not run\n%s", b.managed, out)
	}
	if !strings.Contains(out, "[WARN]") || !strings.Contains(out, "does not run") {
		t.Errorf("want a WARN saying the manager's copy does not run\n%s", out)
	}
}

// The presence rule counts any eza on PATH, so a box where apt's never
// arrived (sudo refused) keeps the download forever. Removing it would leave
// no eza at all, so it is named, never removed.
func TestCheckShadowingSystemCopies_TheOnlyCopyIsNamedAndKept(t *testing.T) {
	b := newShadowBox(t, false, false)
	stale := filepath.Join(b.localBin, "eza")
	writeExec(t, stale)
	out := b.run(t, true)
	if !exists(stale) {
		t.Fatalf("the only eza must stay\n%s", out)
	}
	if !strings.Contains(out, "[WARN]") || !strings.Contains(out, stale) || !strings.Contains(out, "apt") {
		t.Errorf("want a WARN naming the copy and apt\n%s", out)
	}
}

// A link is deliberate (or W2c's own); an entry this OS does not install and
// a tool of another source type are not this check's.
func TestCheckShadowingSystemCopies_LeavesLinksAndOtherEntriesAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege the Windows runner lacks")
	}
	b := newShadowBox(t, true, true)
	link := filepath.Join(b.localBin, "eza")
	if err := os.Symlink(b.managed, link); err != nil {
		t.Fatal(err)
	}
	writeExec(t, filepath.Join(b.localBin, "tea"))
	writeExec(t, filepath.Join(b.localBin, "opencode"))
	out := b.run(t, true)
	if strings.Contains(out, "[WARN]") || strings.Contains(out, "removed") {
		t.Errorf("want nothing reported\n%s", out)
	}
	for _, p := range []string{link, filepath.Join(b.localBin, "tea"), filepath.Join(b.localBin, "opencode")} {
		if !exists(p) {
			t.Errorf("%s must stay", p)
		}
	}
}

// The wiring: doctor's System packages section runs the check against the
// catalog it reads.
func TestCheckSystemPackages_NamesACopyShadowingASystemEntry(t *testing.T) {
	b := newShadowBox(t, true, true)
	stale := filepath.Join(b.localBin, "eza")
	writeExec(t, stale)
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "packages.json"), `{"tools":[{"name":"eza","profile":"full","source":{"type":"system","apt":"eza","brew":"eza","command":"eza"}}]}`)
	env := map[string]string{"HOME": b.sys.Getenv("HOME"), "PATH": b.sys.Getenv("PATH"), "DOTFILES_REPO_DIR": repo}
	sys := newSys(env, nil, nil)
	sys.GOOS = "linux"
	var buf bytes.Buffer
	checkSystemPackages(sys, &Config{DotfilesDir: t.TempDir()}, capture(&buf), false)
	if out := buf.String(); !strings.Contains(out, "[WARN]") || !strings.Contains(out, stale) {
		t.Errorf("want the System packages section to name %s\n%s", stale, out)
	}
}
