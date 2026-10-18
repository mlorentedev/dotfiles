package vm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeProc struct {
	done   chan error
	killed bool
}

func (p *fakeProc) Done() <-chan error { return p.done }

// Kill ends the process as a signal would: Done reports it.
func (p *fakeProc) Kill() error {
	p.killed = true
	select {
	case p.done <- errors.New("signal: killed"):
	default:
	}
	return nil
}

// fakeHost plays QEMU: the install exits on its own (the guest shut itself
// down) unless installHangs, and the check boot's agent answers ver and
// sc query sshd, then powers the guest off on guest-shutdown.
type fakeHost struct {
	t            *testing.T
	installHangs bool
	sshdExit     int

	mu     sync.Mutex
	starts [][]string
	imgs   [][]string
	procs  []*fakeProc
	qmp    []string
	keys   []string
	agent  *fakeAgent
}

func (h *fakeHost) Start(args []string, _ io.Writer) (Process, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	p := &fakeProc{done: make(chan error, 1)}
	if len(h.starts) == 0 && !h.installHangs {
		go func() { time.Sleep(50 * time.Millisecond); p.done <- nil }()
	}
	h.starts = append(h.starts, args)
	h.procs = append(h.procs, p)
	return p, nil
}

func (h *fakeHost) Img(args ...string) error {
	h.imgs = append(h.imgs, args)
	if args[0] == "create" {
		return os.WriteFile(args[len(args)-2], []byte("qcow2"), 0o644)
	}
	return nil
}

func (h *fakeHost) Dial(socket string) (net.Conn, error) {
	host, guest := net.Pipe()
	h.t.Cleanup(func() { _ = host.Close(); _ = guest.Close() })
	if strings.HasSuffix(socket, "qmp.sock") {
		go h.serveQMP(guest)
		return host, nil
	}
	h.mu.Lock()
	p := h.procs[len(h.procs)-1]
	h.mu.Unlock()
	pids := map[string]int{"cmd.exe": 1, "sc.exe": 2}
	h.agent = &fakeAgent{t: h.t, handlers: map[string]func(json.RawMessage) any{
		"guest-exec": func(a json.RawMessage) any {
			var e struct {
				Path string `json:"path"`
			}
			_ = json.Unmarshal(a, &e)
			return map[string]int{"pid": pids[e.Path]}
		},
		"guest-exec-status": func(a json.RawMessage) any {
			var s struct {
				PID int `json:"pid"`
			}
			_ = json.Unmarshal(a, &s)
			if s.PID == 1 {
				return map[string]any{"exited": true, "exitcode": 0, "out-data": b64("Microsoft Windows [Version 10.0.26200]\r\n")}
			}
			return map[string]any{"exited": true, "exitcode": h.sshdExit}
		},
		"guest-shutdown": func(json.RawMessage) any { p.done <- nil; return map[string]any{} },
	}}
	go h.agent.serve(guest)
	return host, nil
}

func (h *fakeHost) serveQMP(conn net.Conn) {
	w := bufio.NewWriter(conn)
	_, _ = w.WriteString(`{"QMP": {"version": {}, "capabilities": []}}` + "\n")
	_ = w.Flush()
	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return
		}
		var req struct {
			Execute   string `json:"execute"`
			Arguments struct {
				Keys []struct {
					Data string `json:"data"`
				} `json:"keys"`
			} `json:"arguments"`
		}
		_ = json.Unmarshal(line, &req)
		h.mu.Lock()
		h.qmp = append(h.qmp, req.Execute)
		for _, k := range req.Arguments.Keys {
			h.keys = append(h.keys, k.Data)
		}
		h.mu.Unlock()
		_, _ = w.WriteString(`{"return": {}}` + "\n")
		_ = w.Flush()
	}
}

func (h *fakeHost) count(cmd string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, c := range h.qmp {
		if c == cmd {
			n++
		}
	}
	return n
}

func buildOptions(t *testing.T) BuildOptions {
	t.Helper()
	iso, virtio := []byte("windows"), []byte("virtio")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(map[string][]byte{"/win.iso": iso, "/virtio.iso": virtio}[r.URL.Path])
	}))
	t.Cleanup(srv.Close)
	dir := stateDir(t)
	vars := filepath.Join(dir, "template-vars.fd")
	if err := os.WriteFile(vars, []byte("vars"), 0o644); err != nil {
		t.Fatal(err)
	}
	return BuildOptions{
		Machine:      Machine{Layout: Layout{Dir: filepath.Join(dir, "win")}, Firmware: "/fw/code.fd", CPUs: 2, MemoryMB: 4096},
		VarsTemplate: vars,
		Windows:      Medium{Name: "windows", SHA256: sum(iso)},
		Virtio:       Medium{Name: "virtio", URL: srv.URL + "/virtio.iso", SHA256: sum(virtio)},
		ISO:          srv.URL + "/win.iso",
		Client:       srv.Client(),
		DiskSize:     "64G",
		BootKeys:     30 * time.Millisecond,
		KeyEvery:     5 * time.Millisecond,
		Install:      2 * time.Second,
		Agent:        2 * time.Second,
		Out:          &bytes.Buffer{},
	}
}

