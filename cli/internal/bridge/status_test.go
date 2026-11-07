package bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifyStatus(t *testing.T) {
	cases := []struct {
		name string
		obs  Observation
		want State
	}{
		{"stopped", Observation{}, StateStopped},
		{"unowned", Observation{ProcessPresent: true}, StateUnowned},
		{"starting", Observation{ProcessPresent: true, ProcessOwned: true}, StateStarting},
		{"destination failed", Observation{ProcessPresent: true, ProcessOwned: true, ListenerReady: true}, StateDestinationFailed},
		{"healthy", Observation{ProcessPresent: true, ProcessOwned: true, ListenerReady: true, DestinationReady: true}, StateHealthy},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.obs); got != tc.want {
				t.Fatalf("Classify(%+v) = %q, want %q", tc.obs, got, tc.want)
			}
		})
	}
}

func TestStatusResultUsesAgentOrientedErrors(t *testing.T) {
	for _, tc := range []struct {
		state State
		want  string
	}{
		{StateStopped, "dotf bridge start --profile office"},
		{StateUnowned, "refusing to signal it"},
		{StateStarting, "listener is not ready"},
		{StateDestinationFailed, "private destination failed"},
	} {
		t.Run(string(tc.state), func(t *testing.T) {
			got := NewStatusResult("office", tc.state)
			if got.Error == "" || got.Why == "" || got.Fix == "" {
				t.Fatalf("result lacks error/why/fix: %+v", got)
			}
			if !strings.Contains(got.Error+" "+got.Why+" "+got.Fix, tc.want) {
				t.Fatalf("result %+v lacks %q", got, tc.want)
			}
		})
	}

	healthy := NewStatusResult("office", StateHealthy)
	if healthy.Error != "" || healthy.Why != "" || healthy.Fix != "" {
		t.Fatalf("healthy result carries an error envelope: %+v", healthy)
	}
}

func TestServiceStatusUsesOwnedProcessAndHealth(t *testing.T) {
	tests := []struct {
		name       string
		process    bool
		inspection ProcessInspection
		health     Health
		want       State
	}{
		{"stopped without process state", false, ProcessInspection{}, Health{}, StateStopped},
		{"stopped when recorded process is gone", true, ProcessInspection{}, Health{}, StateStopped},
		{"unowned", true, ProcessInspection{Present: true}, Health{}, StateUnowned},
		{"starting", true, ProcessInspection{Present: true, Owned: true}, Health{}, StateStarting},
		{"destination failed", true, ProcessInspection{Present: true, Owned: true}, Health{ListenerReady: true}, StateDestinationFailed},
		{"healthy", true, ProcessInspection{Present: true, Owned: true}, Health{ListenerReady: true, DestinationReady: true}, StateHealthy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(t.TempDir())
			launch := testDescriptor("office")
			if err := store.SaveLaunch(launch); err != nil {
				t.Fatal(err)
			}
			if tt.process {
				process := ProcessRecord{
					PID:           42,
					Executable:    filepath.Join(t.TempDir(), "ts-bridge"),
					StartIdentity: "platform:12345",
					Fingerprint:   "sha256:launch",
				}
				if err := store.SaveProcess("office", process); err != nil {
					t.Fatal(err)
				}
			}
			service := Service{
				Store: store,
				Inspect: func(ProcessRecord) (ProcessInspection, error) {
					return tt.inspection, nil
				},
				Probe: func(LaunchDescriptor) Health {
					return tt.health
				},
			}
			got, err := service.Status("office")
			if err != nil || got.State != tt.want {
				t.Fatalf("Status() = %+v, %v; want state %q", got, err, tt.want)
			}
		})
	}
}

func TestServiceStatusReturnsErrorForCorruptProcessState(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.SaveLaunch(testDescriptor("office")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root, "office", "process.json")
	if err := os.WriteFile(path, []byte(`{"pid":42}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := (Service{Store: store}).Status("office")
	if err == nil || got != (StatusResult{}) || !strings.Contains(err.Error(), "process executable") {
		t.Fatalf("Status() = %+v, %v; want unavailable corrupt-state error", got, err)
	}
}

func TestServiceStatusRequiresProcessInspector(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.SaveLaunch(testDescriptor("office")); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProcess("office", ProcessRecord{
		PID:           42,
		Executable:    filepath.Join(t.TempDir(), "ts-bridge"),
		StartIdentity: "platform:12345",
		Fingerprint:   "sha256:launch",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := (Service{Store: store}).Status("office")
	if err == nil || got != (StatusResult{}) || !strings.Contains(err.Error(), "inspector") {
		t.Fatalf("Status() = %+v, %v; want missing-inspector error", got, err)
	}
}

func TestServiceStatusRequiresHealthProbeForOwnedProcess(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.SaveLaunch(testDescriptor("office")); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProcess("office", ProcessRecord{
		PID:           42,
		Executable:    filepath.Join(t.TempDir(), "ts-bridge"),
		StartIdentity: "platform:12345",
		Fingerprint:   "sha256:launch",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := (Service{
		Store: store,
		Inspect: func(ProcessRecord) (ProcessInspection, error) {
			return ProcessInspection{Present: true, Owned: true}, nil
		},
	}).Status("office")
	if err == nil || got != (StatusResult{}) || !strings.Contains(err.Error(), "health probe") {
		t.Fatalf("Status() = %+v, %v; want missing-probe error", got, err)
	}
}
