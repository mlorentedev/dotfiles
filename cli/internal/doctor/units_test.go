package doctor

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.7.1", "1.6", 1},
		{"1.6", "1.7.1", -1},
		{"2.30.0", "2.30.0", 0},
		{"1.7", "1.7.0", 0}, // missing component treated as 0
		{"2.43.0", "2.30.0", 1},
		{"1.6", "1.6.0", 0},
		{"jq-1.7", "1.6", 1}, // leading non-digits stripped per component
		{"0.2.0", "0.2.0", 0},
	}
	for _, tt := range tests {
		if got := compareVersions(tt.a, tt.b); got != tt.want {
			t.Errorf("compareVersions(%q,%q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestAtLeast(t *testing.T) {
	if !atLeast("2.43.0", "2.30.0") {
		t.Error("2.43.0 should be >= 2.30.0")
	}
	if atLeast("1.5", "1.6") {
		t.Error("1.5 should NOT be >= 1.6")
	}
	if !atLeast("1.6", "1.6") {
		t.Error("1.6 should be >= 1.6 (equal)")
	}
}

func TestParseVersionsConf(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "versions.conf")
	content := "# comment\n\nGO_VERSION=1.26.0\n  YARN_VERSION = 1.22.22 \nDOTF_VERSION=0.2.0\nnot a kv line\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := parseVersionsConf(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["GO_VERSION"] != "1.26.0" {
		t.Errorf("GO_VERSION = %q, want 1.26.0", got["GO_VERSION"])
	}
	if got["YARN_VERSION"] != "1.22.22" {
		t.Errorf("YARN_VERSION = %q (whitespace not trimmed?), want 1.22.22", got["YARN_VERSION"])
	}
	if got["DOTF_VERSION"] != "0.2.0" {
		t.Errorf("DOTF_VERSION = %q, want 0.2.0", got["DOTF_VERSION"])
	}
}

func TestLoadContract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env-contract.json")
	json := `{
	  "env_vars": [
	    {"name":"DOTFILES_DIR","required":true,"default":{"linux":"$HOME/.dotfiles"},"validation":"path_exists"},
	    {"name":"USERPROFILE","required_on":"windows","validation":"path_exists"}
	  ],
	  "required_path_entries": {"linux":["$HOME/.local/bin"]},
	  "required_binaries": [{"name":"git","required":true,"min_version":"2.30.0","version_pattern":"git version ([0-9]+\\.[0-9]+\\.[0-9]+)"}],
	  "optional_binaries": [{"name":"gh","purpose":"GitHub CLI"}]
	}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := loadContract(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.EnvVars) != 2 || c.EnvVars[0].Name != "DOTFILES_DIR" || !c.EnvVars[0].Required {
		t.Fatalf("env_vars parsed wrong: %+v", c.EnvVars)
	}
	if c.EnvVars[1].RequiredOn != "windows" {
		t.Errorf("required_on not parsed: %+v", c.EnvVars[1])
	}
	if c.EnvVars[0].Default["linux"] != "$HOME/.dotfiles" {
		t.Errorf("default.linux not parsed: %+v", c.EnvVars[0].Default)
	}
	if len(c.RequiredBinaries) != 1 || c.RequiredBinaries[0].MinVersion != "2.30.0" {
		t.Errorf("required_binaries parsed wrong: %+v", c.RequiredBinaries)
	}
	if len(c.OptionalBinaries) != 1 || c.OptionalBinaries[0].Purpose != "GitHub CLI" {
		t.Errorf("optional_binaries parsed wrong: %+v", c.OptionalBinaries)
	}
}

func TestLoadContractInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env-contract.json")
	if err := os.WriteFile(path, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadContract(path); err == nil {
		t.Error("expected error parsing malformed JSON")
	}
}

func TestReportExitCode(t *testing.T) {
	r := NewReport(io.Discard, false)
	r.Section("s")
	r.Pass("a")
	r.Warn("b")
	r.Skip("c")
	if r.ExitCode() != 0 {
		t.Error("warn/skip must not drive a non-zero exit")
	}
	r.Fail("d")
	if r.ExitCode() != 1 {
		t.Error("a fail must drive exit 1")
	}
	if r.Failures() != 1 {
		t.Errorf("Failures() = %d, want 1", r.Failures())
	}
}

func TestReportSuppressesPassesUnlessVerbose(t *testing.T) {
	var quiet, loud bytes.Buffer
	q := NewReport(&quiet, false)
	q.Section("s")
	q.Pass("hidden pass")
	q.Summary()
	if strings.Contains(quiet.String(), "hidden pass") {
		t.Error("non-verbose report must not list passing checks")
	}
	if !strings.Contains(quiet.String(), "all ok") {
		t.Error("a fully-passing section must still print an all-ok summary line")
	}

	l := NewReport(&loud, true)
	l.Section("s")
	l.Pass("shown pass")
	l.Summary()
	if !strings.Contains(loud.String(), "shown pass") {
		t.Error("verbose report must list passing checks")
	}
}

func TestReport_Coloring(t *testing.T) {
	var buf bytes.Buffer
	r := NewReport(&buf, true)
	r.SetColor(true)
	r.Section("Color Section")
	r.Pass("pass msg")
	r.Fail("fail msg")
	r.Warn("warn msg")
	r.Skip("skip msg")
	r.Info("info msg")
	r.Fix("fix msg")
	r.Summary()

	out := buf.String()
	if !strings.Contains(out, "\033[32m[ OK ]\033[0m") {
		t.Errorf("pass tag should be colorized with green, got:\n%s", out)
	}
	if !strings.Contains(out, "\033[31m[FAIL]\033[0m") {
		t.Errorf("fail tag should be colorized with red, got:\n%s", out)
	}
	if !strings.Contains(out, "\033[33m[WARN]\033[0m") {
		t.Errorf("warn tag should be colorized with yellow, got:\n%s", out)
	}
	if !strings.Contains(out, "\033[36m[SKIP]\033[0m") {
		t.Errorf("skip tag should be colorized with cyan, got:\n%s", out)
	}
	if !strings.Contains(out, "\033[34m[INFO]\033[0m") {
		t.Errorf("info tag should be colorized with blue, got:\n%s", out)
	}
	if !strings.Contains(out, "\033[35m[FIX ]\033[0m") {
		t.Errorf("fix tag should be colorized with magenta, got:\n%s", out)
	}
	if !strings.Contains(out, "\033[1m[Color Section]\033[0m") {
		t.Errorf("section header should be bolded, got:\n%s", out)
	}
}

