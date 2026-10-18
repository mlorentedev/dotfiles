package vm

import (
	"bufio"
	"encoding/json"
	"net"
	"strings"
	"testing"
)

// fakeQMP greets, then answers each command through handle, sending an event
// before every reply as QEMU may.
func fakeQMP(t *testing.T, handle func(cmd string, args json.RawMessage) string) (*QMP, *[]string) {
	t.Helper()
	host, guest := net.Pipe()
	t.Cleanup(func() { _ = host.Close(); _ = guest.Close() })
	var seen []string
	go func() {
		w := bufio.NewWriter(guest)
		send := func(s string) { _, _ = w.WriteString(s + "\n"); _ = w.Flush() }
		send(`{"QMP": {"version": {"qemu": {"major": 11}}, "capabilities": []}}`)
		r := bufio.NewReader(guest)
		for {
			line, err := r.ReadBytes('\n')
			if err != nil {
				return
			}
			var req struct {
				Execute   string          `json:"execute"`
				Arguments json.RawMessage `json:"arguments"`
			}
			_ = json.Unmarshal(line, &req)
			seen = append(seen, strings.TrimSpace(string(line)))
			send(`{"event": "RTC_CHANGE", "data": {}}`)
			send(handle(req.Execute, req.Arguments))
		}
	}()
	q, err := NewQMP(host)
	if err != nil {
		t.Fatal(err)
	}
	return q, &seen
}

func TestQMP_NegotiatesThenSkipsEvents(t *testing.T) {
	q, seen := fakeQMP(t, func(cmd string, _ json.RawMessage) string {
		if cmd == "query-status" {
			return `{"return": {"status": "running", "running": true}}`
		}
		return `{"return": {}}`
	})
	s, err := q.Status()
	if err != nil || s != "running" {
		t.Fatalf("status %q, %v", s, err)
	}
	if (*seen)[0] != `{"execute":"qmp_capabilities"}` {
		t.Errorf("first command %s, want the capabilities handshake", (*seen)[0])
	}
}

func TestQMP_PressKeyAndScreenshot(t *testing.T) {
	q, seen := fakeQMP(t, func(string, json.RawMessage) string { return `{"return": {}}` })
	if err := q.PressKey("ret"); err != nil {
		t.Fatal(err)
	}
	if err := q.Screenshot("/state/stuck.png"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		`{"arguments":{"keys":[{"data":"ret","type":"qcode"}]},"execute":"send-key"}`,
		`{"arguments":{"filename":"/state/stuck.png","format":"png"},"execute":"screendump"}`,
	}
	for i, w := range want {
		if (*seen)[i+1] != w {
			t.Errorf("command %d: %s, want %s", i+1, (*seen)[i+1], w)
		}
	}
}

func TestQMP_AnErrorNamesTheCommand(t *testing.T) {
	q, _ := fakeQMP(t, func(cmd string, _ json.RawMessage) string {
		if cmd == "qmp_capabilities" {
			return `{"return": {}}`
		}
		return `{"error": {"class": "GenericError", "desc": "no display"}}`
	})
	err := q.Screenshot("/x.png")
	if err == nil || err.Error() != "screendump: GenericError: no display" {
		t.Fatalf("got %v", err)
	}
}

func TestQMP_RefusesAPeerWithoutAGreeting(t *testing.T) {
	host, guest := net.Pipe()
	defer func() { _ = host.Close(); _ = guest.Close() }()
	go func() { _, _ = guest.Write([]byte(`{"return": {}}` + "\n")) }()
	if _, err := NewQMP(host); err == nil {
		t.Fatal("want an error without a QMP banner")
	}
}
