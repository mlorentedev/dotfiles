package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/bridge"
	"github.com/spf13/cobra"
)

func bridgeCommandFixture(t *testing.T, process bool, inspection bridge.ProcessInspection, health bridge.Health) (*cobra.Command, bridge.Store) {
	t.Helper()
	stateDir := t.TempDir()
	store := bridge.NewStore(stateDir)
	launch := bridge.LaunchDescriptor{
		Profile:    "office",
		Socks5:     "127.0.0.1:1081",
		HealthAddr: "127.0.0.1:9081",
	}
	if err := store.SaveLaunch(launch); err != nil {
		t.Fatal(err)
	}
	if process {
		if err := store.SaveProcess("office", bridge.ProcessRecord{
			PID:           42,
			Executable:    filepath.Join(t.TempDir(), "ts-bridge"),
			StartIdentity: "platform:12345",
			Fingerprint:   launch.Fingerprint(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	command := newBridgeCmdWithDeps(bridgeDeps{
		StateDir:    func() (string, error) { return stateDir, nil },
		RoleEnabled: func() (bool, error) { return true, nil },
		Inspect: func(bridge.ProcessRecord) (bridge.ProcessInspection, error) {
			return inspection, nil
		},
		Probe: func(bridge.LaunchDescriptor) bridge.Health { return health },
	})
	return command, store
}

func runBridgeCommand(t *testing.T, command *cobra.Command, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs(args)
	err := command.Execute()
	return stdout.String(), stderr.String(), err
}

func TestBridgeStatusCommandHealthy(t *testing.T) {
	command, _ := bridgeCommandFixture(t, true,
		bridge.ProcessInspection{Present: true, Owned: true},
		bridge.Health{ListenerReady: true, DestinationReady: true})

	stdout, stderr, err := runBridgeCommand(t, command, "status", "--profile", "office")
	if err != nil || stderr != "" || !strings.Contains(stdout, "bridge office: healthy") {
		t.Fatalf("status = stdout %q, stderr %q, err %v", stdout, stderr, err)
	}
}

func TestBridgeStatusCommandUnhealthyUsesErrorWhyFix(t *testing.T) {
	command, _ := bridgeCommandFixture(t, false, bridge.ProcessInspection{}, bridge.Health{})

	stdout, stderr, err := runBridgeCommand(t, command, "status", "--profile", "office")
	if err == nil || ExitCode(err) != 1 || stdout != "" {
		t.Fatalf("status = stdout %q, stderr %q, err %v", stdout, stderr, err)
	}
	for _, marker := range []string{"ERROR: bridge is stopped", "WHY:", "FIX:"} {
		if !strings.Contains(stderr, marker) {
			t.Errorf("stderr %q lacks %q", stderr, marker)
		}
	}
}

func TestBridgeStatusCommandJSONIsMachineReadableOnFailure(t *testing.T) {
	command, _ := bridgeCommandFixture(t, false, bridge.ProcessInspection{}, bridge.Health{})

	stdout, stderr, err := runBridgeCommand(t, command, "status", "--profile", "office", "--json")
	if err == nil || ExitCode(err) != 1 || stderr != "" {
		t.Fatalf("status = stdout %q, stderr %q, err %v", stdout, stderr, err)
	}
	var result bridge.StatusResult
	if jsonErr := json.Unmarshal([]byte(stdout), &result); jsonErr != nil || result.State != bridge.StateStopped {
		t.Fatalf("JSON status = %+v, %v; raw %q", result, jsonErr, stdout)
	}
}

// TestBridgeStatusCommandJSONCarriesUnavailableOnServiceError — a corrupt
// process record is a failure a JSON consumer must be able to read, not an
// empty stdout it cannot tell apart from a successful empty run.
func TestBridgeStatusCommandJSONCarriesUnavailableOnServiceError(t *testing.T) {
	command, store := bridgeCommandFixture(t, false, bridge.ProcessInspection{}, bridge.Health{})
	corrupt := filepath.Join(store.Root, "office", "process.json")
	if err := os.MkdirAll(filepath.Dir(corrupt), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corrupt, []byte(`{"pid":42}`), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runBridgeCommand(t, command, "status", "--profile", "office", "--json")
	if err == nil || ExitCode(err) != 1 || stderr != "" {
		t.Fatalf("status = stdout %q, stderr %q, err %v", stdout, stderr, err)
	}
	var result bridge.StatusResult
	if jsonErr := json.Unmarshal([]byte(stdout), &result); jsonErr != nil {
		t.Fatalf("JSON status is not parseable: %v; raw %q", jsonErr, stdout)
	}
	if result.State != bridge.StateUnavailable || result.Profile != "office" {
		t.Fatalf("JSON status = %+v; want state %q for profile office", result, bridge.StateUnavailable)
	}
	if result.Error == "" || result.Why == "" || result.Fix == "" {
		t.Fatalf("JSON status %+v lacks error/why/fix", result)
	}
}

// TestBridgeStatusCommandJSONReportsUnavailableForConfigurationFailures — the
// failures that happen before a profile is resolved lose the same fields.
func TestBridgeStatusCommandJSONReportsUnavailableForConfigurationFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		deps bridgeDeps
		want string
	}{
		{
			name: "role disabled",
			deps: bridgeDeps{
				StateDir:    func() (string, error) { return t.TempDir(), nil },
				RoleEnabled: func() (bool, error) { return false, nil },
			},
			want: "ts-bridge-client",
		},
		{
			name: "state dir failure",
			deps: bridgeDeps{
				StateDir:    func() (string, error) { return "", errors.New("no absolute home") },
				RoleEnabled: func() (bool, error) { return true, nil },
			},
			want: "no absolute home",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runBridgeCommand(t, newBridgeCmdWithDeps(tc.deps), "status", "--json")
			if err == nil || ExitCode(err) != 1 || stderr != "" {
				t.Fatalf("status = stdout %q, stderr %q, err %v", stdout, stderr, err)
			}
			var result bridge.StatusResult
			if jsonErr := json.Unmarshal([]byte(stdout), &result); jsonErr != nil || result.State != bridge.StateUnavailable {
				t.Fatalf("JSON status = %+v, %v; raw %q", result, jsonErr, stdout)
			}
			if !strings.Contains(result.Why+" "+result.Fix, tc.want) {
				t.Fatalf("JSON status %+v lacks %q", result, tc.want)
			}
		})
	}
}

func TestBridgeStatusCommandRequiresRole(t *testing.T) {
	command := newBridgeCmdWithDeps(bridgeDeps{
		StateDir:    func() (string, error) { return t.TempDir(), nil },
		RoleEnabled: func() (bool, error) { return false, nil },
	})
	_, stderr, err := runBridgeCommand(t, command, "status", "--profile", "office")
	if err == nil || !strings.Contains(stderr, "dotf env role add ts-bridge-client") {
		t.Fatalf("role refusal = %q, %v", stderr, err)
	}
}

func TestBridgeStatusCommandSurfacesStateDirFailure(t *testing.T) {
	command := newBridgeCmdWithDeps(bridgeDeps{
		StateDir:    func() (string, error) { return "", errors.New("no absolute home") },
		RoleEnabled: func() (bool, error) { return true, nil },
	})
	_, stderr, err := runBridgeCommand(t, command, "status")
	if err == nil || !strings.Contains(stderr, "no absolute home") {
		t.Fatalf("state refusal = %q, %v", stderr, err)
	}
}
