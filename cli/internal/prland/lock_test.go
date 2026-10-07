package prland

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcquireLock_RefusesWhileHeldAndNamesThePID(t *testing.T) {
	dir := t.TempDir()
	release, err := AcquireLock(dir, "o/r", func(int) bool { return true }, new(bytes.Buffer))
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	_, err = AcquireLock(dir, "o/r", func(int) bool { return true }, new(bytes.Buffer))
	if err == nil {
		t.Fatal("a second lander took the lock while it was held")
	}
	if !errors.Is(err, ErrLandRunning) {
		t.Errorf("error %q is not ErrLandRunning", err)
	}
	if !strings.Contains(err.Error(), fmt.Sprint(os.Getpid())) {
		t.Errorf("error %q does not name the holder's PID %d", err, os.Getpid())
	}
}

func TestAcquireLock_LocksAreIndependentPerRepo(t *testing.T) {
	dir := t.TempDir()
	release, err := AcquireLock(dir, "o/one", func(int) bool { return true }, new(bytes.Buffer))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	other, err := AcquireLock(dir, "o/two", func(int) bool { return true }, new(bytes.Buffer))
	if err != nil {
		t.Fatalf("a lock on one repo blocked another: %v", err)
	}
	other()
}

// A holder that died leaves its PID file behind; the kernel freed the lock.
func TestAcquireLock_TakesOverAStaleLockWithANote(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "o+r.pid"), []byte("424242\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var note bytes.Buffer
	release, err := AcquireLock(dir, "o/r", func(pid int) bool { return pid != 424242 }, &note)
	if err != nil {
		t.Fatalf("a stale lock was not taken over: %v", err)
	}
	defer release()
	if !strings.Contains(note.String(), "424242") || !strings.Contains(note.String(), "taking over") {
		t.Errorf("the take-over is not visible: %q", note.String())
	}
	got, _ := os.ReadFile(filepath.Join(dir, "o+r.pid"))
	if strings.TrimSpace(string(got)) != fmt.Sprint(os.Getpid()) {
		t.Errorf("pid file holds %q after the take-over, want our PID", got)
	}
}

// A recorded PID that is alive but whose lock is free is not stale news: the
// kernel lock is the truth, the PID only names the holder.
func TestAcquireLock_ALivePIDWithAFreeLockIsNotATakeOver(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "o+r.pid"), []byte("424242\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var note bytes.Buffer
	release, err := AcquireLock(dir, "o/r", func(int) bool { return true }, &note)
	if err != nil {
		t.Fatal(err)
	}
	release()
	if note.Len() != 0 {
		t.Errorf("unexpected note: %q", note.String())
	}
}

func TestAcquireLock_ReleaseFreesTheLockAndRemovesThePIDFile(t *testing.T) {
	dir := t.TempDir()
	release, err := AcquireLock(dir, "o/r", func(int) bool { return true }, new(bytes.Buffer))
	if err != nil {
		t.Fatal(err)
	}
	release()
	release() // releasing twice is harmless: every exit path may call it

	if _, err := os.Stat(filepath.Join(dir, "o+r.pid")); !os.IsNotExist(err) {
		t.Errorf("pid file survived the release: %v", err)
	}
	again, err := AcquireLock(dir, "o/r", func(int) bool { return true }, new(bytes.Buffer))
	if err != nil {
		t.Fatalf("the lock was not released: %v", err)
	}
	again()
}

func TestRepoName_UsesTheOneGivenElseAsksGH(t *testing.T) {
	asked := false
	run := func(_ context.Context, args ...string) ([]byte, error) {
		asked = true
		if strings.Join(args, " ") != "repo view --json nameWithOwner --jq .nameWithOwner" {
			t.Errorf("unexpected gh call %v", args)
		}
		return []byte("mlorentedev/dotfiles\n"), nil
	}
	if got, err := RepoName(context.Background(), Options{Run: run, Repo: "o/r"}); err != nil || got != "o/r" || asked {
		t.Errorf("a given repo: %q, %v, asked gh %v", got, err, asked)
	}
	if got, err := RepoName(context.Background(), Options{Run: run}); err != nil || got != "mlorentedev/dotfiles" {
		t.Errorf("the current repo: %q, %v", got, err)
	}
	failing := func(context.Context, ...string) ([]byte, error) { return nil, errors.New("not a repository") }
	if _, err := RepoName(context.Background(), Options{Run: failing}); err == nil {
		t.Error("an unresolvable repo was accepted: nothing to key the lock on")
	}
}
