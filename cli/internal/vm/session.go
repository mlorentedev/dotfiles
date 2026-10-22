package vm

import "fmt"

// Status is what a verb reports about itself, never about the guest command it
// ran (ADR-055 decision 8): `ok` means the verb ran to completion.
type Status string

const (
	StatusOK          Status = "ok"
	StatusFailed      Status = "failed"      // dotf could not run the verb
	StatusBusy        Status = "busy"        // the session name is held
	StatusUnsupported Status = "unsupported" // the guest cannot do what was asked
	StatusNoSession   Status = "no-session"  // no such session
	StatusTimeout     Status = "timeout"
	// StatusStale is additive (ADR-055 decision 7 names no version bump for it):
	// a state record outlived its guest, and `down` clears it.
	StatusStale Status = "stale"
)

// Record is the one JSON object a `--json` call writes to stdout, whatever the
// outcome (ADR-055 decision 7). A consumer reads Status and Exit, never stderr
// and never the process code.
type Record struct {
	Version    int    `json:"version"`
	Session    string `json:"session"`
	OS         string `json:"os"`
	Status     Status `json:"status"`
	Exit       *int   `json:"exit"` // the guest command's code; nil for a verb that runs none
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Truncated  bool   `json:"truncated"`
	DurationMS int64  `json:"duration_ms"`
	StateDir   string `json:"state_dir,omitempty"` // up and down: where the session's state lives
}

// NewRecord is a record of version 1 with no exit code and no output.
func NewRecord(session, os string, s Status) Record {
	return Record{Version: 1, Session: session, OS: os, Status: s}
}

// ProcessCode is the code dotf exits with (ADR-055 decision 8): 0 when the verb
// is ok and the guest command exited 0 or ran none; 2 for a refusal before any
// work; 3 when no session could serve the call; 1 for everything else,
// including a timeout, so a composer does not retry it elsewhere.
func (r Record) ProcessCode() int {
	switch r.Status {
	case StatusOK:
		if r.Exit == nil || *r.Exit == 0 {
			return 0
		}
		return 1
	case StatusBusy, StatusUnsupported, StatusStale:
		return 2
	case StatusNoSession:
		return 3
	}
	return 1
}

// Summary is one line for the error a non-zero process code carries.
func (r Record) Summary() string {
	msg := r.Stderr
	if msg == "" {
		msg = string(r.Status)
	}
	return fmt.Sprintf("%s: %s", r.Session, msg)
}
