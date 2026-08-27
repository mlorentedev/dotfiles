package orca

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestExport_ExtractsKeybindingsAndSettings(t *testing.T) {
	repoRoot := t.TempDir()
	orcaUserDir := t.TempDir()
	orcaHomeDir := t.TempDir()

	// 1. Seed keybindings.json
	sampleKeybindings := map[string]any{
		"version": 1,
		"platforms": map[string]any{
			"linux": map[string]any{
				"tab.nextSameType": []string{"Mod+Shift+BracketRight"},
			},
		},
	}
	kbBytes, _ := json.Marshal(sampleKeybindings)
	_ = os.WriteFile(filepath.Join(orcaHomeDir, "keybindings.json"), kbBytes, 0o644)

	// 2. Seed orca-data.json
	sampleData := map[string]any{
		"schemaVersion": 1,
		"settings": map[string]any{
			"theme":              "dark",
			"defaultTuiAgent":    "claude",
			"terminalFontSize":   14,
			"telemetry": map[string]any{
				"optedIn": false,
			},
		},
		"runtimeState": "ephemeral-data-not-to-export",
	}
	dataBytes, _ := json.Marshal(sampleData)
	_ = os.WriteFile(filepath.Join(orcaUserDir, "orca-data.json"), dataBytes, 0o644)

	// Run Export
	rep, err := Export(repoRoot, orcaUserDir, orcaHomeDir)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if !rep.KeybindingsCopied {
		t.Errorf("expected KeybindingsCopied to be true")
	}
	if !rep.SettingsExported {
		t.Errorf("expected SettingsExported to be true")
	}
	if rep.SettingsCount != 4 {
		t.Errorf("expected 4 settings, got %d", rep.SettingsCount)
	}

	// Verify exported settings file does not contain root ephemeral fields
	expSettingsBytes, err := os.ReadFile(rep.RepoSettings)
	if err != nil {
		t.Fatalf("failed reading exported settings: %v", err)
	}
	var exportedMap map[string]any
	_ = json.Unmarshal(expSettingsBytes, &exportedMap)
	if _, exists := exportedMap["runtimeState"]; exists {
		t.Errorf("expected runtimeState to be excluded from exported settings")
	}
	if exportedMap["defaultTuiAgent"] != "claude" {
		t.Errorf("expected defaultTuiAgent=claude, got %v", exportedMap["defaultTuiAgent"])
	}
}

func TestTune_DryRunDoesNotModify(t *testing.T) {
	orcaUserDir := t.TempDir()
	dataPath := filepath.Join(orcaUserDir, "orca-data.json")

	initialData := map[string]any{
		"settings": map[string]any{
			"experimentalAgentHibernation": false,
			"agentHibernationIdleMs":       1800000.0,
			"telemetry": map[string]any{
				"optedIn": true,
			},
		},
	}
	dataBytes, _ := json.Marshal(initialData)
	_ = os.WriteFile(dataPath, dataBytes, 0o644)

	rep, err := Tune(orcaUserDir, true, func() bool { return false })
	if err != nil {
		t.Fatalf("Tune dry-run failed: %v", err)
	}

	if len(rep.Changes) == 0 {
		t.Errorf("expected changes to be detected")
	}

	// Verify file was untouched
	afterBytes, _ := os.ReadFile(dataPath)
	if string(afterBytes) != string(dataBytes) {
		t.Errorf("dry-run mutated the file")
	}
}

func TestTune_AppliesBaselineAndCreatesBackup(t *testing.T) {
	orcaUserDir := t.TempDir()
	dataPath := filepath.Join(orcaUserDir, "orca-data.json")

	initialData := map[string]any{
		"settings": map[string]any{
			"experimentalAgentHibernation": false,
			"telemetry": map[string]any{
				"optedIn": true,
			},
		},
	}
	dataBytes, _ := json.Marshal(initialData)
	_ = os.WriteFile(dataPath, dataBytes, 0o644)

	rep, err := Tune(orcaUserDir, false, func() bool { return false })
	if err != nil {
		t.Fatalf("Tune apply failed: %v", err)
	}

	if len(rep.Changes) == 0 {
		t.Errorf("expected changes")
	}
	if rep.BackupPath == "" {
		t.Errorf("expected backup path to be populated")
	}
	if _, err := os.Stat(rep.BackupPath); err != nil {
		t.Errorf("backup file was not created on disk: %v", err)
	}

	// Verify new settings
	updatedBytes, _ := os.ReadFile(dataPath)
	var updatedData map[string]any
	_ = json.Unmarshal(updatedBytes, &updatedData)
	settings := updatedData["settings"].(map[string]any)

	if settings["experimentalAgentHibernation"] != true {
		t.Errorf("expected experimentalAgentHibernation=true, got %v", settings["experimentalAgentHibernation"])
	}

	// Idempotency: re-running produces 0 changes
	rep2, err := Tune(orcaUserDir, false, func() bool { return false })
	if err != nil {
		t.Fatalf("second Tune run failed: %v", err)
	}
	if len(rep2.Changes) != 0 {
		t.Errorf("expected 0 changes on second run, got %d", len(rep2.Changes))
	}
}

func TestTune_GuardsAgainstRunningProcess(t *testing.T) {
	orcaUserDir := t.TempDir()
	dataPath := filepath.Join(orcaUserDir, "orca-data.json")

	initialData := map[string]any{
		"settings": map[string]any{
			"experimentalAgentHibernation": false,
		},
	}
	dataBytes, _ := json.Marshal(initialData)
	_ = os.WriteFile(dataPath, dataBytes, 0o644)

	_, err := Tune(orcaUserDir, false, func() bool { return true })
	if !errors.Is(err, ErrOrcaRunning) {
		t.Errorf("expected ErrOrcaRunning, got %v", err)
	}
}
