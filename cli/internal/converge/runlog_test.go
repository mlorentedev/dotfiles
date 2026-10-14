package converge

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

var runLogStart = time.Date(2026, 10, 14, 3, 55, 12, 0, time.UTC)

func openTestRunLog(t *testing.T, secrets map[string]string) (*RunLog, string) {
	t.Helper()
	state := t.TempDir()
	rl, err := OpenRunLog(state, runLogStart, secrets, "dotf converge (test)")
	if err != nil {
		t.Fatal(err)
	}
	return rl, state
}

func readRunLog(t *testing.T, rl *RunLog) string {
	t.Helper()
	if err := rl.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(rl.Path())
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// A transcript captures what uv, npm, winget and brew print, and any of them
// can print a credential. Nothing secret-shaped reaches the file.
func TestRunLog_RedactsKnownValuesAndTokenShapes(t *testing.T) {
	const known = "s3cr3t-value-from-the-registry"
	gh := "ghp_" + strings.Repeat("A1b2", 9)
	pat := "github_pat_" + strings.Repeat("x9", 20)
	sk := "sk-ant-" + strings.Repeat("Zq7", 10)
	ageKey := "AGE-SECRET-KEY-1" + strings.Repeat("QPZRY9X8", 6)
	rl, _ := openTestRunLog(t, map[string]string{"OPENROUTER_API_KEY": known})
	w := rl.Stream()
	for _, line := range []string{
		"npm notice token " + known,
		"git push https://" + gh + "@github.com/o/r",
		"pip: GET https://manu:hunter2pass@pypi.example.com/simple",
		"curl -H 'Authorization: Bearer abcdefghijklmnopqrstuvwxyz012345'",
		"export GH_TOKEN=" + pat,
		"anthropic " + sk,
		"aws AKIAIOSFODNN7EXAMPLE",
		"age " + ageKey,
	} {
		_, _ = fmt.Fprintln(w, line)
	}
	got := readRunLog(t, rl)
	for _, leak := range []string{known, gh, pat, sk, ageKey, "hunter2pass", "AKIAIOSFODNN7EXAMPLE", "abcdefghijklmnopqrstuvwxyz012345"} {
		if strings.Contains(got, leak) {
			t.Errorf("the log holds %q:\n%s", leak, got)
		}
	}
	if !strings.Contains(got, "[REDACTED:OPENROUTER_API_KEY]") {
		t.Errorf("a registry value should be named by its variable:\n%s", got)
	}
	// What surrounds a secret stays readable: the log is for debugging.
	for _, keep := range []string{"npm notice token", "@github.com/o/r", "https://manu:", "Authorization: Bearer", "pypi.example.com"} {
		if !strings.Contains(got, keep) {
			t.Errorf("redaction ate %q:\n%s", keep, got)
		}
	}
}

// A secret split across two writes is still caught: the log redacts whole
// lines, never the fragments a pipe happens to deliver.
func TestRunLog_ValueSplitAcrossWritesIsRedacted(t *testing.T) {
	const known = "split-across-two-writes-value"
	rl, _ := openTestRunLog(t, map[string]string{"BW_SESSION": known})
	w := rl.Stream()
	_, _ = io.WriteString(w, "session="+known[:10])
	_, _ = io.WriteString(w, known[10:]+" done\n")
	if got := readRunLog(t, rl); strings.Contains(got, known) || !strings.Contains(got, "session=[REDACTED:BW_SESSION] done") {
		t.Errorf("a split value leaked or the line was mangled:\n%s", got)
	}
}

// A run that dies mid-line still has its last words on disk.
func TestRunLog_CloseFlushesAPartialLine(t *testing.T) {
	rl, _ := openTestRunLog(t, nil)
	_, _ = io.WriteString(rl.Stream(), "Error: failed to install: eza")
	if got := readRunLog(t, rl); !strings.Contains(got, "Error: failed to install: eza\n") {
		t.Errorf("the unterminated last line was lost:\n%s", got)
	}
}

// A value shorter than six bytes is not masked: "1", "true" or "main" as a
// secret would shred every line of the transcript (the floor secrets run
// already uses).
func TestRunLog_ShortValuesAreNotMasked(t *testing.T) {
	rl, _ := openTestRunLog(t, map[string]string{"SOME_TOKEN": "main"})
	_, _ = fmt.Fprintln(rl.Stream(), "checkout main is clean")
	if got := readRunLog(t, rl); !strings.Contains(got, "checkout main is clean") {
		t.Errorf("a short value was masked:\n%s", got)
	}
}

// Colour codes are for the terminal; in a file they are noise that defeats grep.
func TestRunLog_StripsTerminalEscapes(t *testing.T) {
	rl, _ := openTestRunLog(t, nil)
	_, _ = fmt.Fprintln(rl.Stream(), "\x1b[1;32m✓\x1b[0m installed \x1b[2Kpoetry")
	if got := readRunLog(t, rl); strings.Contains(got, "\x1b") || !strings.Contains(got, "✓ installed poetry") {
		t.Errorf("escapes survived or the text was mangled: %q", got)
	}
}

// Each stream keeps its own partial line, so stdout and stderr interleave by
// whole lines and never splice into each other.
func TestRunLog_StreamsInterleaveByWholeLines(t *testing.T) {
	rl, _ := openTestRunLog(t, nil)
	out, errs := rl.Stream(), rl.Stream()
	_, _ = io.WriteString(out, "installing ")
	_, _ = io.WriteString(errs, "warning: slow mirror\n")
	_, _ = io.WriteString(out, "uv\n")
	got := readRunLog(t, rl)
	if !strings.Contains(got, "installing uv\n") || !strings.Contains(got, "warning: slow mirror\n") {
		t.Errorf("streams spliced into each other:\n%s", got)
	}
}

func TestRunLog_HeaderNamesTheRun(t *testing.T) {
	rl, _ := openTestRunLog(t, nil)
	got := readRunLog(t, rl)
	if !strings.HasPrefix(got, "# dotf converge (test)\n# started 2026-10-14T03:55:12Z\n") {
		t.Errorf("header missing or misplaced:\n%s", got)
	}
}

// The transcript and its report share a stamp, so one names the other.
func TestRunLog_ReportSitsBesideTheTranscript(t *testing.T) {
	rl, state := openTestRunLog(t, nil)
	if err := rl.WriteReport(Report{GOOS: "darwin"}, errors.New("tools: boom")); err != nil {
		t.Fatal(err)
	}
	_ = rl.Close()
	runs := filepath.Join(state, "converge", "runs")
	for _, name := range []string{"20261014T035512Z.log", "20261014T035512Z.json"} {
		if _, err := os.Stat(filepath.Join(runs, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(runs, "20261014T035512Z.json"))
	if !strings.Contains(string(raw), `"result": "failed"`) {
		t.Errorf("the run's report should record the failure:\n%s", raw)
	}
}

// Two runs in the same second never overwrite each other's transcript.
func TestRunLog_SameSecondGetsItsOwnFile(t *testing.T) {
	state := t.TempDir()
	a, err := OpenRunLog(state, runLogStart, nil, "a")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	b, err := OpenRunLog(state, runLogStart, nil, "b")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Close() }()
	if a.Path() == b.Path() {
		t.Fatalf("both runs write %s", a.Path())
	}
}

// Owner-only, set at creation: a transcript is a machine's whole install
// history. Windows has no POSIX mode bits; there the profile's ACL applies.
func TestRunLog_IsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no POSIX mode bits on Windows; StateDir sits under the user profile's ACL")
	}
	state := t.TempDir()
	runs := filepath.Join(state, "converge", "runs")
	// A directory an older dotf or a hand created loosely is tightened.
	if err := os.MkdirAll(runs, 0o755); err != nil {
		t.Fatal(err)
	}
	rl, err := OpenRunLog(state, runLogStart, nil, "perm")
	if err != nil {
		t.Fatal(err)
	}
	if err := rl.WriteReport(Report{GOOS: "linux"}, nil); err != nil {
		t.Fatal(err)
	}
	_ = rl.Close()
	if fi, _ := os.Stat(runs); fi.Mode().Perm() != 0o700 {
		t.Errorf("runs/ is %v, want 0700", fi.Mode().Perm())
	}
	for _, name := range []string{"20261014T035512Z.log", "20261014T035512Z.json"} {
		if fi, _ := os.Stat(filepath.Join(runs, name)); fi.Mode().Perm() != 0o600 {
			t.Errorf("%s is %v, want 0600", name, fi.Mode().Perm())
		}
	}
}

