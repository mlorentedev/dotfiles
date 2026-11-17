package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
)

func TestCheckCopilotSettingsReportsFileAndKeyWithoutValue(t *testing.T) {
	sys, dst := copilotSettingsDoctorBox(t, `{
		"telemetry": {"disabled": "SECRET-VALUE"},
		"model": "gpt-5.6-sol"
	}`)
	sys.CommandOutput = func(string, ...string) (string, error) {
		t.Fatal("Copilot settings validation must not execute copilot config --list")
		return "", nil
	}
	var buf bytes.Buffer

	checkCopilotSettings(sys, capture(&buf), false)

	out := buf.String()
	if !strings.Contains(out, "[FAIL] "+dst) || !strings.Contains(out, "telemetry.disabled") {
		t.Fatalf("doctor did not name file and key:\n%s", out)
	}
	if strings.Contains(out, "SECRET-VALUE") {
		t.Fatalf("doctor exposed a setting value:\n%s", out)
	}
}

func TestCheckCopilotSettingsFixBacksUpPreservesAndSecondRunPasses(t *testing.T) {
	settings := `{"telemetry":{"disabled":true},"model":"gpt-5.6-sol","keep":{"x":1}}`
	sys, dst := copilotSettingsDoctorBox(t, settings)
	var buf bytes.Buffer

	checkCopilotSettings(sys, capture(&buf), true)
	if out := buf.String(); !strings.Contains(out, "] removed telemetry.disabled") || strings.Contains(out, "[FAIL]") {
		t.Fatalf("doctor --fix did not report the repair:\n%s", out)
	}
	raw, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, keep := range []string{`"model": "gpt-5.6-sol"`, `"keep"`, `"x": 1`} {
		if !strings.Contains(got, keep) {
			t.Errorf("repair dropped %s:\n%s", keep, got)
		}
	}
	if strings.Contains(got, "telemetry") {
		t.Errorf("repair kept the empty retired parent:\n%s", got)
	}
	if backup, err := os.ReadFile(dst + deploy.BackupSuffix); err != nil || string(backup) != settings {
		t.Fatalf("backup = %q, err=%v", backup, err)
	}

	buf.Reset()
	checkCopilotSettings(sys, capture(&buf), true)
	if out := buf.String(); !strings.Contains(out, "[ OK ] no retired local Copilot settings") || strings.Contains(out, "removed") {
		t.Fatalf("second run was not idempotent:\n%s", out)
	}
}

func TestCheckCopilotSettingsMissingIsSkipped(t *testing.T) {
	sys, dst := copilotSettingsDoctorBox(t, "")
	var buf bytes.Buffer

	checkCopilotSettings(sys, capture(&buf), true)

	if out := buf.String(); !strings.Contains(out, "[SKIP] "+dst+" not deployed") {
		t.Fatalf("missing settings should skip:\n%s", out)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("doctor created missing settings")
	}
}

func TestCheckCopilotSettingsMissingPolicyFails(t *testing.T) {
	sys, _ := copilotSettingsDoctorBox(t, `{"model":"gpt-5.6-sol"}`)
	if err := os.Remove(filepath.Join(sys.Getenv("DOTFILES_REPO_DIR"), "ai", "copilot", "retired-settings.json")); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer

	checkCopilotSettings(sys, capture(&buf), false)

	if out := buf.String(); !strings.Contains(out, "[FAIL]") || !strings.Contains(out, "retired-settings policy") {
		t.Fatalf("missing policy should fail loud:\n%s", out)
	}
}

func TestCheckCopilotSettingsMalformedFailsWithoutRewrite(t *testing.T) {
	settings := `{"telemetry":`
	sys, dst := copilotSettingsDoctorBox(t, settings)
	var buf bytes.Buffer

	checkCopilotSettings(sys, capture(&buf), true)

	out := buf.String()
	if !strings.Contains(out, "[FAIL]") || !strings.Contains(out, "not a JSON object") {
		t.Fatalf("malformed settings did not fail loud:\n%s", out)
	}
	if raw, _ := os.ReadFile(dst); string(raw) != settings {
		t.Fatal("failed doctor repair rewrote malformed settings")
	}
	if _, err := os.Stat(dst + deploy.BackupSuffix); !os.IsNotExist(err) {
		t.Fatal("failed doctor repair created a backup")
	}
}

func copilotSettingsDoctorBox(t *testing.T, settings string) (*System, string) {
	t.Helper()
	repo, home := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "ai", "copilot", "retired-settings.json"),
		`{"version":1,"retired_paths":["telemetry.disabled"]}`)
	writeFile(t, filepath.Join(repo, "ai", "deploy.json"), `{
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
	copilotHome := filepath.Join(home, ".copilot")
	dst := filepath.Join(copilotHome, "settings.json")
	if settings != "" {
		writeFile(t, dst, settings)
	}
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo, "HOME": home}, nil, nil)
	sys.ResolvePath = func(name string) string {
		if name == "COPILOT_HOME" {
			return copilotHome
		}
		return ""
	}
	return sys, dst
}
