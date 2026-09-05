package vault

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// frozen is the clock every RunMaintain test injects, so the two `date` stamps
// are assertable rather than merely present.
var frozen = time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

// recorder captures what the notifier was asked to send. A test that let the
// real notifier run would fire a desktop notification on the developer's
// machine — the same class of leak tests/golden/vault-health guards against by
// replacing PATH rather than extending it.
type recorder struct {
	calls []string
}

func (r *recorder) notify(u Urgency, title, body string) {
	r.calls = append(r.calls, fmt.Sprintf("%s|%s|%s", u, title, body))
}

// baseOpts builds an options set that touches no vault, no real log location
// and no desktop bus.
func baseOpts(t *testing.T, crystallizeOut string, healthOut string, healthCode int) (MaintainOptions, *recorder) {
	t.Helper()
	rec := &recorder{}
	return MaintainOptions{
		Home:    t.TempDir(),
		Today:   "2026-09-05",
		LogFile: filepath.Join(t.TempDir(), "vault-maintenance", "latest.log"),
		Now:     func() time.Time { return frozen },
		Notify:  rec.notify,
		crystallizeStep: func(w io.Writer) error {
			_, _ = io.WriteString(w, crystallizeOut)
			return nil
		},
		healthStep: func(w io.Writer) (int, error) {
			_, _ = io.WriteString(w, healthOut)
			return healthCode, nil
		},
	}, rec
}

func TestCountIssues(t *testing.T) {
	tests := []struct {
		name string
		log  string
		want int
	}{
		{"empty", "", 0},
		{"no keywords", "everything is fine\nnothing to see", 0},
		{"one warning", "WARNING: stale file", 1},
		{"case insensitive", "warning\nWARNING\nWaRnInG", 3},
		// grep -c counts LINES, not matches: three keywords on one line is one.
		{"counts lines not matches", "warning fail action stale", 1},
		{"one per line", "warning\nfail\naction\nstale", 4},
		// Faithful to the oracle's substring matching, which over-counts.
		// Reproduced, not fixed — see maintain.go's comment on issueRE.
		{"substring matches over-count", "the build failed\nstalemate reached", 2},
		{"blank lines ignored", "warning\n\n\nfail", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CountIssues(tt.log); got != tt.want {
				t.Errorf("CountIssues(%q) = %d, want %d", tt.log, got, tt.want)
			}
		})
	}
}

