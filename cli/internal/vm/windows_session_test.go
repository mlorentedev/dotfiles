package vm

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/filelock"
)

func windowsStore(t *testing.T, g *fakeGuest) WindowsSessions {
	t.Helper()
	o := sealedBase(t)
	return WindowsSessions{
		H:       g,
		Machine: o.Machine,
		Agent:   o.Agent,
		Poll:    o.Poll,
		Now:     func() time.Time { return time.Unix(1700000000, 0) },
	}
}

func holdName(t *testing.T, s WindowsSessions, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(s.Machine.Layout.Sessions(), name), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestWindowsUpHoldsTheGuestAndNamesIt(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	s := windowsStore(t, g)
	rec := s.Up(context.Background(), "", io.Discard, io.Discard)
	if rec.Status != StatusOK || rec.Session != "windows-1700000000" || rec.OS != "windows" {
		t.Fatalf("Up = %+v", rec)
	}
	if _, err := os.Stat(rec.StateDir); err != nil {
		t.Errorf("no state record: %v", err)
	}
	if _, err := os.Stat(s.Machine.Layout.Disk()); err != nil {
		t.Errorf("the overlay must outlive up: %v", err)
	}
	if g.procs[0].killed {
		t.Error("up must leave the guest running")
	}
	list, err := s.List(context.Background())
	if err != nil || len(list) != 1 || list[0].Status != StatusOK || list[0].Session != rec.Session {
		t.Errorf("List = %+v, %v", list, err)
	}
}

func TestWindowsUpRefusesASecondSession(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	s := windowsStore(t, g)
	s.Up(context.Background(), "first", io.Discard, io.Discard)
	rec := s.Up(context.Background(), "second", io.Discard, io.Discard)
	if rec.Status != StatusBusy || rec.ProcessCode() != 2 || !strings.Contains(rec.Stderr, "first") {
		t.Errorf("Up = %+v, want busy naming the holder", rec)
	}
	if len(g.starts) != 1 {
		t.Errorf("a refused up started QEMU: %d starts", len(g.starts))
	}
}

// The host has one Windows guest, so two ups under any names race for it
// (#3105): the loser answers busy and never starts QEMU.
func TestWindowsUpWhileAnotherUpBootsIsBusy(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	s := windowsStore(t, g)
	release, err := filelock.TryLock(s.upLock())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	rec := s.Up(context.Background(), "second", io.Discard, io.Discard)
	if rec.Status != StatusBusy || rec.ProcessCode() != 2 || !strings.Contains(rec.Stderr, "another `up`") {
		t.Errorf("Up = %+v, want busy naming the other up", rec)
	}
	if len(g.starts) != 0 {
		t.Errorf("a refused up started QEMU: %d starts", len(g.starts))
	}
}

func TestWindowsUpRefusesAGuestNoSessionHolds(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	g.running = true
	s := windowsStore(t, g)
	rec := s.Up(context.Background(), "proof", io.Discard, io.Discard)
	if rec.Status != StatusBusy || !strings.Contains(rec.Stderr, "dotf vm windows stop") {
		t.Errorf("Up = %+v, want busy naming stop", rec)
	}
}

func TestWindowsUpOnAGoneGuestIsStale(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	s := windowsStore(t, g)
	holdName(t, s, "old")
	rec := s.Up(context.Background(), "proof", io.Discard, io.Discard)
	if rec.Status != StatusStale || rec.ProcessCode() != 2 || !strings.Contains(rec.Stderr, "down --session old") {
		t.Errorf("Up = %+v, want stale naming down", rec)
	}
	if len(g.starts) != 0 {
		t.Error("a stale up booted a guest")
	}
}

func TestWindowsUpNeedsACleanBase(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	s := windowsStore(t, g)
	if err := os.Remove(s.Machine.Layout.BaseDisk()); err != nil {
		t.Fatal(err)
	}
	rec := s.Up(context.Background(), "proof", io.Discard, io.Discard)
	if rec.Status != StatusFailed || !strings.Contains(rec.Stderr, "dotf vm windows build") {
		t.Errorf("Up = %+v", rec)
	}
	if name, _ := s.holder(); name != "" {
		t.Errorf("a failed up holds %q", name)
	}
}

func TestWindowsUpThatFailsLeavesNothingBehind(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	g.stuck = true // the clock check fails after the guest booted
	s := windowsStore(t, g)
	rec := s.Up(context.Background(), "proof", io.Discard, io.Discard)
	if rec.Status != StatusFailed {
		t.Fatalf("Up = %+v", rec)
	}
	if !g.procs[0].killed {
		t.Error("a failed up left QEMU running")
	}
	if name, _ := s.holder(); name != "" {
		t.Errorf("a failed up holds %q", name)
	}
	if _, err := os.Stat(s.Machine.Layout.Disk()); err == nil {
		t.Error("a failed up left its overlay")
	}
}

func TestWindowsDownStopsTheGuestAndKeepsCleanBase(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	s := windowsStore(t, g)
	s.Up(context.Background(), "proof", io.Discard, io.Discard)
	rec := s.Down(context.Background(), "proof", io.Discard)
	if rec.Status != StatusOK {
		t.Fatalf("Down = %+v", rec)
	}
	if g.quits == 0 {
		t.Error("down did not stop the guest")
	}
	l := s.Machine.Layout
	for _, gone := range []string{l.Disk(), l.Vars(), filepath.Join(l.Sessions(), "proof")} {
		if _, err := os.Stat(gone); err == nil {
			t.Errorf("%s outlived down", gone)
		}
	}
	if _, err := os.Stat(l.BaseDisk()); err != nil {
		t.Errorf("down touched clean-base: %v", err)
	}
}

func TestWindowsDownClearsAStaleSession(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	s := windowsStore(t, g)
	holdName(t, s, "old")
	if rec := s.Down(context.Background(), "old", io.Discard); rec.Status != StatusOK {
		t.Fatalf("Down = %+v", rec)
	}
	if name, _ := s.holder(); name != "" {
		t.Errorf("still held by %q", name)
	}
}

func TestWindowsDownOnAnUnknownNameIsNoSession(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	g.running = true // a kept run: down must not stop what no session holds
	s := windowsStore(t, g)
	rec := s.Down(context.Background(), "ghost", io.Discard)
	if rec.Status != StatusNoSession || rec.ProcessCode() != 3 {
		t.Errorf("Down = %+v", rec)
	}
	if g.quits != 0 {
		t.Error("down stopped a guest no session holds")
	}
}

func TestWindowsListIsEmptyNotNilAndReportsStale(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	s := windowsStore(t, g)
	list, err := s.List(context.Background())
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("List = %#v, %v", list, err)
	}
	holdName(t, s, "old")
	list, _ = s.List(context.Background())
	if len(list) != 1 || list[0].Status != StatusStale {
		t.Errorf("List = %+v, want one stale", list)
	}
}
