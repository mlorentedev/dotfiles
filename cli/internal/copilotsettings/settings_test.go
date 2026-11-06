package copilotsettings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
)

const retiredSettings = `{
  "telemetry": {"disabled": "SECRET-VALUE"},
  "model": "gpt-5.6-sol",
  "custom": {"keep": true}
}`

func TestInspectNamesRetiredKeysWithoutValues(t *testing.T) {
	repo, home, resolve, dst := copilotSettingsFixture(t, retiredSettings)

	got, err := Inspect(repo, home, resolve)
	if err != nil {
		t.Fatal(err)
	}
	if got.Settings != dst || got.Missing || got.Changed {
		t.Fatalf("Inspect() = %+v", got)
	}
	if !reflect.DeepEqual(got.Keys, []string{"telemetry.disabled"}) {
		t.Fatalf("keys = %v", got.Keys)
	}
	if strings.Contains(strings.Join(got.Keys, " "), "SECRET-VALUE") {
		t.Fatal("inspection exposed a setting value")
	}
}

func TestRepairRemovesOnlyRetiredLeavesAndIsIdempotent(t *testing.T) {
	repo, home, resolve, dst := copilotSettingsFixture(t, retiredSettings)

	first, err := Repair(repo, home, resolve, false)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || !reflect.DeepEqual(first.Keys, []string{"telemetry.disabled"}) {
		t.Fatalf("first Repair() = %+v", first)
	}
	got := readObject(t, dst)
	if _, ok := got["telemetry"]; ok {
		t.Fatalf("empty telemetry parent survived: %+v", got)
	}
	if got["model"] != "gpt-5.6-sol" || got["custom"].(map[string]any)["keep"] != true {
		t.Fatalf("unrelated settings changed: %+v", got)
	}
	if backup, err := os.ReadFile(dst + deploy.BackupSuffix); err != nil || string(backup) != retiredSettings {
		t.Fatalf("backup = %q, err=%v", backup, err)
	}

	second, err := Repair(repo, home, resolve, false)
	if err != nil || second.Changed || len(second.Keys) != 0 {
		t.Fatalf("second Repair() = %+v, err=%v", second, err)
	}
}

func TestRepairDryRunChangesNothing(t *testing.T) {
	repo, home, resolve, dst := copilotSettingsFixture(t, retiredSettings)

	got, err := Repair(repo, home, resolve, true)
	if err != nil || !got.Changed {
		t.Fatalf("dry-run Repair() = %+v, err=%v", got, err)
	}
	if raw, _ := os.ReadFile(dst); string(raw) != retiredSettings {
		t.Fatal("dry-run rewrote settings")
	}
	if _, err := os.Stat(dst + deploy.BackupSuffix); !os.IsNotExist(err) {
		t.Fatal("dry-run created a backup")
	}
}

func TestRepairPreservesSiblingsOfARetiredLeaf(t *testing.T) {
	settings := `{"telemetry":{"disabled":true,"keep":"yes"},"model":"gpt-5.6-sol"}`
	repo, home, resolve, dst := copilotSettingsFixture(t, settings)

	if _, err := Repair(repo, home, resolve, false); err != nil {
		t.Fatal(err)
	}
	got := readObject(t, dst)
	telemetry, ok := got["telemetry"].(map[string]any)
	if !ok || telemetry["keep"] != "yes" {
		t.Fatalf("telemetry sibling was not preserved: %+v", got)
	}
	if _, ok := telemetry["disabled"]; ok {
		t.Fatalf("retired leaf survived: %+v", got)
	}
}