func TestPruneRuns_KeepsTheNewestRuns(t *testing.T) {
	dir := t.TempDir()
	for i := range 25 {
		stamp := runLogStart.Add(time.Duration(i) * time.Minute).Format(runStampLayout)
		for _, ext := range []string{".log", ".json"} {
			if err := os.WriteFile(filepath.Join(dir, stamp+ext), nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	// An orphan transcript (a run killed before its report) is a run too.
	orphan := runLogStart.Add(30*time.Minute).Format(runStampLayout) + ".log"
	_ = os.WriteFile(filepath.Join(dir, orphan), nil, 0o600)
	// Something that is not a run is never touched.
	_ = os.WriteFile(filepath.Join(dir, "notes.txt"), nil, 0o600)

	if err := PruneRuns(dir, 20); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	runs := map[string]bool{}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
		if stamp := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".json"), ".log"); stamp != e.Name() {
			runs[stamp] = true
		}
	}
	sort.Strings(names)
	if len(runs) != 20 {
		t.Errorf("kept %d runs, want 20: %v", len(runs), names)
	}
	oldest := runLogStart.Format(runStampLayout)
	if runs[oldest] {
		t.Errorf("the oldest run survived: %v", names)
	}
	if !runs[strings.TrimSuffix(orphan, ".log")] {
		t.Errorf("the newest run (an orphan transcript) was pruned: %v", names)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Errorf("prune removed a file that is not a run: %v", err)
	}
}

