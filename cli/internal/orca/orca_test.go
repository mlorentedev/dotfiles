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

	sampleData := map[string]any{
		"schemaVersion": 1,
		"settings": map[string]any{
			"theme":                 "dark",
			"defaultTuiAgent":       "claude",
			"claudeManagedAccounts": []string{"acc-123"},
			"workspaceDirHistory":   []string{"/home/manu/old"},
			"telemetry": map[string]any{
				"optedIn":   false,
				"installId": "secret-uuid",
			},
		},
		"runtimeState": "ephemeral-data-not-to-export",
	}
	dataBytes, _ := json.Marshal(sampleData)
	_ = os.WriteFile(filepath.Join(orcaUserDir, "orca-data.json"), dataBytes, 0o644)

	rep, err := Export(repoRoot, orcaUserDir, orcaHomeDir)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if !rep.KeybindingsCopied || !rep.SettingsExported {
		t.Errorf("expected both keybindings and settings to be exported")
	}

	expSettingsBytes, err := os.ReadFile(rep.RepoSettings)
	if err != nil {
		t.Fatalf("failed reading exported settings: %v", err)
	}
	var exportedMap map[string]any
	_ = json.Unmarshal(expSettingsBytes, &exportedMap)

	// Sanitization assertions
	if _, exists := exportedMap["claudeManagedAccounts"]; exists {
		t.Errorf("expected claudeManagedAccounts to be sanitized from export")
	}
	if _, exists := exportedMap["workspaceDirHistory"]; exists {
		t.Errorf("expected workspaceDirHistory to be sanitized from export")
	}
	if tel, ok := exportedMap["telemetry"].(map[string]any); ok {
		if _, exists := tel["installId"]; exists {
			t.Errorf("expected telemetry.installId to be sanitized")
		}
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
				"optedIn": "false", // string "false" should not equal boolean false
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

	if len(rep.Changes) == 0 || rep.BackupPath == "" {
		t.Errorf("expected changes and backup path")
	}
	if _, err := os.Stat(rep.BackupPath); err != nil {
		t.Errorf("backup file was not created on disk: %v", err)
	}

	updatedBytes, _ := os.ReadFile(dataPath)
	var updatedData map[string]any
	_ = json.Unmarshal(updatedBytes, &updatedData)
	settings := updatedData["settings"].(map[string]any)

	if settings["experimentalAgentHibernation"] != true {
		t.Errorf("expected experimentalAgentHibernation=true, got %v", settings["experimentalAgentHibernation"])
	}

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
