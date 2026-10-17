package update

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Every outcome is recorded, a failure with its error, and reads back as it
// was written.
func TestRecord_RoundTripsEveryOutcome(t *testing.T) {
	at := time.Date(2026, 10, 17, 3, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		out Outcome
		err error
	}{
		{Outcome{Status: "updated", Message: "self-update complete"}, nil},
		{Outcome{Status: "dirty", Message: "dirty worktree in /r — skipping"}, nil},
		{Outcome{Status: "converge-failed", Message: "converge failed"}, errors.New("converge: exit 1")},
	} {
		dir := t.TempDir()
		if err := WriteRecord(dir, NewRecord(at, tt.out, tt.err)); err != nil {
			t.Fatal(err)
		}
		got, err := ReadRecord(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !got.At.Equal(at) || got.Status != tt.out.Status || got.Message != tt.out.Message || got.Failed() != (tt.err != nil) {
			t.Errorf("%s: read back %+v", tt.out.Status, got)
		}
		if tt.err != nil && got.Error != tt.err.Error() {
			t.Errorf("%s: error %q, want %q", tt.out.Status, got.Error, tt.err)
		}
	}
}

func TestRecord_LivesUnderTheStateDir(t *testing.T) {
	if got, want := RecordPath("/s"), filepath.Join("/s", "update", "last.json"); got != want {
		t.Errorf("RecordPath = %q, want %q", got, want)
	}
}

// No run yet is not an error to hide: the reader is told the file is absent.
func TestReadRecord_AbsentIsNotExist(t *testing.T) {
	if _, err := ReadRecord(t.TempDir()); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, want fs.ErrNotExist", err)
	}
}

// A file that is not a record is an error, never a zero record that reads as
// a run with no status.
func TestReadRecord_RejectsAFileWithNoStatus(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "update"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(RecordPath(dir), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRecord(dir); err == nil {
		t.Error("a record with no status was accepted")
	}
}

// The skips that recur every night until someone acts are told apart from the
// ones that clear by themselves.
func TestRecord_StuckNamesTheSkipsThatNeedAHuman(t *testing.T) {
	for status, want := range map[string]bool{
		"dirty": true, "diverged": true, "not-a-repo": true, "no-upstream": true, "ff-failed": true,
		"offline": false, "current": false, "ahead": false, "updated": false, "reinstalled": false,
	} {
		if got := (Record{Status: status}).Stuck(); got != want {
			t.Errorf("%s: Stuck = %v, want %v", status, got, want)
		}
	}
}
