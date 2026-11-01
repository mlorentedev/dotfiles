package vm

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestCopySession(t *testing.T) {
	for _, c := range []struct {
		src, dst, want string
		wantErr        bool
	}{
		{"f.txt", "x:/tmp/f", "x", false},
		{"x:/tmp/f", "f.txt", "x", false},
		{"./a:b", "x:/tmp/f", "x", false},
		{"/abs/path", "sshd-proof:rel/f", "sshd-proof", false},
		{"f.txt", "g.txt", "", true},
		{"x:/a", "y:/b", "", true},
		{"f.txt", "x:", "", true},
		{"f.txt", "Bad Name:/tmp/f", "", true},
	} {
		got, err := CopySession(c.src, c.dst)
		if got != c.want || (err != nil) != c.wantErr {
			t.Errorf("CopySession(%q, %q) = %q, %v", c.src, c.dst, got, err)
		}
	}
}

// AC4: both directions go through limactl copy, with no share and no mount,
// over scp: rsync's remote shell splits a path at a space and copies a directory
// to nowhere with exit 0 (measured on Lima 2.2.1).
func TestCopyBothWays(t *testing.T) {
	for _, c := range []struct{ src, dst, want string }{
		{"./f", "x:/tmp/a b", "copy --backend=scp ./f dotf-x:/tmp/a b"},
		{"x:/tmp/f", "./back", "copy --backend=scp dotf-x:/tmp/f ./back"},
	} {
		f := &listLima{}
		r := heldSession(t, f).Copy(context.Background(), c.src, c.dst, io.Discard)
		if r.Status != StatusOK || r.Exit != nil || r.ProcessCode() != 0 || r.Session != "x" {
			t.Errorf("Copy = %+v, want ok, exit null", r)
		}
		if got := shellCalls(f); len(got) != 1 || got[0] != c.want {
			t.Errorf("calls = %q, want %q", got, c.want)
		}
	}
}

// A copy that fails names limactl's reason; a copy runs no guest command, so
// exit stays null and the status alone says so.
func TestCopyFailureIsFailed(t *testing.T) {
	f := &listLima{guest: func(_ context.Context, _ io.Reader, _, errw io.Writer, _ []string) error {
		_, _ = io.WriteString(errw, "scp: /tmp/nope: No such file or directory\n")
		return guestExit(1)
	}}
	var plain strings.Builder
	r := heldSession(t, f).Copy(context.Background(), "x:/tmp/nope", "./f", &plain)
	if r.Status != StatusFailed || r.Exit != nil || r.ProcessCode() != 1 {
		t.Errorf("Copy = %+v, want failed, exit null", r)
	}
	if !strings.Contains(r.Stderr, "No such file") || !strings.Contains(plain.String(), "No such file") {
		t.Errorf("record %q, plain %q, want limactl's reason in both", r.Stderr, plain.String())
	}
}

// AC9, AC11: a name no guest serves never reaches limactl copy.
func TestCopyRefusesBeforeTheGuest(t *testing.T) {
	boom := func(context.Context, io.Reader, io.Writer, io.Writer, []string) error {
		t.Error("a refusal reached the guest")
		return errors.New("must not run")
	}
	r := newSessions(t, &listLima{instances: map[string]string{}, guest: boom}).Copy(context.Background(), "./f", "ghost:/tmp/f", io.Discard)
	if r.Status != StatusNoSession || r.ProcessCode() != 3 {
		t.Errorf("Copy = %+v, want no-session, exit 3", r)
	}
	f := &listLima{guest: boom}
	s := heldSession(t, f)
	f.instances["dotf-x"] = "Stopped"
	if r := s.Copy(context.Background(), "./f", "x:/tmp/f", io.Discard); r.Status != StatusStale || r.ProcessCode() != 2 {
		t.Errorf("Copy = %+v, want stale, exit 2", r)
	}
}
