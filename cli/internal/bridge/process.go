package bridge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var errProcessNotFound = errors.New("process not found")

type processSnapshot struct {
	Executable    string
	StartIdentity string
}

type processObserver func(int) (processSnapshot, error)

// ProcessInspection is what the platform process seam proved about a record.
type ProcessInspection struct {
	Present bool
	Owned   bool
}

// CaptureProcess records the live identity needed to reject stale or reused PIDs.
func CaptureProcess(pid int, fingerprint string) (ProcessRecord, error) {
	snapshot, err := observeProcess(pid)
	if err != nil {
		return ProcessRecord{}, fmt.Errorf("capture process %d: %w", pid, err)
	}
	record := ProcessRecord{
		PID:           pid,
		Executable:    snapshot.Executable,
		StartIdentity: snapshot.StartIdentity,
		Fingerprint:   fingerprint,
	}
	return record, record.Validate()
}

// InspectProcess proves whether a recorded process still names the same live process.
func InspectProcess(record ProcessRecord) (ProcessInspection, error) {
	return inspectProcess(record, observeProcess)
}

func inspectProcess(record ProcessRecord, observe processObserver) (ProcessInspection, error) {
	snapshot, err := observe(record.PID)
	if errors.Is(err, errProcessNotFound) {
		return ProcessInspection{}, nil
	}
	if err != nil {
		return ProcessInspection{}, fmt.Errorf("inspect process %d: %w", record.PID, err)
	}
	return ProcessInspection{
		Present: true,
		Owned: sameExecutable(record.Executable, snapshot.Executable) &&
			record.StartIdentity == snapshot.StartIdentity,
	}, nil
}

func sameExecutable(want, got string) bool {
	wantInfo, wantErr := os.Stat(want)
	gotInfo, gotErr := os.Stat(got)
	if wantErr == nil && gotErr == nil {
		return os.SameFile(wantInfo, gotInfo)
	}
	want = filepath.Clean(want)
	got = filepath.Clean(got)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(want, got)
	}
	return want == got
}