// The report beside the transcript is held to the same rule: a step's detail
// or the run's error can echo a remote URL with a token in it.
func TestRunLog_ReportIsRedactedToo(t *testing.T) {
	gh := "ghp_" + strings.Repeat("R3p0", 9)
	rl, state := openTestRunLog(t, nil)
	rep := Report{GOOS: "linux", Entries: []Entry{{Name: "checkout", Status: StatusFailed, Detail: "fetch https://x:" + gh + "@github.com/o/r failed"}}}
	runErr := errors.New("checkout: " + gh)
	red, redErr := rl.Redacted(rep, runErr)
	if err := rl.WriteReport(red, redErr); err != nil {
		t.Fatal(err)
	}
	_ = rl.Close()
	raw, _ := os.ReadFile(filepath.Join(state, "converge", "runs", "20261014T035512Z.json"))
	if strings.Contains(string(raw), gh) || !strings.Contains(string(raw), "[REDACTED]") {
		t.Errorf("the run's report leaks or was not masked:\n%s", raw)
	}
	if rep.Entries[0].Detail == red.Entries[0].Detail {
		t.Errorf("Redacted must copy, not mask the caller's report in place")
	}
}

// npm and uv are the children a converge captures most; their token shapes
// are masked by form.
func TestRunLog_RedactsPackageRegistryTokens(t *testing.T) {
	npm := "npm_" + strings.Repeat("a1B2c3", 6)
	pypi := "pypi-AgEIcHlwaS5vcmc" + strings.Repeat("Xy_9", 8)
	rl, _ := openTestRunLog(t, nil)
	_, _ = fmt.Fprintln(rl.Stream(), "//registry.npmjs.org/:_authToken="+npm+" uv publish --token "+pypi)
	got := readRunLog(t, rl)
	for _, leak := range []string{npm, pypi} {
		if strings.Contains(got, leak) {
			t.Errorf("the log holds %q:\n%s", leak, got)
		}
	}
}

// The header is a line like any other: argv can carry a token.
func TestRunLog_HeaderIsRedacted(t *testing.T) {
	gh := "ghp_" + strings.Repeat("H3ad", 9)
	rl, err := OpenRunLog(t.TempDir(), runLogStart, nil, "dotf converge --repo https://x:"+gh+"@github.com/o/r")
	if err != nil {
		t.Fatal(err)
	}
	if got := readRunLog(t, rl); strings.Contains(got, gh) {
		t.Errorf("the header leaks a token:\n%s", got)
	}
}
