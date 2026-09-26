package pi

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func writeJSON(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func manifestRepo(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	writeJSON(t, filepath.Join(root, ManifestFile), body)
	return root
}

// AC2: a manifest that cannot be read must never read as "nothing declared",
// because an empty want-list would remove every live package.
func TestLoadManifestRefusesWhatItCannotRead(t *testing.T) {
	cases := map[string]string{
		"not json":        `{"packages": [`,
		"no packages":     `{"version": 1, "packages": []}`,
		"an empty source": `{"packages": [{"source": "", "why": "x"}]}`,
		"a duplicate":     `{"packages": [{"source": "npm:a@1", "why": "x"}, {"source": "npm:a@2", "why": "y"}]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadManifest(manifestRepo(t, body)); err == nil {
				t.Fatal("want an error")
			}
		})
	}
	if _, err := LoadManifest(t.TempDir()); err == nil {
		t.Fatal("an absent manifest must be an error")
	}
}

func TestLoadManifestReadsTheShippedOne(t *testing.T) {
	m, err := LoadManifest(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Packages) == 0 {
		t.Fatal("the shipped manifest declares packages")
	}
}

// Both entry forms pi writes: a plain string, and upstream's object form.
func TestLiveSourcesReadsBothEntryForms(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	writeJSON(t, path, `{"packages": ["npm:a@1", {"source": "npm:@s/b@2", "extensions": ["x"]}]}`)
	got, err := LiveSources(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"npm:a@1", "npm:@s/b@2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got, err := LiveSources(filepath.Join(t.TempDir(), "absent.json")); err != nil || len(got) != 0 {
		t.Fatalf("an absent settings file is an empty live set, got %v %v", got, err)
	}
	writeJSON(t, path, `{"packages": [`)
	if _, err := LiveSources(path); err == nil {
		t.Fatal("unreadable live settings must be an error: removals planned against them would be guesses")
	}
}

func TestIdentityIgnoresTheVersion(t *testing.T) {
	cases := map[string]string{
		"npm:pi-effort@0.0.8":         "npm:pi-effort",
		"npm:@ayulab/pi-rewind@0.4.6": "npm:@ayulab/pi-rewind",
		"npm:@scope/unpinned":         "npm:@scope/unpinned",
		"git:github.com/o/r@v1":       "git:github.com/o/r",
		"./local/extension":           "./local/extension",
	}
	for in, want := range cases {
		if got := Identity(in); got != want {
			t.Errorf("Identity(%q) = %q, want %q", in, got, want)
		}
	}
}

func manifestOf(sources ...string) Manifest {
	var m Manifest
	for _, s := range sources {
		m.Packages = append(m.Packages, Package{Source: s, Why: "x"})
	}
	return m
}

// AC1: an undeclared package is removed, a missing one installed, and a
// version bump is an install only, because pi install replaces the entry of
// the same package (measured on pi 0.87.1).
func TestNewPlan(t *testing.T) {
	m := manifestOf("npm:keep@1", "npm:bump@2", "npm:new@1")
	p := NewPlan(m, []string{"npm:keep@1", "npm:bump@1", "npm:gone@3"})
	if !reflect.DeepEqual(p.Remove, []string{"npm:gone@3"}) {
		t.Errorf("remove = %v", p.Remove)
	}
	if !reflect.DeepEqual(p.Install, []string{"npm:bump@2", "npm:new@1"}) {
		t.Errorf("install = %v", p.Install)
	}
	if p.Empty() {
		t.Error("a plan with work is not empty")
	}
	if !NewPlan(m, []string{"npm:keep@1", "npm:bump@2", "npm:new@1"}).Empty() {
		t.Error("live equal to the manifest is an empty plan")
	}
}

// fakePi edits a real settings.json the way pi does, so a second plan reads
// what the first apply left behind.
type fakePi struct {
	t        *testing.T
	settings string
	calls    []string
	fail     map[string]bool
}

func (f *fakePi) run(_ string, args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	verb, src := args[0], args[1]
	if f.fail[src] {
		return "npm ERR! boom", errors.New("exit status 1")
	}
	live, err := LiveSources(f.settings)
	if err != nil {
		f.t.Fatal(err)
	}
	var next []string
	for _, s := range live {
		if Identity(s) != Identity(src) {
			next = append(next, s)
		}
	}
	if verb == "install" {
		next = append(next, src)
	}
	body, _ := json.Marshal(map[string]any{"packages": next})
	writeJSON(f.t, f.settings, string(body))
	return "added 1 package\nfound 0 vulnerabilities", nil
}

func agentDirWith(t *testing.T, live string) (agentDir, settings string) {
	t.Helper()
	agentDir = t.TempDir()
	settings = filepath.Join(agentDir, "settings.json")
	writeJSON(t, settings, live)
	return agentDir, settings
}

func TestApplyConvergesAndASecondRunCallsNothing(t *testing.T) {
	_, settings := agentDirWith(t, `{"packages": ["npm:keep@1", "npm:bump@1", "npm:gone@3"]}`)
	m := manifestOf("npm:keep@1", "npm:bump@2", "npm:new@1")
	f := &fakePi{t: t, settings: settings}
	var log strings.Builder
	opt := Options{Log: &log, SlowAfter: time.Hour}

	live, _ := LiveSources(settings)
	res := Apply(NewPlan(m, live), opt, f.run)
	if res.Failed != 0 || res.Changed() != 3 {
		t.Fatalf("want 3 changes and no failure, got %+v\n%s", res, log.String())
	}
	if f.calls[0] != "remove npm:gone@3" {
		t.Errorf("removals come first, got %v", f.calls)
	}

	live, _ = LiveSources(settings)
	again := NewPlan(m, live)
	if !again.Empty() {
		t.Fatalf("the second plan must be empty, got %+v", again)
	}
	before := len(f.calls)
	if res := Apply(again, opt, f.run); res.Changed() != 0 || len(f.calls) != before {
		t.Fatalf("a second run must make no call, got %+v and %v", res, f.calls[before:])
	}
}

// AC3: every call's elapsed time is logged, and a failed or slow one prints
// its captured output inside a fence, so empty output is itself legible.
func TestApplyLogsTimeAndFencesTheOutputOfAFailure(t *testing.T) {
	_, settings := agentDirWith(t, `{"packages": []}`)
	f := &fakePi{t: t, settings: settings, fail: map[string]bool{"npm:bad@1": true}}
	var log strings.Builder
	res := Apply(NewPlan(manifestOf("npm:ok@1", "npm:bad@1"), nil), Options{Log: &log, SlowAfter: time.Hour}, f.run)
	out := log.String()
	if res.Failed != 1 || res.Changed() != 1 {
		t.Fatalf("want one failure and one install, got %+v", res)
	}
	for _, want := range []string{"npm:ok@1 installed in", "pi install npm:bad@1 failed after", "--- pi install npm:bad@1 (exit 1,", "npm ERR! boom", "--- end pi install npm:bad@1 ---"} {
		if !strings.Contains(out, want) {
			t.Errorf("log lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "--- pi install npm:ok@1") {
		t.Errorf("a fast success prints no output:\n%s", out)
	}
}

// The threshold gates verbosity only: a slow success still installs.
func TestApplySlowSuccessIsFencedAndStillCounted(t *testing.T) {
	_, settings := agentDirWith(t, `{"packages": []}`)
	f := &fakePi{t: t, settings: settings}
	var log strings.Builder
	res := Apply(NewPlan(manifestOf("npm:slow@1"), nil), Options{Log: &log, SlowAfter: -1}, f.run)
	if res.Changed() != 1 || !strings.Contains(log.String(), "--- pi install npm:slow@1") {
		t.Fatalf("want the install counted and its output fenced, got %+v\n%s", res, log.String())
	}
}
