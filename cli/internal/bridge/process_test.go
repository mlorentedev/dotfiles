package bridge

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectProcessRequiresExactStartIdentity(t *testing.T) {
	record := ProcessRecord{
		PID:           42,
		Executable:    filepath.Join(t.TempDir(), "ts-bridge"),
		StartIdentity: "start:one",
		Fingerprint:   "sha256:launch",
	}
	observe := func(int) (processSnapshot, error) {
		return processSnapshot{
			Executable:    record.Executable,
			StartIdentity: "start:two",
		}, nil
	}

	got, err := inspectProcess(record, observe)
	if err != nil || got != (ProcessInspection{Present: true}) {
		t.Fatalf("inspectProcess() = %+v, %v; want present but unowned", got, err)
	}
}

func TestInspectProcessRequiresExactExecutable(t *testing.T) {
	record := ProcessRecord{
		PID:           42,
		Executable:    filepath.Join(t.TempDir(), "ts-bridge"),
		StartIdentity: "start:one",
		Fingerprint:   "sha256:launch",
	}
	observe := func(int) (processSnapshot, error) {
		return processSnapshot{
			Executable:    filepath.Join(t.TempDir(), "foreign"),
			StartIdentity: record.StartIdentity,
		}, nil
	}

	got, err := inspectProcess(record, observe)
	if err != nil || got != (ProcessInspection{Present: true}) {
		t.Fatalf("inspectProcess() = %+v, %v; want present but unowned", got, err)
	}
}

func TestInspectProcessDistinguishesMissingFromInspectionFailure(t *testing.T) {
	record := ProcessRecord{
		PID:           42,
		Executable:    filepath.Join(t.TempDir(), "ts-bridge"),
		StartIdentity: "start:one",
		Fingerprint:   "sha256:launch",
	}
	for _, tc := range []struct {
		name    string
		err     error
		want    ProcessInspection
		wantErr bool
	}{
		{"missing", errProcessNotFound, ProcessInspection{}, false},
		{"inspection failure", errors.New("access denied"), ProcessInspection{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := inspectProcess(record, func(int) (processSnapshot, error) {
				return processSnapshot{}, tc.err
			})
			if got != tc.want || (err != nil) != tc.wantErr {
				t.Fatalf("inspectProcess() = %+v, %v; want %+v, error=%v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}

func TestCaptureAndInspectCurrentProcess(t *testing.T) {
	record, err := CaptureProcess(os.Getpid(), "sha256:test")
	if err != nil {
		t.Fatal(err)
	}
	got, err := InspectProcess(record)
	if err != nil || got != (ProcessInspection{Present: true, Owned: true}) {
		t.Fatalf("InspectProcess() = %+v, %v; want exact owned process", got, err)
	}

	record.StartIdentity += "-reused"
	got, err = InspectProcess(record)
	if err != nil || got != (ProcessInspection{Present: true}) {
		t.Fatalf("InspectProcess(reused PID) = %+v, %v; want present but unowned", got, err)
	}
}
