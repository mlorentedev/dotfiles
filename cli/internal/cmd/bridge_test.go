package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
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
			Fingerprint:   "sha256:launch",
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
