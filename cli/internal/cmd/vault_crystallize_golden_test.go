package cmd

// `dotf vault crystallize` against the frozen golden corpus in
// tests/golden/crystallize/ (CI-009 / #1745). The goldens were captured from
// the deleted shell twin (see that directory's ORACLE) and are the contract the
// Go implementation keeps. They were replayed only by the bats suite
// tests/knowledge-crystallize-go-parity.bats, which this file replaces, and
// which skipped all 13 cases when `go build` failed (#1345).
//
// Driven through the command, not vault.CrystallizeOne: the [INFO] Date line,
// the refusal's exit status and the path resolution live in cmd/vault.go, and
// the goldens include them. Per case, the three artefacts lib.sh compares: the
// exit code, the report, and every resulting MEMORY.md.

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/memlink"
)

func TestVaultCrystallizeGoldenCorpus(t *testing.T) {
	casesDir, err := filepath.Abs(filepath.Join("..", "..", "..", "tests", "golden", "crystallize", "cases"))
	if err != nil {
		t.Fatalf("resolve the corpus: %v", err)
	}
	entries, err := os.ReadDir(casesDir)
	if err != nil {
		t.Fatalf("read the golden corpus: %v", err)
	}

	ran := 0
	for _, e := range entries {
		// `help` is excluded, as it was from the bats suite: the shell
		// hand-rolled its usage text and cobra generates its own, so the golden
		// records a framework difference, not a contract.
		if !e.IsDir() || e.Name() == "help" {
			continue
		}
		ran++
		caseDir := filepath.Join(casesDir, e.Name())
		t.Run(e.Name(), func(t *testing.T) { runCrystallizeCase(t, caseDir) })
	}
	if ran == 0 {
		t.Fatalf("no cases under %s: the corpus moved, and a loop over nothing passes", casesDir)
	}
}

func runCrystallizeCase(t *testing.T, caseDir string) {
	home := crystallizeSandbox(t)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows

	var projects []string
	switch {
	case isDirPath(filepath.Join(caseDir, "projects")):
		dirs, err := os.ReadDir(filepath.Join(caseDir, "projects"))
		if err != nil {
			t.Fatalf("read projects: %v", err)
		}
		for _, d := range dirs {
			if d.IsDir() {
				projects = append(projects, d.Name())
				placeProject(t, home, d.Name(), filepath.Join(caseDir, "projects", d.Name(), "input.md"))
			}
		}
	case isFilePath(filepath.Join(caseDir, "input.md")):
		projects = []string{"demo"}
		placeProject(t, home, "demo", filepath.Join(caseDir, "input.md"))
	default:
		// No fixture: the "no MEMORY.md found" path. The project dir must exist.
		projects = []string{"demo"}
		mustMkdir(t, filepath.Join(home, "Projects", "demo"))
	}

	// Orphans: memory dirs whose key decodes to nothing on disk. --all must
	// count them skipped, not processed.
	for _, orphan := range fixtureLines(t, filepath.Join(caseDir, "orphans")) {
		dir := filepath.Join(home, ".claude", "projects", orphan, "memory")
		mustMkdir(t, dir)
		mustWrite(t, filepath.Join(dir, "MEMORY.md"), "# Orphan memory\n")
	}

	args := []string{"vault", "crystallize"}
	if extra := fixtureLines(t, filepath.Join(caseDir, "args")); len(extra) > 0 {
		args = append(args, strings.Fields(strings.Join(extra, " "))...)
	} else {
		args = append(args, filepath.Join(home, "Projects", "demo"))
	}

	// `runs` > 1 pins idempotence: the last run's report and the final file
	// are what is compared, so a second pass that changes anything shows.
	runs := 1
	if r := fixtureLines(t, filepath.Join(caseDir, "runs")); len(r) > 0 {
		n, err := strconv.Atoi(r[0])
		if err != nil {
			t.Fatalf("runs fixture: %v", err)
		}
		runs = n
	}
	today := time.Now().Format("2006-01-02")
	var report string
	var code int
	for i := 0; i < runs; i++ {
		stdout, stderr, err := execute(t, args...)
		report, code = stdout+stderr, ExitCode(err)
	}

	var memory strings.Builder
	for _, name := range projects {
		mf := filepath.Join(memlink.ClaudeMemoryTarget(home, filepath.Join(home, "Projects", name)), "MEMORY.md")
		b, err := os.ReadFile(mf)
		if err != nil {
			continue // a project the run never stamped has no file, as in lib.sh
		}
		memory.WriteString("===== " + name + " =====\n")
		memory.WriteString(normalizeCrystallize(string(b), home, today))
	}

	expected := filepath.Join(caseDir, "expected")
	compareGolden(t, "exit", filepath.Join(expected, "exit"), strconv.Itoa(code)+"\n")
	compareGolden(t, "stdout", filepath.Join(expected, "stdout"), normalizeCrystallize(report, home, today))
	compareGolden(t, "memory.md", filepath.Join(expected, "memory.md"), memory.String())
}

// crystallizeSandbox is a fake HOME with no dash in its path. The decoder
// reverses the project-key encoding by mapping '-' back to a separator, so a
// dash in the sandbox would make the project undecodable and change which
// branch the case exercises. t.TempDir() embeds the subtest name, and case
// names have dashes, hence MkdirTemp. The check stays explicit: TMPDIR is the
// environment's, and an assumption about it is not isolation.
func crystallizeSandbox(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "gc")
	if err != nil {
		t.Fatalf("sandbox: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	home := filepath.Join(dir, "h")
	mustMkdir(t, home)
	if strings.Contains(home, "-") {
		t.Fatalf("sandbox path contains a dash, which the key decoder cannot reverse: %s", home)
	}
	return home
}

// placeProject installs fixture as project name's MEMORY.md. Fixtures are
// named input.md, never MEMORY.md: GUARD-001 forbids an agent-memory filename
// outside the vault, so the sandbox is where it gets that name.
func placeProject(t *testing.T, home, name, fixture string) {
	t.Helper()
	proj := filepath.Join(home, "Projects", name)
	mustMkdir(t, proj)
	memDir := memlink.ClaudeMemoryTarget(home, proj)
	mustMkdir(t, memDir)
	if b, err := os.ReadFile(fixture); err == nil {
		mustWrite(t, filepath.Join(memDir, "MEMORY.md"), string(b))
	}
}

// normalizeCrystallize is lib.sh's _gc_normalize: the home key, the home path
// and today's date are what legitimately vary. Fixtures carry old dates, so an
// unchanged line cannot normalise to <TODAY> and hide a missed write. On
// Windows the report prints native separators; the goldens are POSIX.
func normalizeCrystallize(s, home, today string) string {
	s = strings.ReplaceAll(s, memlink.ClaudeProjectKey(home), "<HOMEKEY>")
	s = strings.ReplaceAll(s, home, "<HOME>")
	if runtime.GOOS == "windows" {
		s = strings.ReplaceAll(s, `\`, "/")
	}
	return strings.ReplaceAll(s, today, "<TODAY>")
}

func compareGolden(t *testing.T, name, goldenPath, got string) {
	t.Helper()
	b, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read %s: %v", goldenPath, err)
	}
	if want := string(b); want != got {
		t.Errorf("%s differs from the golden\n--- want ---\n%s--- got ---\n%s", name, want, got)
	}
}

// fixtureLines reads an optional fixture file's non-empty lines; absent is nil.
func fixtureLines(t *testing.T, p string) []string {
	t.Helper()
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

func mustWrite(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

func isDirPath(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func isFilePath(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
