package orca

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrOrcaRunning    = errors.New("Orca is currently running; quit it fully before applying settings")
	ErrDataNotFound   = errors.New("orca-data.json not found")
	ErrEmptyRepoRoot  = errors.New("repository root path cannot be empty")
)

// ProcessChecker reports whether an Orca instance is currently running.
type ProcessChecker func() bool

// DefaultProcessChecker checks for active orca-ide / AppImage / orca.exe processes.
func DefaultProcessChecker() bool {
	// On Linux/Darwin, check using pgrep
	cmd := exec.Command("pgrep", "-f", "orca-ide|orca-linux.*AppImage|orca\\.exe")
	if err := cmd.Run(); err == nil {
		return true
	}
	return false
}

// ExportReport captures what happened during settings export.
type ExportReport struct {
	KeybindingsCopied bool
	SettingsExported  bool
	SettingsCount     int
	RepoKeybindings   string
	RepoSettings      string
}

// SettingChange captures an individual setting mutation.
type SettingChange struct {
	Key string
	Old any
	New any
}

// TuneReport captures what happened during settings tuning.
type TuneReport struct {
	Changes    []SettingChange
	DryRun     bool
	BackupPath string
}

// DefaultDesiredSettings defines the baseline optimizations for Orca ADE.
var DefaultDesiredSettings = map[string]any{
	"experimentalAgentHibernation":        true,
	"agentHibernationIdleMs":               float64(600000),
	"refreshLocalBaseRefOnWorktreeCreate": true,
	"telemetry.optedIn":                   false,
}

// Export extracts keybindings.json and clean settings from orca-data.json into the repo.
func Export(repoRoot, orcaUserDataDir, orcaHomeDir string) (*ExportReport, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return nil, ErrEmptyRepoRoot
	}

	targetDir := filepath.Join(repoRoot, "ai", "orca")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating ai/orca target directory: %w", err)
	}

	report := &ExportReport{
		RepoKeybindings: filepath.Join(targetDir, "keybindings.json"),
		RepoSettings:    filepath.Join(targetDir, "settings.json"),
	}

	// 1. Export keybindings.json if it exists
	srcKeybindings := filepath.Join(orcaHomeDir, "keybindings.json")
	if b, err := os.ReadFile(srcKeybindings); err == nil && json.Valid(b) {
		var pretty map[string]any
		if err := json.Unmarshal(b, &pretty); err == nil {
			formatted, _ := json.MarshalIndent(pretty, "", "  ")
			if err := os.WriteFile(report.RepoKeybindings, append(formatted, '\n'), 0o644); err == nil {
				report.KeybindingsCopied = true
			}
		}
	}

	// 2. Export settings object from orca-data.json
	srcData := filepath.Join(orcaUserDataDir, "orca-data.json")
	dataBytes, err := os.ReadFile(srcData)
	if err == nil && json.Valid(dataBytes) {
		var fullData map[string]any
		if err := json.Unmarshal(dataBytes, &fullData); err == nil {
			if rawSettings, ok := fullData["settings"]; ok {
				if settingsMap, ok := rawSettings.(map[string]any); ok {
					formattedSettings, err := json.MarshalIndent(settingsMap, "", "  ")
					if err == nil {
						if err := os.WriteFile(report.RepoSettings, append(formattedSettings, '\n'), 0o644); err == nil {
							report.SettingsExported = true
							report.SettingsCount = len(settingsMap)
						}
					}
				}
			}
		}
	}

	return report, nil
}

// Tune applies the baseline desired tuning to orca-data.json.
func Tune(orcaUserDataDir string, dryRun bool, isRunning ProcessChecker) (*TuneReport, error) {
	dataPath := filepath.Join(orcaUserDataDir, "orca-data.json")
	dataBytes, err := os.ReadFile(dataPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrDataNotFound, dataPath)
		}
		return nil, fmt.Errorf("reading %s: %w", dataPath, err)
	}

	if !json.Valid(dataBytes) {
		return nil, fmt.Errorf("%s is not valid JSON", dataPath)
	}

	var fullData map[string]any
	if err := json.Unmarshal(dataBytes, &fullData); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", dataPath, err)
	}

	rawSettings, ok := fullData["settings"]
	var settingsMap map[string]any
	if ok && rawSettings != nil {
		settingsMap, _ = rawSettings.(map[string]any)
	}
	if settingsMap == nil {
		settingsMap = make(map[string]any)
		fullData["settings"] = settingsMap
	}

	var changes []SettingChange
	for key, desiredVal := range DefaultDesiredSettings {
		parts := strings.Split(key, ".")
		cur := getNestedValue(settingsMap, parts)
		if !valuesEqual(cur, desiredVal) {
			changes = append(changes, SettingChange{
				Key: key,
				Old: cur,
				New: desiredVal,
			})
			setNestedValue(settingsMap, parts, desiredVal)
		}
	}

	report := &TuneReport{
		Changes: changes,
		DryRun:  dryRun,
	}

	if len(changes) == 0 || dryRun {
		return report, nil
	}

	// Active process guard: prevent race conditions when Orca is running
	if isRunning != nil && isRunning() {
		return nil, ErrOrcaRunning
	}

	// Backup before write
	backupPath := fmt.Sprintf("%s.bak.%s", dataPath, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(backupPath, dataBytes, 0o600); err != nil {
		return nil, fmt.Errorf("creating backup at %s: %w", backupPath, err)
	}
	report.BackupPath = backupPath

	// Atomic write via temp file
	updatedBytes, err := json.MarshalIndent(fullData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling updated JSON: %w", err)
	}

	tmpPath := dataPath + ".tmp"
	if err := os.WriteFile(tmpPath, append(updatedBytes, '\n'), 0o600); err != nil {
		return nil, fmt.Errorf("writing temp file %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, dataPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("replacing %s: %w", dataPath, err)
	}

	return report, nil
}

func getNestedValue(root map[string]any, parts []string) any {
	cur := root
	for i, part := range parts {
		if i == len(parts)-1 {
			return cur[part]
		}
		next, ok := cur[part].(map[string]any)
		if !ok {
			return nil
		}
		cur = next
	}
	return nil
}

func setNestedValue(root map[string]any, parts []string, val any) {
	cur := root
	for i, part := range parts {
		if i == len(parts)-1 {
			cur[part] = val
			return
		}
		next, ok := cur[part].(map[string]any)
		if !ok {
			next = make(map[string]any)
			cur[part] = next
		}
		cur = next
	}
}

func valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
