package vault

// RunHealth against the frozen golden corpus in tests/golden/vault-health/
// (CI-009 / #1745). The corpus was captured from scripts/vault-health.sh and
// was only ever replayed against a built `dotf` by the bats suite
// tests/vault-health-go-parity.bats, which this file replaces: RunHealth had no
// Go test at all. The shell twin and its bats replay were retired under
// CLI-023 (#492), so this test is now the corpus's only consumer and the
// corpus is Go-owned: its ORACLE is a historical provenance record, with no
// re-capture path.
//
// The oracle is three artefacts per case, exactly the ones the retired shell
// replay (tests/golden/vault-health/lib.sh) compared: the exit code, the
// normalised report, and
// every `obsidian` invocation in order. The last one is the reason this does
// not inject a fake runner: four sections shell out to `obsidian`, and a fake
// in-process runner would test a seam the binary never uses. Instead the test
// binary itself plays obsidian (TestMain below), found through a real PATH
// lookup, so exec.LookPath and the argv RunHealth builds are what is checked.

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// The stub's contract, read from its environment. Set only on the process
// TestRunHealthGoldenCorpus spawns through PATH; a normal `go test` never has
// them, so TestMain falls through to the suite.
const (
	stubEnvActive = "DOTF_VAULT_HEALTH_OBSIDIAN_STUB"
	stubEnvCase   = "DOTF_VAULT_HEALTH_STUB_CASE"
	stubEnvLog    = "DOTF_VAULT_HEALTH_STUB_LOG"
)

func TestMain(m *testing.M) {
	if os.Getenv(stubEnvActive) == "1" {
		os.Exit(runObsidianStub(os.Args[1:]))
	}
	os.Exit(m.Run())
}

// runObsidianStub is lib.sh's stub, byte for byte: append the argv to the log,
// take the first argument naming a subcommand (so `--vault knowledge` never
// matches), print stub/<sub> when the case has one, and exit 0.
func runObsidianStub(args []string) int {
	logf, err := os.OpenFile(os.Getenv(stubEnvLog), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		_, _ = io.WriteString(os.Stderr, "obsidian stub: "+err.Error()+"\n")
		return 97
	}
	_, _ = io.WriteString(logf, strings.Join(args, " ")+"\n")
	_ = logf.Close()

	for _, a := range args {
		switch a {
		case "vault", "orphans", "dead-ends", "unresolved", "tags":
			if b, err := os.ReadFile(filepath.Join(os.Getenv(stubEnvCase), "stub", a)); err == nil {
				_, _ = os.Stdout.Write(b)
			}
			return 0
		}
	}
	return 0
}

// goldenRepoRoot is the checkout root, from this package's directory.
func goldenRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func TestRunHealthGoldenCorpus(t *testing.T) {
	root := goldenRepoRoot(t)
	casesDir := filepath.Join(root, "tests", "golden", "vault-health", "cases")
	entries, err := os.ReadDir(casesDir)
	if err != nil {
		t.Fatalf("read the golden corpus: %v", err)
	}

	// Resolved BEFORE PATH is narrowed below. A missing git fails rather than
	// skips: it exists on every CI leg, and a suite that skips when its tools
	// vanish reads as green (#807 / BUG-055).
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("git is required by the worktree cases: %v", err)
	}
	stubBin := installObsidianStub(t)
	emptyBin := t.TempDir()
	basePath := pathWithoutObsidian(os.Getenv("PATH"))

	ran := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ran++
		caseDir := filepath.Join(casesDir, e.Name())
		t.Run(e.Name(), func(t *testing.T) {
			bin := stubBin
			if fileExists(filepath.Join(caseDir, "no-obsidian")) {
				bin = emptyBin
			}
			runGoldenCase(t, goldenCase{
				dir:     caseDir,
				path:    bin + string(os.PathListSeparator) + basePath,
				stubBin: bin,
				gitPath: gitPath,
			})
		})
	}
	if ran == 0 {
		t.Fatalf("no cases under %s: the corpus moved, and a loop over nothing passes", casesDir)
	}
}

type goldenCase struct {
	dir, path, stubBin, gitPath string
}

func runGoldenCase(t *testing.T, c goldenCase) {
	vaultDir := filepath.Join(t.TempDir(), "vault")
	if fileExists(filepath.Join(c.dir, "vault")) {
		copyTree(t, filepath.Join(c.dir, "vault"), vaultDir)
	}
	if mode := readTrimmed(t, filepath.Join(c.dir, "gitmode")); mode != "" {
		seedGit(t, c, vaultDir, mode)
	}

	logPath := filepath.Join(t.TempDir(), "obsidian-argv.log")
	t.Setenv("PATH", c.path)
	t.Setenv(stubEnvActive, "1")
	t.Setenv(stubEnvCase, c.dir)
	t.Setenv(stubEnvLog, logPath)
	assertObsidianResolves(t, c.stubBin)

	var out strings.Builder
	code, err := RunHealth(&out, HealthOptions{
		VaultDir: vaultDir,
		// What healthOptions() resolves when neither --vault nor $VAULT_NAME is
		// set, which is how every case was captured.
		VaultName: "knowledge",
		Verbose:   strings.Contains(readTrimmed(t, filepath.Join(c.dir, "args")), "--verbose"),
		// The argv depends on the OS, so each case pins one: linux, where every
		// case was captured, unless the case names another in its goos file.
		GOOS: goldenGOOS(t, c.dir),
	})
	if err != nil {
		t.Fatalf("RunHealth: %v", err)
	}

	argv, _ := os.ReadFile(logPath) // absent when obsidian was never invoked
	expected := filepath.Join(c.dir, "expected")
	assertArtefact(t, "exit", readFile(t, filepath.Join(expected, "exit")), strconv.Itoa(code)+"\n")
	assertArtefact(t, "stdout", readFile(t, filepath.Join(expected, "stdout")), normalizeGolden(out.String(), vaultDir))
	assertArtefact(t, "obsidian-argv", readFile(t, filepath.Join(expected, "obsidian-argv")), normalizeGolden(string(argv), vaultDir))
}

