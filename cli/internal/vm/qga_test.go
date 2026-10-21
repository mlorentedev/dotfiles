package vm

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeAgent answers QEMU guest agent commands on the far end of a pipe, the
// way qemu-ga answers on its virtio-serial port: one JSON reply per line.
type fakeAgent struct {
	t        *testing.T
	stale    []string                                  // lines sent before the first reply, as a previous client's leftovers
	handlers map[string]func(args json.RawMessage) any // a returned error string becomes {"error":...}
	seen     []string
}

type agentError string

func (f *fakeAgent) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	first := true
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return
		}
		var req struct {
			Execute   string          `json:"execute"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(line, &req); err != nil {
			f.t.Errorf("not one JSON command per line: %q", line)
			return
		}
		f.seen = append(f.seen, req.Execute)
		if first {
			for _, s := range f.stale {
				_, _ = conn.Write([]byte(s + "\n"))
			}
			first = false
		}
		var reply any
		h, ok := f.handlers[req.Execute]
		switch {
		case req.Execute == "guest-sync":
			var a struct {
				ID int64 `json:"id"`
			}
			_ = json.Unmarshal(req.Arguments, &a)
			reply = map[string]any{"return": a.ID}
		case !ok:
			reply = map[string]any{"error": map[string]string{"class": "CommandNotFound", "desc": "no " + req.Execute}}
		default:
			ret := h(req.Arguments)
			if e, isErr := ret.(agentError); isErr {
				reply = map[string]any{"error": map[string]string{"class": "GenericError", "desc": string(e)}}
			} else {
				reply = map[string]any{"return": ret}
			}
		}
		b, _ := json.Marshal(reply)
		_, _ = conn.Write(append(b, '\n'))
	}
}

func newFakeQGA(t *testing.T, f *fakeAgent) *QGA {
	t.Helper()
	f.t = t
	host, guest := net.Pipe()
	go f.serve(guest)
	t.Cleanup(func() { _ = host.Close() })
	q := NewQGA(host)
	q.Timeout = 2 * time.Second
	return q
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// A previous client can leave replies in the channel; Sync must read past
// them to the reply that answers its own id, or every later call is off by one.
func TestQGA_SyncSkipsAPreviousClientsReplies(t *testing.T) {
	f := &fakeAgent{stale: []string{`{"return": {"pid": 7}}`, `{"return": 41}`}}
	q := newFakeQGA(t, f)
	if err := q.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := q.Sync(); err != nil {
		t.Fatalf("a second Sync must find its own reply, not a stale one: %v", err)
	}
}

func TestQGA_AnAgentErrorNamesTheCommand(t *testing.T) {
	q := newFakeQGA(t, &fakeAgent{handlers: map[string]func(json.RawMessage) any{
		"guest-exec": func(json.RawMessage) any { return agentError("Failed to execute child process") },
	}})
	_, err := q.Run("C:\\missing.exe", nil)
	if err == nil || !strings.Contains(err.Error(), "guest-exec") || !strings.Contains(err.Error(), "Failed to execute") {
		t.Fatalf("want an error naming guest-exec and the agent's reason, got %v", err)
	}
}

// Run starts the process with its output captured and polls until it exits.
func TestQGA_RunReturnsExitCodeAndOutput(t *testing.T) {
	polls := 0
	var execArgs struct {
		Path    string   `json:"path"`
		Arg     []string `json:"arg"`
		Capture bool     `json:"capture-output"`
	}
	q := newFakeQGA(t, &fakeAgent{handlers: map[string]func(json.RawMessage) any{
		"guest-exec": func(a json.RawMessage) any {
			_ = json.Unmarshal(a, &execArgs)
			return map[string]int{"pid": 42}
		},
		"guest-exec-status": func(json.RawMessage) any {
			polls++
			if polls < 3 {
				return map[string]bool{"exited": false}
			}
			return map[string]any{"exited": true, "exitcode": 3, "out-data": b64("hello\r\n"), "err-data": b64("warn")}
		},
	}})
	q.Poll = time.Millisecond
	res, err := q.Run("powershell.exe", []string{"-NoProfile", "-Command", "exit 3"})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 3 || string(res.Stdout) != "hello\r\n" || string(res.Stderr) != "warn" {
		t.Errorf("got %+v", res)
	}
	if execArgs.Path != "powershell.exe" || len(execArgs.Arg) != 3 || !execArgs.Capture {
		t.Errorf("guest-exec arguments: %+v", execArgs)
	}
	if polls != 3 {
		t.Errorf("polled %d times, want until exited (3)", polls)
	}
}

// A file is pushed in chunks small enough for the agent's line parser and
// pulled back until the agent reports EOF; the handle is closed both ways.
func TestQGA_WriteAndReadAFile(t *testing.T) {
	var stored bytes.Buffer
	readAt := 0
	f := &fakeAgent{handlers: map[string]func(json.RawMessage) any{
		"guest-file-open": func(json.RawMessage) any { return 5 },
		"guest-file-write": func(a json.RawMessage) any {
			var w struct {
				Handle int    `json:"handle"`
				Buf    string `json:"buf-b64"`
			}
			_ = json.Unmarshal(a, &w)
			b, _ := base64.StdEncoding.DecodeString(w.Buf)
			if len(b) > qgaChunk {
				return agentError("chunk too large")
			}
			stored.Write(b)
			return map[string]any{"count": len(b), "eof": false}
		},
		"guest-file-read": func(json.RawMessage) any {
			end := min(readAt+1000, stored.Len())
			chunk := stored.Bytes()[readAt:end]
			readAt = end
			return map[string]any{"count": len(chunk), "buf-b64": base64.StdEncoding.EncodeToString(chunk), "eof": readAt == stored.Len()}
		},
		"guest-file-close": func(json.RawMessage) any { return map[string]any{} },
	}}
	q := newFakeQGA(t, f)
	payload := bytes.Repeat([]byte("dotfiles\n"), qgaChunk/4)
	if err := q.WriteFile(`C:\Users\dotf\src.zip`, bytes.NewReader(payload)); err != nil {
		t.Fatal(err)
	}
	got, err := q.ReadFile(`C:\Users\dotf\src.zip`)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("round trip lost data: wrote %d bytes, read %d", len(payload), len(got))
	}
	closes := strings.Count(strings.Join(f.seen, " "), "guest-file-close")
	if closes != 2 {
		t.Errorf("closed %d handles, want 2: %v", closes, f.seen)
	}
}

// A live agent reporting a process that never exits must not hang the host
// either: Timeout bounds each reply, Wait bounds the whole process (#2820).
func TestQGA_RunGivesUpOnAProcessThatNeverExits(t *testing.T) {
	q := newFakeQGA(t, &fakeAgent{handlers: map[string]func(json.RawMessage) any{
		"guest-exec":        func(json.RawMessage) any { return map[string]int{"pid": 42} },
		"guest-exec-status": func(json.RawMessage) any { return map[string]bool{"exited": false} },
	}})
	q.Poll = time.Millisecond
	q.Wait = 50 * time.Millisecond
	done := make(chan error, 1)
	go func() {
		_, err := q.Run("powershell.exe", []string{"-NoProfile", "-Command", "Get-CimInstance Win32_ComputerSystem"})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("want an error from a process that never exits")
		}
		if !strings.Contains(err.Error(), "powershell.exe") || !strings.Contains(err.Error(), "42") {
			t.Errorf("the error should name the command and its pid: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run still polling after 5s with Wait = 50ms")
	}
}

// Every caller gets the bound without setting it: zero would wait forever.
func TestNewQGA_BoundsEveryRun(t *testing.T) {
	host, guest := net.Pipe()
	defer func() { _ = host.Close(); _ = guest.Close() }()
	if q := NewQGA(host); q.Wait <= 0 {
		t.Errorf("NewQGA left Wait at %s, so Run polls a hung process forever", q.Wait)
	}
}

// A guest that stops answering must not hang the host forever.
func TestQGA_ASilentAgentTimesOut(t *testing.T) {
	host, guest := net.Pipe()
	defer func() { _ = host.Close(); _ = guest.Close() }()
	go func() { _, _ = bufio.NewReader(guest).ReadBytes('\n') }() // reads, never answers
	q := NewQGA(host)
	q.Timeout = 50 * time.Millisecond
	if err := q.Sync(); err == nil {
		t.Fatal("want a timeout from an agent that never answers")
	}
}
