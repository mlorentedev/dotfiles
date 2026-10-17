package schedule

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func testJob() Job {
	return Job{
		Names:       Names{Launchd: "com.example.job", Systemd: "example-job", Task: "ExampleJob"},
		Description: "Example daily job",
		Program:     "/home/me/.local/bin/dotf",
		Args:        []string{"update"},
		Daily:       &Clock{Hour: 3, Minute: 5},
		Env:         []EnvVar{{Name: "PATH", Value: "/home/me/.local/bin:/usr/bin"}},
		Log:         "/home/me/Library/Logs/job.log",
	}
}

// exitStatus is a failed command's exit, as *exec.ExitError reports it.
type exitStatus int

func (e exitStatus) Error() string { return "exit status " + strconv.Itoa(int(e)) }
func (e exitStatus) ExitCode() int { return int(e) }

// fakeLaunchctl is launchd's view of one user: the labels it has loaded. fail
// makes the named subcommand fail with that error.
type fakeLaunchctl struct {
	loaded map[string]bool
	fail   map[string]error
	calls  []string
}

func (f *fakeLaunchctl) run(args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	if err := f.fail[args[0]]; err != nil {
		return "", err
	}
	switch args[0] {
	case "print":
		if f.loaded[args[1]] {
			return "loaded", nil
		}
		return "", exitStatus(113)
	case "bootstrap":
		f.loaded["gui/501/com.example.job"] = true
	case "bootout":
		delete(f.loaded, args[1])
	}
	return "", nil
}

func TestLaunchdPlist_RendersTheTimetableAndEnvironment(t *testing.T) {
	var doc struct {
		Dict struct {
			Keys []string `xml:"key"`
		} `xml:"dict"`
	}
	plist := LaunchdPlist(testJob())
	if err := xml.Unmarshal(plist, &doc); err != nil {
		t.Fatalf("the plist must be well-formed XML: %v", err)
	}
	want := "Label ProgramArguments EnvironmentVariables StartCalendarInterval StandardOutPath StandardErrorPath"
	if got := strings.Join(doc.Dict.Keys, " "); got != want {
		t.Errorf("keys = %q, want %q", got, want)
	}
	for _, s := range []string{"<key>Hour</key>\n\t\t<integer>3</integer>", "<key>Minute</key>\n\t\t<integer>5</integer>", "<key>PATH</key>"} {
		if !strings.Contains(string(plist), s) {
			t.Errorf("plist lacks %q", s)
		}
	}
	if strings.Contains(string(plist), "RunAtLoad") {
		t.Error("a timer job must not run at load: loading it would run it")
	}
}

func TestLaunchd_InstallThenCurrentThenRemove(t *testing.T) {
	lc := &fakeLaunchctl{loaded: map[string]bool{}}
	b := Launchd{Run: lc.run, UID: 501, Dir: filepath.Join(t.TempDir(), "LaunchAgents")}
	job := testJob()

	st, err := b.Inspect(job)
	if err != nil || st.Present || st.Current() || len(st.Pending) != 2 {
		t.Fatalf("fresh: %+v, %v; want absent with write+load pending", st, err)
	}
	if err := b.Install(job); err != nil {
		t.Fatal(err)
	}
	if st, _ = b.Inspect(job); !st.Current() || !st.Managed {
		t.Fatalf("after install: %+v, want current and managed", st)
	}
	lc.calls = nil
	if err := b.Install(job); err != nil || len(lc.calls) != 1 {
		t.Errorf("a second install must only inspect, ran %q (%v)", lc.calls, err)
	}

	job.Daily = &Clock{Hour: 4}
	if err := b.Install(job); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(lc.calls[1:], "; "); !strings.Contains(got, "bootout gui/501/com.example.job; bootstrap gui/501") {
		t.Errorf("a changed loaded agent is booted out, then bootstrapped: %q", got)
	}

	if err := b.Remove(job); err != nil {
		t.Fatal(err)
	}
	if st, _ = b.Inspect(job); st.Present {
		t.Errorf("after remove: %+v, want absent", st)
	}
}

// A print that fails for any reason but an unknown service is an error: read
// as "not loaded", opt-out would delete a running agent's plist and leave it
// running.
func TestLaunchd_AFailedPrintIsNotAnAbsentAgent(t *testing.T) {
	lc := &fakeLaunchctl{loaded: map[string]bool{}}
	b := Launchd{Run: lc.run, UID: 501, Dir: filepath.Join(t.TempDir(), "LaunchAgents")}
	job := testJob()
	if err := b.Install(job); err != nil {
		t.Fatal(err)
	}
	lc.fail = map[string]error{"print": exitStatus(5)}
	if _, err := b.Inspect(job); err == nil {
		t.Error("inspect must fail when print fails")
	}
	if err := b.Remove(job); err == nil {
		t.Error("remove must fail when print fails")
	}
	if _, err := os.Stat(filepath.Join(b.Dir, "com.example.job.plist")); err != nil {
		t.Errorf("the plist must survive a failed remove: %v", err)
	}
}

// A failed bootout leaves the old plist, so the next run still sees the agent
// stale and retries, instead of reading a current file over a stale load.
func TestLaunchd_AFailedBootoutLeavesTheChangePending(t *testing.T) {
	lc := &fakeLaunchctl{loaded: map[string]bool{}}
	b := Launchd{Run: lc.run, UID: 501, Dir: filepath.Join(t.TempDir(), "LaunchAgents")}
	job := testJob()
	if err := b.Install(job); err != nil {
		t.Fatal(err)
	}
	job.Daily = &Clock{Hour: 4}
	lc.fail = map[string]error{"bootout": errors.New("boom")}
	if err := b.Install(job); err == nil {
		t.Fatal("install must fail when bootout fails")
	}
	lc.fail = nil
	if st, err := b.Inspect(job); err != nil || st.Current() {
		t.Errorf("after a failed bootout: %+v, %v; want the change still pending", st, err)
	}
}

func TestFor_PicksTheBackendOfTheOS(t *testing.T) {
	b, err := For("darwin", Runners{Launchctl: (&fakeLaunchctl{}).run, UID: 501})
	if l, ok := b.(Launchd); err != nil || !ok || l.UID != 501 {
		t.Errorf("For(darwin) = %#v, %v", b, err)
	}
	for _, goos := range []string{"linux", "windows"} {
		if _, err := For(goos, Runners{Launchctl: (&fakeLaunchctl{}).run}); !errors.Is(err, ErrUnsupported) {
			t.Errorf("For(%s): want ErrUnsupported until its backend lands, got %v", goos, err)
		}
	}
	if _, err := For("darwin", Runners{}); !errors.Is(err, ErrUnsupported) {
		t.Errorf("an unwired runner is unsupported, got %v", err)
	}
}
