package doctor

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestCheckBWServeDaemon_Absent(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{BWServeStatus: func() (string, error) { return "absent", nil }}

	checkBWServeDaemon(sys, rep)

	if got := buf.String(); !strings.Contains(got, "no daemon running") {
		t.Fatalf("expected an absent-daemon info line, got: %s", got)
	}
}

func TestCheckBWServeDaemon_Locked(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{BWServeStatus: func() (string, error) { return "locked", nil }}

	checkBWServeDaemon(sys, rep)

	if got := buf.String(); !strings.Contains(got, "locked") {
		t.Fatalf("expected a locked-daemon info line, got: %s", got)
	}
}

func TestCheckBWServeDaemon_Unlocked(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{BWServeStatus: func() (string, error) { return "unlocked", nil }}

	checkBWServeDaemon(sys, rep)

	if got := buf.String(); !strings.Contains(got, "unlocked") {
		t.Fatalf("expected an unlocked-daemon pass line, got: %s", got)
	}
}

func TestCheckBWServeDaemon_StatusUnreadable(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{BWServeStatus: func() (string, error) { return "", errors.New("boom") }}

	checkBWServeDaemon(sys, rep)

	if got := buf.String(); !strings.Contains(got, "boom") {
		t.Fatalf("expected the underlying error surfaced, got: %s", got)
	}
}
