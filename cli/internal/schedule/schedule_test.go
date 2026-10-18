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

// fakeSystemctl is a user manager: which timers are enabled and active.
type fakeSystemctl struct {
	on    map[string]bool
	fail  bool // the manager cannot be reached
	calls []string
}

func (f *fakeSystemctl) run(args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	switch args[0] {
	case "show":
		if f.fail {
			return "", exitStatus(1)
		}
		if f.on[args[3]] {
			return "UnitFileState=enabled\nActiveState=active\n", nil
		}
		return "UnitFileState=\nActiveState=inactive\n", nil
	case "enable":
		f.on[args[2]] = true
	case "disable":
		delete(f.on, args[2])
	}
	return "", nil
}

func TestSystemdUnits_RenderTheJob(t *testing.T) {
	svc, tmr := SystemdUnits(testJob())
	for _, s := range []string{Marker + "\n", "Type=oneshot\n", "ExecStart=/home/me/.local/bin/dotf update\n"} {
		if !strings.Contains(string(svc), s) {
			t.Errorf("service lacks %q", s)
		}
	}
	for _, s := range []string{"OnCalendar=*-*-* 03:05:00\n", "Persistent=true\n", "RandomizedDelaySec=", "WantedBy=timers.target\n"} {
		if !strings.Contains(string(tmr), s) {
			t.Errorf("timer lacks %q", s)
		}
	}
	if !strings.Contains(string(svc), `Environment="PATH=/home/me/.local/bin:/usr/bin"`+"\n") {
		t.Errorf("the job's environment must reach the service:\n%s", svc)
	}
	job := testJob()
	job.Program = "/home/a b/.local/bin/dotf"
	if svc, _ = SystemdUnits(job); !strings.Contains(string(svc), `ExecStart="/home/a b/.local/bin/dotf" update`) {
		t.Errorf("a path with a space must be quoted:\n%s", svc)
	}
}

// systemd expands %-specifiers in ExecStart and Description too, not only in
// Environment= (OPS-053 review round 1, F3): a home holding `%h` ran another
// path. Each `%` is doubled wherever the job's text reaches a unit.
func TestSystemdUnits_DoublesPercentEverywhereTheJobReaches(t *testing.T) {
	job := testJob()
	job.Program = "/home/m%h/.local/bin/dotf"
	job.Args = []string{"update", "--note=100%"}
	job.Description = "dotf update at 100%"
	svc, tmr := SystemdUnits(job)
	for _, want := range []string{"ExecStart=/home/m%%h/.local/bin/dotf update --note=100%%\n", "Description=dotf update at 100%%\n"} {
		if !strings.Contains(string(svc), want) {
			t.Errorf("service lacks %q:\n%s", want, svc)
		}
	}
	if !strings.Contains(string(tmr), "Description=dotf update at 100%%\n") {
		t.Errorf("timer lacks the escaped description:\n%s", tmr)
	}
}

// A manager dotf cannot reach is an error, not a disabled timer: read as one,
// opt-out would delete the units of a running timer.
func TestSystemd_AnUnreachableManagerIsNotAnAbsentTimer(t *testing.T) {
	sc := &fakeSystemctl{on: map[string]bool{}}
	b := Systemd{Run: sc.run, Dir: filepath.Join(t.TempDir(), "systemd", "user")}
	if err := b.Install(testJob()); err != nil {
		t.Fatal(err)
	}
	sc.fail = true
	if _, err := b.Inspect(testJob()); err == nil {
		t.Error("inspect must fail when the manager cannot be reached")
	}
	if err := b.Remove(testJob()); err == nil {
		t.Error("remove must fail when the manager cannot be reached")
	}
}

