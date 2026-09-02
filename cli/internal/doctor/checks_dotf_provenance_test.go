package doctor

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// These cases pin the distinction the whole check exists for: a report that is
// SILENT about provenance is indistinguishable from one that established it.
// Measured 2026-09-01 (#1158), the deployed binary matched the versions.conf pin
// exactly and was two feature merges stale, so every case below is written
// against fakes that reproduce that arrangement rather than a synthetic one.

// provenanceSys builds a System whose two command seams answer for the two
// distinct subjects this check interrogates: the dotf binary (CommandOutput) and
// git inside the checkout (CommandOutputDir). Keying git replies on the
// SUBCOMMAND rather than the full argv keeps the fixtures readable while still
// letting a case fail one specific git call.
func provenanceSys(t *testing.T, dotfOut string, dotfErr error, git map[string]string, gitErr map[string]error) *System {
	t.Helper()
	return &System{
		LookPath: func(name string) (string, error) {
			if name == "dotf" {
				return "/home/u/.local/bin/dotf", nil
			}
			return "", fmt.Errorf("not found: %s", name)
		},
		CommandOutput: func(_ string, _ ...string) (string, error) {
			return dotfOut, dotfErr
		},
		CommandOutputDir: func(_, name string, args ...string) (string, error) {
			if name != "git" || len(args) == 0 {
				return "", fmt.Errorf("unexpected command %s", name)
			}
			if err, ok := gitErr[args[0]]; ok {
				return "", err
			}
			if out, ok := git[args[0]]; ok {
				return out, nil
			}
			return "", fmt.Errorf("no fixture for git %s", args[0])
		},
	}
}

const (
	headSHA  = "6e0d180ae57c24148fbe3e74af7c7f75d86c40f1"
	staleSHA = "11a68b1c0ffee0000000000000000000000000aa"
)

func TestCheckDotfProvenance(t *testing.T) {
	for _, tc := range []struct {
		name       string
		repoDir    string
		dotfOut    string
		dotfErr    error
		git        map[string]string
		gitErr     map[string]error
		wantSubstr string
		wantWarn   bool
		wantPass   bool
	}{
		{
			// The case that motivated the ticket. Version-equality says "fine".
			name:    "a binary behind HEAD on cli/ warns and names the distance",
			repoDir: "/repo",
			dotfOut: staleSHA + "\n",
			git: map[string]string{
				"rev-parse":  headSHA,
				"cat-file":   "",
				"merge-base": "",
				"rev-list":   "7\n",
			},
			wantSubstr: "7 cli/ commit(s) behind HEAD",
			wantWarn:   true,
		},
		{
			name:    "a binary current with HEAD for cli/ passes",
			repoDir: "/repo",
			dotfOut: headSHA + "\n",
			git: map[string]string{
				"rev-parse":  headSHA,
				"cat-file":   "",
				"merge-base": "",
				"rev-list":   "0\n",
			},
			wantSubstr: "current with HEAD for cli/",
			wantPass:   true,
		},
		{
			// Docs/spec/harness commits move HEAD without changing the binary's
			// behaviour. Reporting those would make the check noisy enough to
			// ignore, which is how a guard dies.
			name:    "commits that miss cli/ do not count as staleness",
			repoDir: "/repo",
			dotfOut: staleSHA + "\n",
			git: map[string]string{
				"rev-parse":  headSHA,
				"cat-file":   "",
				"merge-base": "",
				"rev-list":   "0\n",
			},
			wantSubstr: "current with HEAD for cli/",
			wantPass:   true,
		},
		{
			// The ticket's guard clause, verbatim: "not an ancestor of HEAD".
			// Its remedy is the OPPOSITE of the behind case, so it gets its own
			// message rather than being folded into a commit count.
			name:    "a binary built off a branch is named as such, not as behind",
			repoDir: "/repo",
			dotfOut: staleSHA + "\n",
			git: map[string]string{
				"rev-parse": headSHA,
				"cat-file":  "",
			},
			gitErr:     map[string]error{"merge-base": fmt.Errorf("exit status 1")},
			wantSubstr: "is not an ancestor of HEAD",
			wantWarn:   true,
		},
		{
			name:    "a commit this checkout does not contain is distinguished from staleness",
			repoDir: "/repo",
			dotfOut: staleSHA + "\n",
			git:     map[string]string{"rev-parse": headSHA},
			gitErr:  map[string]error{"cat-file": fmt.Errorf("exit status 1")},

			wantSubstr: "which this checkout does not contain",
			wantWarn:   true,
		},
		{
			// The state measured on the box: the installed binary predates the
			// flag. The probe FAILING is the answer, not a broken probe.
			name:       "a binary predating the --commit flag reports unestablished provenance",
			repoDir:    "/repo",
			dotfErr:    fmt.Errorf("unknown flag: --commit"),
			git:        map[string]string{"rev-parse": headSHA},
			wantSubstr: "predates the build stamp",
			wantWarn:   true,
		},
		{
			// `go build ./cmd/dotf` leaves main.commit empty. Deliberate on a dev
			// box, and the installers refuse to replace it — so it is reported
			// rather than treated as breakage.
			name:       "a source build with an empty stamp warns without scolding",
			repoDir:    "/repo",
			dotfOut:    "\n",
			git:        map[string]string{"rev-parse": headSHA},
			wantSubstr: "source build carrying no commit stamp",
			wantWarn:   true,
		},
		{
			// C15: a check that cannot answer says so. This is most machines.
			name:       "outside a checkout the check skips rather than passing",
			repoDir:    "",
			wantSubstr: "no HEAD to compare",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sys := provenanceSys(t, tc.dotfOut, tc.dotfErr, tc.git, tc.gitErr)

			var buf bytes.Buffer
			rep := NewReport(&buf, true)
			checkDotfProvenance(sys, &Config{RepoDir: tc.repoDir}, rep)

			text := buf.String()
			if !strings.Contains(text, tc.wantSubstr) {
				t.Fatalf("report does not contain %q:\n%s", tc.wantSubstr, text)
			}
			// Severity is load-bearing. Running a released binary from inside a
			// checkout is legitimate and common; failing the health command over
			// a normal state trains the reader to ignore the line, which is the
			// exact failure this check was built to end.
			if rep.Failures() != 0 {
				t.Errorf("dotf provenance must never FAIL, only WARN:\n%s", text)
			}
			if tc.wantWarn && rep.Warnings() == 0 {
				t.Errorf("expected a WARN, got none:\n%s", text)
			}
			if tc.wantPass && rep.Warnings() != 0 {
				t.Errorf("expected a clean PASS, got a warning:\n%s", text)
			}
		})
	}
}