// installObsidianStub copies the running test binary to <dir>/obsidian, so the
// name RunHealth looks up on PATH runs TestMain's stub. A copy rather than a
// symlink: Windows needs the .exe and no symlink privilege.
func installObsidianStub(t *testing.T) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("locate the test binary: %v", err)
	}
	dir := t.TempDir()
	name := "obsidian"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	src, err := os.Open(self)
	if err != nil {
		t.Fatalf("open the test binary: %v", err)
	}
	defer func() { _ = src.Close() }()
	dst, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		t.Fatalf("create the stub: %v", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("copy the stub: %v", err)
	}
	if err := dst.Close(); err != nil {
		t.Fatalf("close the stub: %v", err)
	}
	return dir
}

// pathWithoutObsidian drops every PATH entry holding a real obsidian. A
// developer machine has one (~/.local/bin -> the AppImage); left in place, the
// absent-obsidian case would find it, and any case could reach the real GUI.
// Prepending the stub is not enough on its own, which is why lib.sh replaces
// PATH outright. Here the rest is kept, because the worktree cases run git from it.
func pathWithoutObsidian(path string) string {
	var keep []string
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			continue
		}
		if hasObsidian(dir) {
			continue
		}
		keep = append(keep, dir)
	}
	return strings.Join(keep, string(os.PathListSeparator))
}

func hasObsidian(dir string) bool {
	for _, name := range []string{"obsidian", "obsidian.exe", "obsidian.cmd", "obsidian.bat"} {
		if fileExists(filepath.Join(dir, name)) {
			return true
		}
	}
	return false
}

// assertObsidianResolves is lib.sh's isolation check: obsidian must resolve
// into the stub directory, or nowhere for the absent case. Isolation is a claim
// that needs its own assertion, not an assumption about PATH.
func assertObsidianResolves(t *testing.T, stubBin string) {
	t.Helper()
	got, err := exec.LookPath("obsidian")
	if !hasObsidian(stubBin) {
		if err == nil {
			t.Fatalf("PATH leak: obsidian still resolves to %s", got)
		}
		return
	}
	if err != nil || filepath.Dir(got) != stubBin {
		t.Fatalf("PATH leak: obsidian resolved to %q (err %v), not the stub in %s", got, err, stubBin)
	}
}

// seedGit builds the section-1 state: "clean" commits the fixture, "deleted"
// then removes one file from disk but not from HEAD (the 2026-05-13 incident).
func seedGit(t *testing.T, c goldenCase, vaultDir, mode string) {
	t.Helper()
	git := func(args ...string) {
		t.Helper()
		full := append([]string{"-C", vaultDir, "-c", "user.email=t@t", "-c", "user.name=t", "-c", "core.autocrlf=false"}, args...)
		if out, err := exec.Command(c.gitPath, full...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("add", "-A")
	git("commit", "-q", "-m", "fixture")
	if mode == "deleted" {
		rel := readTrimmed(t, filepath.Join(c.dir, "deleted-file"))
		if err := os.Remove(filepath.Join(vaultDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("remove %s: %v", rel, err)
		}
	}
}

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// normalizeGolden is lib.sh's _gvh_normalize: strip ANSI colour and replace the
// sandbox vault path with <VAULT>. The slash form is replaced too, for the
// paths a Windows run prints with forward slashes. On Windows the report joins
// paths with native separators and the goldens are POSIX; no golden carries a
// backslash, so mapping every one back is lossless.
func normalizeGolden(s, vaultDir string) string {
	s = ansiRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, vaultDir, "<VAULT>")
	s = strings.ReplaceAll(s, filepath.ToSlash(vaultDir), "<VAULT>")
	if runtime.GOOS == "windows" {
		s = strings.ReplaceAll(s, `\`, "/")
	}
	return s
}

func assertArtefact(t *testing.T, name, want, got string) {
	t.Helper()
	if want == got {
		return
	}
	t.Errorf("%s differs from the golden\n%s", name, lineDiff(want, got))
}

// lineDiff names the first differing line: enough to act on without a diff
// dependency, and the full texts follow for context.
func lineDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl != gl {
			return "first difference at line " + strconv.Itoa(i+1) +
				"\n  want: " + strconv.Quote(wl) + "\n  got:  " + strconv.Quote(gl) +
				"\n--- want ---\n" + want + "--- got ---\n" + got
		}
	}
	return "texts differ only in a trailing newline"
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			// lib.sh refuses these too: a link out of the fixture breaks isolation.
			t.Fatalf("fixture tree contains a symlink: %s", p)
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture %s: %v", src, err)
	}
}

// readTrimmed reads an optional one-line fixture file; absent is "".
// goldenGOOS is the platform a case's argv was captured for: its goos file, or
// linux when it has none. Never runtime.GOOS — the same corpus must pass on
// every CI leg.
func goldenGOOS(t *testing.T, caseDir string) string {
	t.Helper()
	if goos := readTrimmed(t, filepath.Join(caseDir, "goos")); goos != "" {
		return goos
	}
	return "linux"
}

func readTrimmed(t *testing.T, p string) string {
	t.Helper()
	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatalf("open %s: %v", p, err)
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return strings.TrimSpace(strings.Join(lines, " "))
}
