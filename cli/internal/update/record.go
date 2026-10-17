package update

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Record is what the last `dotf update` did, kept so doctor can say whether
// the daily self-deploy still runs and still lands. The scheduler's own last
// result is a different surface on every OS (launchd's log, systemctl status,
// LastTaskResult) and says nothing of a skip, which exits 0.
type Record struct {
	At      time.Time `json:"at"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Error   string    `json:"error,omitempty"`
}

// NewRecord records out, the outcome Run returned at at, and err, its error.
func NewRecord(at time.Time, out Outcome, err error) Record {
	r := Record{At: at.UTC(), Status: out.Status, Message: out.Message}
	if err != nil {
		r.Error = err.Error()
	}
	return r
}

// Failed reports whether the run failed: the deploy broke, not a skip.
func (r Record) Failed() bool { return r.Error != "" }

// stuckStatuses are the skips that recur on every run until someone acts on
// the checkout. A dirty worktree is the one that kept a timer green while the
// deploy never ran (#694). offline clears by itself, and ahead is local work.
var stuckStatuses = []string{"dirty", "diverged", "not-a-repo", "no-upstream", "ff-failed"}

// Stuck reports whether the run skipped for a reason the next run will meet
// again.
func (r Record) Stuck() bool { return slices.Contains(stuckStatuses, r.Status) }

// RecordPath is where the record lives under dotf's state dir.
func RecordPath(stateDir string) string {
	return filepath.Join(stateDir, "update", "last.json")
}

// WriteRecord replaces the record, through a sibling temp file and a rename,
// so a reader never sees half of one.
func WriteRecord(stateDir string, r Record) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	path := RecordPath(stateDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".last-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op once renamed
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// ReadRecord reads the record. No run yet is fs.ErrNotExist; a file that is
// not a record is an error, never a zero record.
func ReadRecord(stateDir string) (Record, error) {
	data, err := os.ReadFile(RecordPath(stateDir))
	if err != nil {
		return Record{}, err
	}
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return Record{}, err
	}
	if r.Status == "" || r.At.IsZero() {
		return Record{}, errors.New(RecordPath(stateDir) + " is not an update record: no status or time")
	}
	return r, nil
}
