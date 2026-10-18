package vm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeGuest plays a booted clean-base: its agent keeps files in a map, the
// test user logs on after the first ask, and `schtasks /run` runs a step by
// writing what guestrun.ps1 would: its log, the converge report and its exit
// code.
type fakeGuest struct {
	t     *testing.T
	steps map[string]guestStep // by step name
	hangs bool                 // no step ever writes its exit file

	mu      sync.Mutex
	starts  [][]string
	imgs    [][]string
	procs   []*fakeProc
	files   map[string][]byte
	open    map[int]string
	nextH   int
	execs   []string
	status  map[int]map[string]any // guest-exec-status by pid
	asked   int                    // logon queries
	quits   int
	running bool // a QMP socket answers before the run starts
}

type guestStep struct {
	exit   string // the exit file's contents
	log    string
	report string // last.json; empty writes none
}

func report(result string, changed int, entries ...string) string {
	return `{"result":"` + result + `","changed":` + itoa(changed) + `,"entries":[` + strings.Join(entries, ",") + `]}`
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func (g *fakeGuest) Start(args []string, _ io.Writer) (Process, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	p := &fakeProc{done: make(chan error, 1)}
	g.starts = append(g.starts, args)
	g.procs = append(g.procs, p)
	return p, nil
}

// Img creates a run's overlay, the only image a run makes; anything else is
// refused rather than written to wherever its last argument points.
func (g *fakeGuest) Img(args ...string) error {
	g.imgs = append(g.imgs, args)
	if !slices.Contains(args, "-b") {
		return fmt.Errorf("fake qemu-img: not an overlay: %v", args)
	}
	return os.WriteFile(args[len(args)-1], []byte("overlay"), 0o644)
}

func (g *fakeGuest) Dial(socket string) (net.Conn, error) {
	host, guest := net.Pipe()
	g.t.Cleanup(func() { _ = host.Close(); _ = guest.Close() })
	if strings.HasSuffix(socket, "qmp.sock") {
		g.mu.Lock()
		running := g.running || len(g.procs) > 0
		g.mu.Unlock()
		if !running {
			return nil, errors.New("connect: no such file or directory")
		}
		go (&fakeHost{t: g.t}).serveQMP(guest)
		g.mu.Lock()
		g.quits++ // a run dials QMP only to stop a guest
		g.mu.Unlock()
		return host, nil
	}
	a := &fakeAgent{t: g.t, handlers: map[string]func(json.RawMessage) any{
		"guest-exec":        g.exec,
		"guest-exec-status": g.execStatus,
		"guest-file-open":   g.fopen,
		"guest-file-write":  g.fwrite,
		"guest-file-read":   g.fread,
		"guest-file-close":  func(json.RawMessage) any { return map[string]any{} },
	}}
	go a.serve(guest)
	return host, nil
}

func (g *fakeGuest) exec(a json.RawMessage) any {
	var e struct {
		Path string   `json:"path"`
		Arg  []string `json:"arg"`
	}
	_ = json.Unmarshal(a, &e)
	g.mu.Lock()
	defer g.mu.Unlock()
	line := e.Path + " " + strings.Join(e.Arg, " ")
	g.execs = append(g.execs, line)
	pid := len(g.execs)
	st := map[string]any{"exited": true, "exitcode": 0}
	switch {
	case strings.Contains(line, "Win32_ComputerSystem"):
		g.asked++
		if g.asked > 1 {
			st["out-data"] = base64.StdEncoding.EncodeToString([]byte("DOTF-VM\\dotf\r\n"))
		}
	case e.Path == "cmd.exe" && len(e.Arg) > 1 && e.Arg[1] == "del":
		delete(g.files, e.Arg[len(e.Arg)-1])
	case e.Path == "schtasks.exe" && e.Arg[0] == "/run":
		name := strings.TrimPrefix(e.Arg[len(e.Arg)-1], "dotf-")
		s, ok := g.steps[name]
		if !ok {
			g.t.Errorf("ran unknown step %q", name)
			break
		}
		if g.files[guestWork+`\src.zip`] == nil || g.files[guestWork+`\guestrun.ps1`] == nil {
			g.t.Errorf("step %s ran before the source and the script were copied in", name)
		}
		g.files[guestWork+`\`+name+".log"] = []byte("\xef\xbb\xbf" + s.log)
		if s.report != "" {
			g.files[guestReport] = []byte(s.report)
		}
		if !g.hangs {
			g.files[guestWork+`\`+name+".exit"] = []byte(s.exit + "\r\n")
		}
	}
	g.status[pid] = st
	return map[string]int{"pid": pid}
}

func (g *fakeGuest) execStatus(a json.RawMessage) any {
	var s struct {
		PID int `json:"pid"`
	}
	_ = json.Unmarshal(a, &s)
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.status[s.PID]
}

func (g *fakeGuest) fopen(a json.RawMessage) any {
	var o struct {
		Path, Mode string
	}
	_ = json.Unmarshal(a, &o)
	g.mu.Lock()
	defer g.mu.Unlock()
	if o.Mode == "rb" {
		if _, ok := g.files[o.Path]; !ok {
			return agentError("Failed to open file '" + o.Path + "'")
		}
	} else {
		g.files[o.Path] = nil
	}
	g.nextH++
	g.open[g.nextH] = o.Path
	return g.nextH
}

func (g *fakeGuest) fwrite(a json.RawMessage) any {
	var w struct {
		Handle int    `json:"handle"`
		Buf    string `json:"buf-b64"`
	}
	_ = json.Unmarshal(a, &w)
	b, _ := base64.StdEncoding.DecodeString(w.Buf)
	g.mu.Lock()
	defer g.mu.Unlock()
	p := g.open[w.Handle]
	g.files[p] = append(g.files[p], b...)
	return map[string]int{"count": len(b)}
}

func (g *fakeGuest) fread(a json.RawMessage) any {
	var r struct {
		Handle int `json:"handle"`
	}
	_ = json.Unmarshal(a, &r)
	g.mu.Lock()
	defer g.mu.Unlock()
	return map[string]any{"buf-b64": base64.StdEncoding.EncodeToString(g.files[g.open[r.Handle]]), "eof": true}
}

func (g *fakeGuest) ran(prefix string) []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []string
	for _, e := range g.execs {
		if strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return out
}

func newFakeGuest(t *testing.T, first, second guestStep) *fakeGuest {
	return &fakeGuest{t: t, files: map[string][]byte{}, open: map[int]string{}, status: map[int]map[string]any{},
		steps: map[string]guestStep{"first": first, "second": second}}
}

var (
	okStep      = guestStep{exit: "0", log: "converge: ok\n", report: report("ok", 3, `{"name":"tools","status":"ok","changes":3}`)}
	nothingStep = guestStep{exit: "0", log: "converge: ok, 0 changed\n", report: report("ok", 0, `{"name":"tools","status":"ok","changes":0}`)}
)

// sealedBase lays out a finished build: a read-only clean-base.
func sealedBase(t *testing.T) RunOptions {
	t.Helper()
	dir := stateDir(t)
	l := Layout{Dir: filepath.Join(dir, "win")}
	if err := os.MkdirAll(l.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{l.BaseDisk(), l.BaseVars()} {
		if err := os.WriteFile(f, []byte("base"), 0o444); err != nil {
			t.Fatal(err)
		}
	}
	src := filepath.Join(dir, "src.zip")
	if err := os.WriteFile(src, []byte("PK zip of the ref"), 0o644); err != nil {
		t.Fatal(err)
	}
	return RunOptions{
		Machine: Machine{Layout: l, Firmware: "/fw/code.fd", CPUs: 2, MemoryMB: 4096},
		Source:  src,
		Agent:   2 * time.Second,
		Step:    2 * time.Second,
		Poll:    5 * time.Millisecond,
		Out:     &bytes.Buffer{},
	}
}

func TestRun_ConvergesTwiceOnAThrowawayOverlay(t *testing.T) {
	o := sealedBase(t)
	g := newFakeGuest(t, okStep, nothingStep)
	if err := Run(g, o); err != nil {
		t.Fatalf("%v\n%s", err, o.Out)
	}
	l := o.Machine.Layout
	if len(g.imgs) != 1 || strings.Join(g.imgs[0], " ") != "create -q -f qcow2 -b "+l.BaseDisk()+" -F qcow2 "+l.Disk() {
		t.Errorf("qemu-img calls: %v", g.imgs)
	}
	if len(g.starts) != 1 {
		t.Fatalf("QEMU started %d times", len(g.starts))
	}
	args := strings.Join(g.starts[0], " ")
	if !strings.Contains(args, l.Disk()) || strings.Contains(args, "file="+l.BaseDisk()) || strings.Contains(args, "media=cdrom") {
		t.Errorf("a run must boot the overlay alone: %s", args)
	}
	if string(g.files[guestWork+`\src.zip`]) != "PK zip of the ref" {
		t.Error("the ref's archive did not reach the guest")
	}
	if !bytes.Equal(g.files[guestWork+`\guestrun.ps1`], guestRunScript) {
		t.Error("guestrun.ps1 did not reach the guest")
	}
	// Each step runs as the logged-on test user, not as the agent's
	// LocalSystem, whose profile is not the one a person converges.
	creates := g.ran("schtasks.exe /create")
	if len(creates) != 2 {
		t.Fatalf("scheduled tasks: %v", creates)
	}
	for _, c := range creates {
		if !strings.Contains(c, "/ru dotf") || !strings.Contains(c, "/it") || !strings.Contains(c, guestWork+`\guestrun.ps1`) {
			t.Errorf("a step does not run guestrun.ps1 as the logged-on user: %s", c)
		}
	}
	if g.asked < 2 {
		t.Error("a step was scheduled before the test user logged on")
	}
	// The second step is judged on its own report.
	if dels := g.ran("cmd.exe /c del"); len(dels) != 2 {
		t.Errorf("stale reports deleted: %v", dels)
	}
	out := o.Out.(*bytes.Buffer).String()
	if !strings.Contains(out, "converge: ok, 0 changed") || strings.Contains(out, "\xef\xbb\xbf") {
		t.Errorf("the guest's output was not relayed cleanly:\n%s", out)
	}
	if !g.procs[0].killed {
		t.Error("the guest was left running")
	}
	for _, f := range []string{l.Disk(), l.Vars()} {
		if _, err := os.Stat(f); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("the run left %s", f)
		}
	}
	if fi, _ := os.Stat(l.BaseDisk()); fi.Mode().Perm()&0o222 != 0 {
		t.Error("the run made clean-base writable")
	}
}

func TestRun_ASecondConvergeThatChangesSomethingFails(t *testing.T) {
	o := sealedBase(t)
	g := newFakeGuest(t, okStep, guestStep{exit: "0", report: report("ok", 1, `{"name":"powercfg","status":"ok","changes":1}`)})
	err := Run(g, o)
	if err == nil || !strings.Contains(err.Error(), "powercfg (1)") {
		t.Fatalf("want the changing step named, got %v", err)
	}
	if !g.procs[0].killed {
		t.Error("a failed run left the guest running")
	}
}

func TestRun_AFailedFirstStepStopsBeforeTheSecond(t *testing.T) {
	o := sealedBase(t)
	g := newFakeGuest(t, guestStep{exit: "1", log: "install: download failed\n", report: report("failed", 0, `{"name":"tools","status":"failed"}`)}, nothingStep)
	err := Run(g, o)
	if err == nil || !strings.Contains(err.Error(), "tools") || !strings.Contains(err.Error(), "exit") {
		t.Fatalf("want the exit code and the failed step, got %v", err)
	}
	if len(g.ran("schtasks.exe /run /tn dotf-second")) != 0 {
		t.Error("the second step ran after the first failed")
	}
	if !strings.Contains(o.Out.(*bytes.Buffer).String(), "install: download failed") {
		t.Error("the failed step's output was not shown")
	}
}

// install.ps1 can fail before dotf runs, and then no report exists: the run
// must not pass on a missing one.
func TestRun_AStepWithoutAReportFails(t *testing.T) {
	o := sealedBase(t)
	g := newFakeGuest(t, guestStep{exit: "0", log: "nothing\n"}, nothingStep)
	if err := Run(g, o); err == nil || !strings.Contains(err.Error(), "no report") {
		t.Fatalf("want a missing-report error, got %v", err)
	}
}

func TestRun_AStepThatNeverEndsTimesOut(t *testing.T) {
	o := sealedBase(t)
	o.Step = 50 * time.Millisecond
	g := newFakeGuest(t, okStep, nothingStep)
	g.hangs = true
	err := Run(g, o)
	if err == nil || !strings.Contains(err.Error(), "did not finish") {
		t.Fatalf("want a timeout, got %v", err)
	}
	if !strings.Contains(o.Out.(*bytes.Buffer).String(), "converge: ok") {
		t.Error("a step that timed out did not show its output so far")
	}
}

func TestRun_KeepLeavesTheGuestRunning(t *testing.T) {
	o := sealedBase(t)
	o.Keep = true
	g := newFakeGuest(t, okStep, nothingStep)
	if err := Run(g, o); err != nil {
		t.Fatal(err)
	}
	if g.procs[0].killed {
		t.Error("--keep stopped the guest")
	}
	if _, err := os.Stat(o.Machine.Layout.Disk()); err != nil {
		t.Errorf("--keep removed the overlay: %v", err)
	}
	if !strings.Contains(o.Out.(*bytes.Buffer).String(), "dotf vm windows stop") {
		t.Error("--keep does not say how to stop the guest")
	}
}

func TestRun_NeedsASealedCleanBase(t *testing.T) {
	o := sealedBase(t)
	l := o.Machine.Layout
	if err := os.Chmod(l.BaseDisk(), 0o644); err != nil { // a build still running, or one that stopped
		t.Fatal(err)
	}
	g := newFakeGuest(t, okStep, nothingStep)
	if err := Run(g, o); err == nil || !strings.Contains(err.Error(), "--rebuild") {
		t.Fatalf("want the unfinished base refused, got %v", err)
	}
	if err := os.Remove(l.BaseDisk()); err != nil {
		t.Fatal(err)
	}
	if err := Run(g, o); err == nil || !strings.Contains(err.Error(), "dotf vm windows build") {
		t.Fatalf("want a missing base to name the build, got %v", err)
	}
	if len(g.starts)+len(g.imgs) != 0 {
		t.Error("a run started without a sealed clean-base")
	}
}

// A guest left by --keep holds the overlay a new run would replace.
func TestRun_RefusesWhileAGuestRuns(t *testing.T) {
	o := sealedBase(t)
	g := newFakeGuest(t, okStep, nothingStep)
	g.running = true
	if err := Run(g, o); err == nil || !strings.Contains(err.Error(), "dotf vm windows stop") {
		t.Fatalf("want a running guest refused, got %v", err)
	}
	if len(g.starts)+len(g.imgs) != 0 {
		t.Error("a run started over a running guest")
	}
}

func TestStop_QuitsTheGuestAndRemovesTheOverlay(t *testing.T) {
	o := sealedBase(t)
	l := o.Machine.Layout
	for _, f := range []string{l.Disk(), l.Vars()} {
		if err := os.WriteFile(f, []byte("overlay"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	g := newFakeGuest(t, okStep, nothingStep)
	g.running = true
	if err := Stop(g, l, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if g.quits != 1 {
		t.Errorf("QMP dialled %d times, want one quit", g.quits)
	}
	for _, f := range []string{l.Disk(), l.Vars()} {
		if _, err := os.Stat(f); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("stop left %s", f)
		}
	}
	// Nothing running is not an error: stop is idempotent.
	g.running = false
	if err := Stop(g, l, &bytes.Buffer{}); err != nil {
		t.Errorf("a second stop: %v", err)
	}
}

// Windows PowerShell decodes native output with the console's OEM code page:
// a live run printed dotf's "→" as "ΓåÆ". The console must be UTF-8 before
// either step starts its child.
func TestGuestRun_DecodesNativeOutputAsUTF8(t *testing.T) {
	code := codeLines(string(guestRunScript))
	enc := strings.Index(code, "[Console]::OutputEncoding = [System.Text.Encoding]::UTF8")
	if enc < 0 {
		t.Fatal("guestrun.ps1 leaves the console on its OEM code page")
	}
	if first := strings.Index(code, "& powershell.exe"); first < enc {
		t.Error("guestrun.ps1 sets the console's encoding after it starts install.ps1")
	}
}

// --dotf converges with a source build instead of the release the ref pins:
// the binary reaches the guest beside the archive, and a run without it copies
// nothing more.
func TestRun_CopiesADotfBuildIntoTheGuest(t *testing.T) {
	o := sealedBase(t)
	o.Dotf = filepath.Join(t.TempDir(), "dotf.exe")
	if err := os.WriteFile(o.Dotf, []byte("MZ dev build"), 0o755); err != nil {
		t.Fatal(err)
	}
	g := newFakeGuest(t, okStep, nothingStep)
	if err := Run(g, o); err != nil {
		t.Fatalf("%v\n%s", err, o.Out)
	}
	if string(g.files[guestWork+`\dotf.exe`]) != "MZ dev build" {
		t.Error("the dotf build did not reach the guest")
	}

	o = sealedBase(t)
	g = newFakeGuest(t, okStep, nothingStep)
	if err := Run(g, o); err != nil {
		t.Fatalf("%v\n%s", err, o.Out)
	}
	if _, ok := g.files[guestWork+`\dotf.exe`]; ok {
		t.Error("a run without --dotf copied a dotf into the guest")
	}
}

// install.ps1 keeps a dotf on PATH that reports "dev" and replaces any other,
// so the copied build must be in ~\.local\bin, the dir step two runs, and on
// PATH before install.ps1 starts.
func TestGuestRun_PutsADotfBuildOnPATHBeforeInstall(t *testing.T) {
	code := codeLines(string(guestRunScript))
	install := strings.Index(code, "& powershell.exe")
	for _, want := range []string{"Copy-Item -Path $dev -Destination $installed", `$env:PATH = "$bin;$env:PATH"`} {
		at := strings.Index(code, want)
		if at < 0 {
			t.Errorf("guestrun.ps1 lacks %q", want)
			continue
		}
		if at > install {
			t.Errorf("guestrun.ps1 runs %q after it starts install.ps1", want)
		}
	}
}

// install.ps1 keeps a dotf that reports "dev", but nothing else asserts that
// the build is what converged: each step compares the binary it ran with the
// copy the host put in, after the run, and a step whose binary differs fails.
func TestGuestRun_FailsAStepThatDidNotRunTheDotfBuild(t *testing.T) {
	code := codeLines(string(guestRunScript))
	run := strings.Index(code, "$code = $LASTEXITCODE")
	check := strings.Index(code, "(Get-FileHash -Path $dev).Hash -ne (Get-FileHash -Path $installed).Hash")
	if check < 0 {
		t.Fatal("guestrun.ps1 never compares the dotf that ran with the --dotf build")
	}
	if check < run {
		t.Error("guestrun.ps1 compares the binaries before the step has run")
	}
	if !strings.Contains(code[check:], "$code = 1") {
		t.Error("a mismatch does not fail the step")
	}
}