func TestBuild_InstallsChecksAndSealsCleanBase(t *testing.T) {
	o := buildOptions(t)
	h := &fakeHost{t: t, sshdExit: 1060}
	if err := Build(h, o); err != nil {
		t.Fatalf("%v\n%s", err, o.Out)
	}
	l := o.Machine.Layout
	if len(h.imgs) != 1 || strings.Join(h.imgs[0], " ") != "create -q -f qcow2 "+l.BaseDisk()+" 64G" {
		t.Errorf("qemu-img calls: %v", h.imgs)
	}
	if len(h.starts) != 2 {
		t.Fatalf("QEMU started %d times, want the install and the check boot", len(h.starts))
	}
	install, check := strings.Join(h.starts[0], " "), strings.Join(h.starts[1], " ")
	if !strings.Contains(install, "fat:ro:"+l.UnattendDir()) || !strings.Contains(install, l.BaseDisk()) {
		t.Errorf("the install boots without its media or off clean-base: %s", install)
	}
	if strings.Contains(check, "media=cdrom") || strings.Contains(check, "fat:ro:") || !strings.Contains(check, l.BaseDisk()) {
		t.Errorf("the check boot must boot clean-base alone: %s", check)
	}
	if h.count("send-key") < 2 {
		t.Errorf("pressed %d keys at the ISO's prompt", h.count("send-key"))
	}
	// Setup's window takes focus within the key window, its Cancel button
	// focused: Enter or Space there asks to quit the install (measured live).
	for _, k := range h.keys {
		if k == "ret" || k == "spc" || k == "kp_enter" {
			t.Errorf("pressed %q, which activates Setup's focused Cancel button", k)
			break
		}
	}
	if !strings.Contains(strings.Join(h.agent.seen, " "), "guest-shutdown") {
		t.Error("the check boot did not power the guest off")
	}
	for _, f := range []string{l.BaseDisk(), l.BaseVars()} {
		fi, err := os.Stat(f)
		if err != nil || fi.Mode().Perm()&0o222 != 0 {
			t.Errorf("%s is not sealed read-only: %v %v", f, fi.Mode(), err)
		}
	}
	if b, _ := os.ReadFile(l.BaseVars()); string(b) != "vars" {
		t.Errorf("clean-base's UEFI variables are not the template's: %q", b)
	}

	// A sealed clean-base is a complete build: nothing runs again.
	if err := Build(h, o); err != nil {
		t.Fatal(err)
	}
	if len(h.starts) != 2 || len(h.imgs) != 1 {
		t.Errorf("a second Build started QEMU or qemu-img again")
	}
}

// A guest that already has sshd would make every #2672 run a false pass.
func TestBuild_RefusesACleanBaseWithSSHD(t *testing.T) {
	o := buildOptions(t)
	h := &fakeHost{t: t, sshdExit: 0}
	err := Build(h, o)
	if err == nil || !strings.Contains(err.Error(), "no sshd") {
		t.Fatalf("want the sshd check to fail, got %v", err)
	}
	assertNoBase(t, o.Machine.Layout)
	if !h.procs[1].killed {
		t.Error("the check boot's QEMU was left running")
	}
}

// A headless install that stops leaves nothing but its screen to diagnose.
func TestBuild_AnInstallThatHangsLeavesAScreenshot(t *testing.T) {
	o := buildOptions(t)
	o.Install = 100 * time.Millisecond
	h := &fakeHost{t: t, installHangs: true}
	err := Build(h, o)
	if err == nil || !strings.Contains(err.Error(), o.Machine.Layout.Stuck()) {
		t.Fatalf("want an error naming the screenshot, got %v", err)
	}
	if h.count("screendump") != 1 || !h.procs[0].killed {
		t.Errorf("screendumps %d, killed %v", h.count("screendump"), h.procs[0].killed)
	}
	assertNoBase(t, o.Machine.Layout)
}

func TestBuild_AnUnfinishedBaseNeedsRebuild(t *testing.T) {
	o := buildOptions(t)
	l := o.Machine.Layout
	if err := os.MkdirAll(l.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(l.BaseDisk(), []byte("half"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := &fakeHost{t: t, sshdExit: 1060}
	if err := Build(h, o); err == nil || !strings.Contains(err.Error(), "--rebuild") {
		t.Fatalf("want an error naming --rebuild, got %v", err)
	}
	if len(h.starts) != 0 {
		t.Fatal("an unfinished base started a build without --rebuild")
	}
	o.Rebuild = true
	if err := Build(h, o); err != nil {
		t.Fatal(err)
	}
}

// The ISO must hash to its pin before anything boots from it.
func TestBuild_AWrongISOStopsBeforeQEMU(t *testing.T) {
	o := buildOptions(t)
	o.Windows.SHA256 = sum([]byte("another release"))
	h := &fakeHost{t: t, sshdExit: 1060}
	if err := Build(h, o); err == nil || !strings.Contains(err.Error(), "SHA256") {
		t.Fatalf("want a hash error, got %v", err)
	}
	if len(h.starts)+len(h.imgs) != 0 {
		t.Error("QEMU or qemu-img ran with an unverified ISO")
	}
}

// stateDir is a guest state dir for one test. On macOS it lies under /tmp:
// the default $TMPDIR there is long enough to push the guest's sockets past
// sun_path. Elsewhere the default temp dir is short enough.
func stateDir(t *testing.T) string {
	t.Helper()
	base := ""
	if runtime.GOOS == "darwin" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "dvm")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func assertNoBase(t *testing.T, l Layout) {
	t.Helper()
	for _, f := range []string{l.BaseDisk(), l.BaseVars()} {
		if _, err := os.Stat(f); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("a failed build left %s", f)
		}
	}
}
