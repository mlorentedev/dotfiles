package filelock

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTryLockIsExclusiveUntilReleased(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")
	unlock, err := TryLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := TryLock(path); !errors.Is(err, ErrLocked) {
		t.Fatalf("second TryLock while held: got %v, want ErrLocked", err)
	}
	unlock()
	again, err := TryLock(path)
	if err != nil {
		t.Fatalf("TryLock after release: %v", err)
	}
	again()
}

func TestLockWaitsForTheHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")
	unlock, err := TryLock(path)
	if err != nil {
		t.Fatal(err)
	}
	time.AfterFunc(100*time.Millisecond, unlock)
	got, err := Lock(path, 5*time.Second)
	if err != nil {
		t.Fatalf("Lock did not wait for the holder: %v", err)
	}
	got()
}

func TestLockGivesUpAndNamesThePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")
	unlock, err := TryLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	_, err = Lock(path, 60*time.Millisecond)
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("got %v, want ErrLocked after the timeout", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the lock path %q", err, path)
	}
}
