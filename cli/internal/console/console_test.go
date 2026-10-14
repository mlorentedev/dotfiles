package console

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func env(kv map[string]string) func(string) string {
	return func(k string) string { return kv[k] }
}

// CLI-114 AC1: colour is a terminal's, and NO_COLOR (non-empty, per
// no-color.org) or TERM=dumb turns it off.
func TestColorEnabled(t *testing.T) {
	for _, c := range []struct {
		name     string
		env      map[string]string
		terminal bool
		want     bool
	}{
		{"a terminal", map[string]string{"TERM": "xterm-256color"}, true, true},
		{"a terminal with no TERM", nil, true, true},
		{"a pipe", map[string]string{"TERM": "xterm-256color"}, false, false},
		{"NO_COLOR", map[string]string{"NO_COLOR": "1"}, true, false},
		{"NO_COLOR with any value", map[string]string{"NO_COLOR": "false"}, true, false},
		{"an empty NO_COLOR", map[string]string{"NO_COLOR": ""}, true, true},
		{"TERM=dumb", map[string]string{"TERM": "dumb"}, true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := ColorEnabled(env(c.env), c.terminal); got != c.want {
				t.Errorf("ColorEnabled = %v, want %v", got, c.want)
			}
		})
	}
}

func TestIsTerminal_NeverForANonFile(t *testing.T) {
	if IsTerminal(&bytes.Buffer{}) {
		t.Error("a buffer is not a terminal")
	}
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if IsTerminal(f) {
		t.Error("a regular file is not a terminal")
	}
}

// Paint wraps text in its codes and a reset only when colour is on, so a pipe
// gets the bare text.
func TestPainter(t *testing.T) {
	on, off := Painter{On: true}, Painter{}
	if got := on.Paint("x", Red, Bold); got != "\033[31m\033[1mx\033[0m" {
		t.Errorf("on: %q", got)
	}
	if got := off.Paint("x", Red, Bold); got != "x" {
		t.Errorf("off: %q", got)
	}
	if got := on.Paint("x"); got != "x" {
		t.Errorf("no codes: %q, want the bare text", got)
	}
}

func TestDuration(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                      "0ms",
		850 * time.Millisecond: "850ms",
		999*time.Millisecond + 600*time.Microsecond: "1.0s",
		12340 * time.Millisecond:                    "12.3s",
		59999 * time.Millisecond:                    "1m00s",
		125 * time.Second:                           "2m05s",
		2*time.Hour + 3*time.Minute:                 "2h03m",
	} {
		if got := Duration(d); got != want {
			t.Errorf("Duration(%v) = %q, want %q", d, got, want)
		}
	}
}