func TestNotificationFor(t *testing.T) {
	tests := []struct {
		name        string
		issues      int
		wantUrgency Urgency
		wantBody    string
	}{
		{"clean", 0, UrgencyLow, "All clean. No action needed."},
		{"boundary at one", 1, UrgencyNormal, "1 potential issues. Run /insights on active projects."},
		{"many", 7, UrgencyNormal, "7 potential issues. Run /insights on active projects."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, title, body := notificationFor(tt.issues)
			if u != tt.wantUrgency {
				t.Errorf("urgency = %q, want %q", u, tt.wantUrgency)
			}
			if title != "Vault Maintenance" {
				t.Errorf("title = %q, want %q", title, "Vault Maintenance")
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestLogFileFor(t *testing.T) {
	tests := []struct {
		name         string
		goos         string
		home         string
		localAppData string
		want         string
	}{
		{
			name: "linux uses XDG data dir under home",
			goos: "linux", home: "/home/u",
			want: filepath.Join("/home/u", ".local", "share", "vault-maintenance", "latest.log"),
		},
		{
			// os.UserCacheDir() would give ~/.cache here — a DIFFERENT directory
			// from the one the twin has been writing to for the log's whole life.
			name: "linux is not the cache dir",
			goos: "linux", home: "/home/u", localAppData: `C:\Users\u\AppData\Local`,
			want: filepath.Join("/home/u", ".local", "share", "vault-maintenance", "latest.log"),
		},
		{
			name: "windows prefers LOCALAPPDATA",
			goos: "windows", home: `C:\Users\u`, localAppData: `C:\Users\u\AppData\Local`,
			want: filepath.Join(`C:\Users\u\AppData\Local`, "vault-maintenance", "latest.log"),
		},
		{
			name: "windows falls back under home when LOCALAPPDATA is unset",
			goos: "windows", home: `C:\Users\u`,
			want: filepath.Join(`C:\Users\u`, "AppData", "Local", "vault-maintenance", "latest.log"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logFileFor(tt.goos, tt.home, tt.localAppData); got != tt.want {
				t.Errorf("logFileFor(%q, %q, %q) = %q, want %q",
					tt.goos, tt.home, tt.localAppData, got, tt.want)
			}
		})
	}
}

func TestRunMaintainWritesLogWithBothSections(t *testing.T) {
	opts, _ := baseOpts(t, "crystallize ran\n", "health ran\n", 0)
	var out strings.Builder

	if err := RunMaintain(&out, opts); err != nil {
		t.Fatalf("RunMaintain: %v", err)
	}

	raw, err := os.ReadFile(opts.LogFile)
	if err != nil {
		t.Fatalf("log not written: %v", err)
	}
	log := string(raw)

	stamp := frozen.Format(time.UnixDate)
	for _, want := range []string{
		"=== Vault Maintenance: " + stamp + " ===",
		"--- dotf vault crystallize --all ---",
		"crystallize ran",
		"--- vault-health ---",
		"health ran",
		"=== Done: " + stamp + " ===",
	} {
		if !strings.Contains(log, want) {
			t.Errorf("log missing %q\n--- log ---\n%s", want, log)
		}
	}

	// Ordering is the point of a composition: health must follow crystallize,
	// and the footer must follow both. Contains() alone would pass on any order.
	iCryst := strings.Index(log, "--- dotf vault crystallize --all ---")
	iHealth := strings.Index(log, "--- vault-health ---")
	iDone := strings.Index(log, "=== Done:")
	if iCryst >= iHealth || iHealth >= iDone {
		t.Errorf("sections out of order: crystallize=%d health=%d done=%d", iCryst, iHealth, iDone)
	}

	if !strings.Contains(out.String(), "Log written to "+opts.LogFile) {
		t.Errorf("stdout missing the log path, got %q", out.String())
	}
}

// The property tests/vault-maintenance-weekly.bats:171 asserts of the shell:
// the issue count picks a notification urgency and NOTHING else. It must never
// become a failure, however many issues the log carries.
func TestRunMaintainIssueCountNeverFails(t *testing.T) {
	noisy := "WARNING: stale\nfail\naction needed\nwarning again\n"
	opts, rec := baseOpts(t, noisy, "", 0)
	var out strings.Builder

	if err := RunMaintain(&out, opts); err != nil {
		t.Fatalf("a log full of issue keywords must not fail the run, got %v", err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("want exactly one notification, got %d", len(rec.calls))
	}
	if !strings.HasPrefix(rec.calls[0], string(UrgencyNormal)+"|") {
		t.Errorf("issues present must raise urgency to %q, got %q", UrgencyNormal, rec.calls[0])
	}
}

func TestRunMaintainCleanLogNotifiesLow(t *testing.T) {
	// No issue keywords anywhere — including in the frame this function writes
	// itself, which is why the section headers deliberately avoid them.
	opts, rec := baseOpts(t, "all good\n", "all good\n", 0)

	if err := RunMaintain(io.Discard, opts); err != nil {
		t.Fatalf("RunMaintain: %v", err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("want exactly one notification, got %d", len(rec.calls))
	}
	if !strings.HasPrefix(rec.calls[0], string(UrgencyLow)+"|") {
		t.Errorf("a clean log must notify at %q, got %q", UrgencyLow, rec.calls[0])
	}
}

// A FINDING IS NOT A FAILURE. Health's 1 and 2 are report verdicts, not run
// failures: they reach stdout, never the error. See RunMaintain's doc comment.
func TestRunMaintainHealthVerdictIsReportedNotFailed(t *testing.T) {
	tests := []struct {
		name       string
		healthCode int
		wantLine   string
	}{
		{"pass is silent", 0, ""},
		{"failed checks", 1, "Vault health: FAILED — one or more checks"},
		{"gui unreachable", 2, "Vault health: SKIPPED — the Obsidian GUI was unreachable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, _ := baseOpts(t, "", "", tt.healthCode)
			var out strings.Builder

			if err := RunMaintain(&out, opts); err != nil {
				t.Fatalf("health code %d must not fail the run, got %v", tt.healthCode, err)
			}
			got := out.String()
			if tt.wantLine == "" {
				if strings.Contains(got, "Vault health:") {
					t.Errorf("a passing health check must print no verdict, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantLine) {
				t.Errorf("stdout = %q, want it to contain %q", got, tt.wantLine)
			}
		})
	}
}

// Both steps are best-effort, exactly as the twin's `|| true`: a crystallize
// that fails must not stop health from running, nor the log from being written.
func TestRunMaintainCrystallizeFailureDoesNotStopHealth(t *testing.T) {
	opts, _ := baseOpts(t, "", "health still ran\n", 0)
	opts.crystallizeStep = func(io.Writer) error { return errors.New("refused: yaml-wrapped") }
	var out strings.Builder

	if err := RunMaintain(&out, opts); err != nil {
		t.Fatalf("a failing crystallize must not fail the run, got %v", err)
	}

	raw, err := os.ReadFile(opts.LogFile)
	if err != nil {
		t.Fatalf("log not written: %v", err)
	}
	log := string(raw)
	if !strings.Contains(log, "refused: yaml-wrapped") {
		t.Errorf("the crystallize error belongs IN the log, got:\n%s", log)
	}
	if !strings.Contains(log, "health still ran") {
		t.Errorf("health must run after a failed crystallize, got:\n%s", log)
	}
}

// The log is the artefact. Failing to write it is the one genuine failure of
// the run, and the only thing that errors.
func TestRunMaintainUnwritableLogIsAnError(t *testing.T) {
	opts, _ := baseOpts(t, "", "", 0)
	// A regular file where the log's parent directory must go: MkdirAll fails.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	opts.LogFile = filepath.Join(blocker, "vault-maintenance", "latest.log")

	if err := RunMaintain(io.Discard, opts); err == nil {
		t.Fatal("an unwritable log location must return an error")
	}
}

// A nil Notifier is the headless case — no desktop bus, nothing to notify. It
// must be a no-op, not a nil-func panic, because that is how this runs under
// cron on a server.
func TestRunMaintainNilNotifierIsSafe(t *testing.T) {
	opts, _ := baseOpts(t, "warning everywhere\n", "", 0)
	opts.Notify = nil

	if err := RunMaintain(io.Discard, opts); err != nil {
		t.Fatalf("a nil Notifier must be a no-op, got %v", err)
	}
}