func TestSystemd_InstallThenCurrentThenRemove(t *testing.T) {
	sc := &fakeSystemctl{on: map[string]bool{}}
	b := Systemd{Run: sc.run, Dir: filepath.Join(t.TempDir(), "systemd", "user")}
	job := testJob()

	if err := b.Install(job); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(sc.calls, "; "); !strings.Contains(got, "daemon-reload; enable --now example-job.timer") {
		t.Errorf("install must reload, then enable --now: %q", got)
	}
	st, err := b.Inspect(job)
	if err != nil || !st.Current() || !st.Managed {
		t.Fatalf("after install: %+v, %v", st, err)
	}

	delete(sc.on, "example-job.timer") // someone ran `systemctl --user disable --now`
	if st, _ = b.Inspect(job); st.Current() || st.Pending[0] != "enable --now example-job.timer" {
		t.Errorf("a disabled timer is pending: %+v", st)
	}
	sc.calls = nil
	if err := b.Install(job); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(sc.calls, "; "); strings.Contains(got, "daemon-reload") {
		t.Errorf("unchanged units need no reload: %q", got)
	}

	if err := b.Remove(job); err != nil {
		t.Fatal(err)
	}
	if st, _ = b.Inspect(job); st.Present {
		t.Errorf("after remove: %+v", st)
	}
}

// A unit a setup script copied before OPS-053 carries no marker.
func TestSystemd_UnitWithoutTheMarkerIsUnmanaged(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "example-job.timer"), []byte("[Timer]\nOnCalendar=daily\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	b := Systemd{Run: (&fakeSystemctl{on: map[string]bool{"example-job.timer": true}}).run, Dir: dir}
	st, err := b.Inspect(testJob())
	if err != nil || !st.Present || st.Managed || st.Current() {
		t.Errorf("legacy unit: %+v, %v; want present, unmanaged, not current", st, err)
	}
}

// errNotFound is schtasks' exit for a task it does not have.
var errNotFound = exitStatus(1)

// fakeSchtasks is the Task Scheduler: the XML each task was created from,
// handed back UTF-16 with a byte-order mark, as schtasks prints it.
type fakeSchtasks struct {
	tasks map[string]string
	fail  bool // the scheduler refuses every query
	calls []string
}