// A dotf that is not installed is a legitimate state (a fresh box mid-bootstrap)
// and must not be reported as drift.
func TestCheckDotfProvenanceSkipsWhenDotfIsAbsent(t *testing.T) {
	sys := provenanceSys(t, "", nil, nil, nil)
	sys.LookPath = func(string) (string, error) { return "", fmt.Errorf("not found") }

	var buf bytes.Buffer
	rep := NewReport(&buf, true)
	checkDotfProvenance(sys, &Config{RepoDir: "/repo"}, rep)

	if !strings.Contains(buf.String(), "not on PATH") {
		t.Fatalf("expected a skip naming the absent binary:\n%s", buf.String())
	}
	if rep.Failures() != 0 || rep.Warnings() != 0 {
		t.Errorf("an absent dotf is not drift:\n%s", buf.String())
	}
}

// The cli/ pathspec must be root-relative. A bare `cli` is resolved against
// git's CWD, so it means `cli/cli` from inside cli/ and silently counts 0 —
// measured at 4-from-root / 0-from-cli while verifying this check. A guard that
// answers 0 because it looked in the wrong place is the failure #1158 is about,
// reproduced inside the fix for it.
func TestCheckDotfProvenanceUsesRootRelativePathspec(t *testing.T) {
	var revList string
	sys := provenanceSys(t, staleSHA+"\n", nil, map[string]string{
		"rev-parse": headSHA, "cat-file": "", "merge-base": "", "rev-list": "4\n",
	}, nil)
	inner := sys.CommandOutputDir
	sys.CommandOutputDir = func(dir, name string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "rev-list" {
			revList = strings.Join(args, " ")
		}
		return inner(dir, name, args...)
	}

	var buf bytes.Buffer
	checkDotfProvenance(sys, &Config{RepoDir: "/repo"}, NewReport(&buf, true))

	if !strings.Contains(revList, ":(top)cli") {
		t.Fatalf("rev-list must scope with the root-relative pathspec, got: %q", revList)
	}
}

// The check abbreviates for the message but must hand git the FULL hash: an
// abbreviated one is ambiguous in principle and does not resolve at all in the
// shallow clone CI checks out by default.
func TestCheckDotfProvenancePassesFullSHAToGit(t *testing.T) {
	var seen []string
	sys := provenanceSys(t, staleSHA+"\n", nil, map[string]string{
		"rev-parse": headSHA, "cat-file": "", "merge-base": "", "rev-list": "3\n",
	}, nil)
	inner := sys.CommandOutputDir
	sys.CommandOutputDir = func(dir, name string, args ...string) (string, error) {
		seen = append(seen, strings.Join(args, " "))
		return inner(dir, name, args...)
	}

	var buf bytes.Buffer
	checkDotfProvenance(sys, &Config{RepoDir: "/repo"}, NewReport(&buf, true))

	for _, call := range seen {
		if strings.Contains(call, staleSHA[:12]) && !strings.Contains(call, staleSHA) {
			t.Fatalf("git received an abbreviated SHA: %q", call)
		}
	}
	if !strings.Contains(buf.String(), staleSHA[:12]) {
		t.Errorf("the human-facing message should abbreviate:\n%s", buf.String())
	}
}
