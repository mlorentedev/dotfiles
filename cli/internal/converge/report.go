package converge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// reportFile is the persisted form of a run (#1843 B7): what each reconciler
// did and how the run ended, readable after the terminal is gone.
type reportFile struct {
	FinishedAt string        `json:"finished_at"`
	GOOS       string        `json:"goos"`
	Result     string        `json:"result"` // "ok" | "failed"
	Error      string        `json:"error,omitempty"`
	Changed    int           `json:"changed"` // changes across every reconciler
	Entries    []reportEntry `json:"entries"`
}

type reportEntry struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Changes int    `json:"changes"`
	Detail  string `json:"detail"`
}

// WriteReport persists an applied run to path, atomically, so a reader never
// sees a half-written report. A plan is never persisted: it changed nothing.
func WriteReport(path string, rep Report, runErr error) error {
	f := reportFile{FinishedAt: time.Now().UTC().Format(time.RFC3339), GOOS: rep.GOOS, Result: "ok"}
	if runErr != nil {
		f.Result, f.Error = "failed", runErr.Error()
	}
	for _, e := range rep.Entries {
		f.Changed += e.Changes
		f.Entries = append(f.Entries, reportEntry{Name: e.Name, Status: e.Status.String(), Changes: e.Changes, Detail: e.Detail})
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
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
