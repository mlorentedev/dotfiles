package orca

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
)

var (
	ErrOrcaRunning   = errors.New("orca is currently running; quit it fully before applying settings")
	ErrDataNotFound  = errors.New("orca-data.json not found")
	ErrEmptyRepoRoot = errors.New("repository root path cannot be empty")
)

// ProcessChecker reports whether an Orca instance is currently running.
type ProcessChecker func() bool

// DefaultProcessChecker checks for active orca-ide / AppImage / orca.exe processes.
func DefaultProcessChecker() bool {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq orca.exe", "/NH")
		out, err := cmd.Output()
		return err == nil && strings.Contains(string(out), "orca.exe")
	}
	cmd := exec.Command("pgrep", "-f", "orca-ide|orca-linux.*AppImage")
	return cmd.Run() == nil
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

var volatileSettingsKeys = map[string]bool{
	"opencodeSessionCookie":                     true,
	"opencodeWorkspaceId":                         true,
	"claudeManagedAccounts":                       true,
	"activeClaudeManagedAccountId":                true,
	"activeClaudeManagedAccountIdsByRuntime":       true,
	"codexManagedAccounts":                        true,
	"activeCodexManagedAccountId":                 true,
	"activeCodexManagedAccountIdsByRuntime":       true,
	"workspaceDir":                                true,
	"workspaceDirHistory":                         true,
	"floatingTerminalCwd":                         true,
	"floatingTerminalTrustedCwds":                 true,
	"floatingTerminalCwdMigratedToAppWorkspace":   true,
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

	report.KeybindingsCopied = exportKeybindings(orcaHomeDir, report.RepoKeybindings)
	report.SettingsExported, report.SettingsCount = exportSettings(orcaUserDataDir, report.RepoSettings)
	return report, nil
}

func exportKeybindings(orcaHomeDir, destFile string) bool {
	src := filepath.Join(orcaHomeDir, "keybindings.json")
	b, err := os.ReadFile(src)
	if err != nil || !json.Valid(b) {
		return false
	}
	var pretty map[string]any
	if err := json.Unmarshal(b, &pretty); err != nil {
		return false
	}
	formatted, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		return false
	}
	return os.WriteFile(destFile, append(formatted, '\n'), 0o644) == nil
}

func exportSettings(orcaUserDataDir, destFile string) (bool, int) {
	src := filepath.Join(orcaUserDataDir, "orca-data.json")
	dataBytes, err := os.ReadFile(src)
	if err != nil || !json.Valid(dataBytes) {
		return false, 0
	}
	var fullData map[string]any
	if err := json.Unmarshal(dataBytes, &fullData); err != nil {
		return false, 0
	}
	rawSettings, ok := fullData["settings"].(map[string]any)
	if !ok {
		return false, 0
	}

	cleanSettings := make(map[string]any)
	for k, v := range rawSettings {
		if !volatileSettingsKeys[k] {
			cleanSettings[k] = v
		}
	}
	if tel, ok := cleanSettings["telemetry"].(map[string]any); ok {
		delete(tel, "installId")
	}

	formatted, err := json.MarshalIndent(cleanSettings, "", "  ")
	if err != nil {
		return false, 0
	}
	if err := os.WriteFile(destFile, append(formatted, '\n'), 0o644); err != nil {
		return false, 0
	}
	return true, len(cleanSettings)
}

// Tune applies the baseline desired tuning to orca-data.json.
func Tune(orcaUserDataDir string, dryRun bool, isRunning ProcessChecker) (*TuneReport, error) {
	dataPath := filepath.Join(orcaUserDataDir, "orca-data.json")
	fullData, settingsMap, dataBytes, err := loadAndValidateData(dataPath)
	if err != nil {
		return nil, err
	}

	changes := computeTuningChanges(settingsMap)
	report := &TuneReport{
		Changes: changes,
		DryRun:  dryRun,
	}

	if len(changes) == 0 || dryRun {
		return report, nil
	}

	if isRunning != nil && isRunning() {
		return nil, ErrOrcaRunning
	}

	backupPath, err := applyAtomicWrite(dataPath, dataBytes, fullData)
	if err != nil {
		return nil, err
	}
	report.BackupPath = backupPath
	return report, nil
}

func loadAndValidateData(dataPath string) (map[string]any, map[string]any, []byte, error) {
	dataBytes, err := os.ReadFile(dataPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil, fmt.Errorf("%w: %s", ErrDataNotFound, dataPath)
		}
		return nil, nil, nil, fmt.Errorf("reading %s: %w", dataPath, err)
	}
	if !json.Valid(dataBytes) {
		return nil, nil, nil, fmt.Errorf("%s is not valid JSON", dataPath)
	}
	var fullData map[string]any
	if err := json.Unmarshal(dataBytes, &fullData); err != nil {
		return nil, nil, nil, fmt.Errorf("parsing %s: %w", dataPath, err)
	}
	settingsMap, ok := fullData["settings"].(map[string]any)
	if !ok || settingsMap == nil {
		settingsMap = make(map[string]any)
		fullData["settings"] = settingsMap
	}
	return fullData, settingsMap, dataBytes, nil
}

func computeTuningChanges(settings map[string]any) []SettingChange {
	var changes []SettingChange
	for key, desiredVal := range DefaultDesiredSettings {
		parts := strings.Split(key, ".")
		cur := getNestedValue(settings, parts)
		if !valuesEqual(cur, desiredVal) {
			changes = append(changes, SettingChange{
				Key: key,
				Old: cur,
				New: desiredVal,
			})
			setNestedValue(settings, parts, desiredVal)
		}
	}
	return changes
}

func applyAtomicWrite(dataPath string, originalBytes []byte, data map[string]any) (string, error) {
	backupPath := fmt.Sprintf("%s.bak.%s", dataPath, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(backupPath, originalBytes, 0o600); err != nil {
		return "", fmt.Errorf("creating backup at %s: %w", backupPath, err)
	}
	updatedBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling updated JSON: %w", err)
	}
	tmpPath := dataPath + ".tmp"
	if err := os.WriteFile(tmpPath, append(updatedBytes, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("writing temp file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, dataPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("replacing %s: %w", dataPath, err)
	}
	return backupPath, nil
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
	return reflect.DeepEqual(a, b)
}