func (f *fakeSchtasks) run(args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	switch args[0] {
	case "/Query":
		if f.fail {
			return "", exitStatus(1)
		}
		if args[1] == "/FO" {
			var b strings.Builder
			for name := range f.tasks {
				b.WriteString(`"\` + name + `","N/A","Ready"` + "\r\n")
			}
			return b.String(), nil
		}
		x, ok := f.tasks[args[2]]
		if !ok {
			return "", errNotFound
		}
		return string(utf16LE(x)), nil
	case "/Create":
		raw, err := os.ReadFile(args[4])
		if err != nil {
			return "", err
		}
		f.tasks[args[2]] = decodeText(raw)
	case "/Delete":
		delete(f.tasks, args[2])
	}
	return "", nil
}

func TestTaskXML_IsWellFormedAndCarriesTheDigest(t *testing.T) {
	job := testJob()
	job.Program = `C:\Users\me\.local\bin\dotf.exe`
	x := TaskXML(job, `HOST\me`, LogonS4U)
	var doc struct {
		Desc    string `xml:"RegistrationInfo>Description"`
		Logon   string `xml:"Principals>Principal>LogonType"`
		Command string `xml:"Actions>Exec>Command"`
		Args    string `xml:"Actions>Exec>Arguments"`
		Start   string `xml:"Triggers>CalendarTrigger>StartBoundary"`
	}
	// encoding/xml refuses a UTF-16 declaration on a Go string; the body is what is checked.
	if err := xml.Unmarshal([]byte(strings.Replace(x, `encoding="UTF-16"`, `encoding="UTF-8"`, 1)), &doc); err != nil {
		t.Fatalf("task XML must be well-formed: %v", err)
	}
	if !strings.HasSuffix(doc.Desc, "[dotf:"+TaskDigest(job)+"]") || doc.Logon != LogonS4U ||
		doc.Command != job.Program || doc.Args != "update" || doc.Start != "2026-01-01T03:05:00" {
		t.Errorf("task XML fields: %+v", doc)
	}
}

func TestSchtasks_InstallThenCurrentThenRemove(t *testing.T) {
	st := &fakeSchtasks{tasks: map[string]string{}}
	elevated := false
	b := Schtasks{Run: st.run, User: `HOST\me`, Elevated: func() bool { return elevated }, TempDir: t.TempDir()}
	job := testJob()

	if err := b.Install(job); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(st.tasks["ExampleJob"], "<LogonType>InteractiveToken</LogonType>") {
		t.Error("a standard user registers an InteractiveToken task")
	}
	s, _ := b.Inspect(job)
	if !s.Current() || !s.Managed {
		t.Fatalf("after install: %+v", s)
	}
	st.calls = nil
	if err := b.Install(job); err != nil || len(st.calls) != 2 {
		t.Errorf("a second install must only list and query, ran %q (%v)", st.calls, err)
	}

	elevated = true
	if s, _ = b.Inspect(job); s.Current() {
		t.Error("an elevated caller upgrades an InteractiveToken task to S4U")
	}
	if err := b.Install(job); err != nil {
		t.Fatal(err)
	}
	elevated = false
	if s, _ = b.Inspect(job); !s.Current() {
		t.Errorf("a standard user never downgrades an S4U task: %+v", s)
	}

	job.Args = []string{"update", "--verbose"}
	if s, _ = b.Inspect(job); s.Current() {
		t.Error("a changed job changes the digest")
	}

	if err := b.Remove(job); err != nil {
		t.Fatal(err)
	}
	if s, _ = b.Inspect(job); s.Present {
		t.Errorf("after remove: %+v", s)
	}
}

// The task setup-windows.ps1 registered carries no digest.
// A refused query is an error, not an absent task: read as absence, opt-out
// would report the task removed while it still runs daily.
func TestSchtasks_ARefusedQueryIsNotAnAbsentTask(t *testing.T) {
	st := &fakeSchtasks{tasks: map[string]string{}}
	b := Schtasks{Run: st.run, User: `HOST\me`, TempDir: t.TempDir()}
	if err := b.Install(testJob()); err != nil {
		t.Fatal(err)
	}
	st.fail = true
	if _, err := b.Inspect(testJob()); err == nil {
		t.Error("inspect must fail when the scheduler refuses the query")
	}
	if err := b.Remove(testJob()); err == nil {
		t.Error("remove must fail when the scheduler refuses the query")
	}
	if _, ok := st.tasks["ExampleJob"]; !ok {
		t.Error("the task must survive a failed remove")
	}
}

func TestListsTask_MatchesTheRootFolderPathOnly(t *testing.T) {
	list := "\"\\Other\",\"N/A\",\"Ready\"\r\n\"\\Sub\\ExampleJob\",\"N/A\",\"Ready\"\r\n"
	if listsTask(list, "ExampleJob") {
		t.Error("a task of the same name in another folder is not this one")
	}
	if !listsTask(list+"\"\\ExampleJob\",\"N/A\",\"Ready\"\r\n", "ExampleJob") {
		t.Error("the root-folder task must be found")
	}
}

func TestSchtasks_TaskWithoutTheDigestIsUnmanaged(t *testing.T) {
	st := &fakeSchtasks{tasks: map[string]string{"ExampleJob": "<Task><RegistrationInfo><Description>dotfiles self-deploy</Description></RegistrationInfo></Task>"}}
	s, err := Schtasks{Run: st.run, TempDir: t.TempDir()}.Inspect(testJob())
	if err != nil || !s.Present || s.Managed || s.Current() {
		t.Errorf("legacy task: %+v, %v", s, err)
	}
}

func TestFor_PicksTheBackendOfTheOS(t *testing.T) {
	r := Runners{Launchctl: (&fakeLaunchctl{}).run, Systemctl: (&fakeSystemctl{}).run, Schtasks: (&fakeSchtasks{}).run}
	for goos, want := range map[string]string{"darwin": "schedule.Launchd", "linux": "schedule.Systemd", "windows": "schedule.Schtasks"} {
		b, err := For(goos, r)
		if err != nil || typeName(b) != want {
			t.Errorf("For(%s) = %s, %v", goos, typeName(b), err)
		}
	}
	if _, err := For("linux", Runners{}); !errors.Is(err, ErrUnsupported) {
		t.Errorf("an unwired runner is unsupported, got %v", err)
	}
}

func typeName(b Backend) string {
	switch b.(type) {
	case Launchd:
		return "schedule.Launchd"
	case Systemd:
		return "schedule.Systemd"
	case Schtasks:
		return "schedule.Schtasks"
	}
	return "<nil>"
}