func TestRepairMissingSettingsLeavesCreationToDeploy(t *testing.T) {
	repo, home, resolve, dst := copilotSettingsFixture(t, "")

	inspected, err := Inspect(repo, home, resolve)
	if err != nil || !inspected.Missing {
		t.Fatalf("Inspect() = %+v, err=%v", inspected, err)
	}
	repaired, err := Repair(repo, home, resolve, false)
	if err != nil || !repaired.Missing || repaired.Changed {
		t.Fatalf("Repair() = %+v, err=%v", repaired, err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("repair created a missing settings file")
	}
}

func TestRepairMalformedOrNonObjectSettingsFailsLoud(t *testing.T) {
	for _, body := range []string{`{"telemetry":`, `[{"telemetry":{"disabled":true}}]`} {
		t.Run(body, func(t *testing.T) {
			repo, home, resolve, dst := copilotSettingsFixture(t, body)
			if _, err := Repair(repo, home, resolve, false); err == nil {
				t.Fatal("Repair() accepted invalid settings")
			}
			if raw, _ := os.ReadFile(dst); string(raw) != body {
				t.Fatal("failed repair rewrote settings")
			}
			if _, err := os.Stat(dst + deploy.BackupSuffix); !os.IsNotExist(err) {
				t.Fatal("failed repair created a backup")
			}
		})
	}
}

func TestRepairUnreadableSettingsFailsLoud(t *testing.T) {
	repo, home, resolve, dst := copilotSettingsFixture(t, "")
	if err := os.MkdirAll(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Repair(repo, home, resolve, false); err == nil {
		t.Fatal("Repair() accepted a directory as settings.json")
	}
	if info, err := os.Stat(dst); err != nil || !info.IsDir() {
		t.Fatal("failed repair replaced the unreadable settings path")
	}
}

func TestRepairMalformedPolicyFailsWithoutTouchingSettings(t *testing.T) {
	repo, home, resolve, dst := copilotSettingsFixture(t, retiredSettings)
	writePolicy(t, repo, `{"version":`)

	if _, err := Repair(repo, home, resolve, false); err == nil {
		t.Fatal("Repair() accepted a malformed policy")
	}
	if raw, _ := os.ReadFile(dst); string(raw) != retiredSettings {
		t.Fatal("malformed policy caused a settings rewrite")
	}
}

func TestRepairConfigIgnoresOtherEntries(t *testing.T) {
	changed, err := RepairConfig(deploy.Config{Name: "other"}, t.TempDir(), t.TempDir(), func(string) string { return "" }, false)
	if err != nil || changed {
		t.Fatalf("RepairConfig(other) = changed %v, err %v", changed, err)
	}
}

func TestRepairConfigMissingPolicyKeepsOldCheckoutsCompatible(t *testing.T) {
	c := deploy.Config{Name: SettingsEntry, Dst: "{COPILOT_HOME}/settings.json", Mode: "0644"}
	changed, err := RepairConfig(c, t.TempDir(), t.TempDir(), func(string) string { return t.TempDir() }, false)
	if err != nil || changed {
		t.Fatalf("RepairConfig() = changed %v, err %v", changed, err)
	}
}

func copilotSettingsFixture(t *testing.T, settings string) (repo, home string, resolve func(string) string, dst string) {
	t.Helper()
	repo, home = t.TempDir(), t.TempDir()
	writePolicy(t, repo, `{"version":1,"retired_paths":["telemetry.disabled"]}`)
	writeFixture(t, repo, "ai/deploy.json", `{
		"version": 6,
		"configs": [{
			"name": "copilot-settings",
			"src": "ai/copilot/settings.json",
			"dst": "{COPILOT_HOME}/settings.json",
			"render": false,
			"mode": "0644",
			"strategy": "merge",
			"requires": "copilot"
		}]
	}`)
	writeFixture(t, repo, "ai/copilot/settings.json", `{"model":"gpt-5.6-sol"}`)
	copilotHome := filepath.Join(home, ".copilot")
	dst = filepath.Join(copilotHome, "settings.json")
	if settings != "" {
		writeFixture(t, home, ".copilot/settings.json", settings)
	}
	resolve = func(name string) string {
		if name == "COPILOT_HOME" {
			return copilotHome
		}
		return ""
	}
	return repo, home, resolve, dst
}

func writeFixture(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readObject(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
