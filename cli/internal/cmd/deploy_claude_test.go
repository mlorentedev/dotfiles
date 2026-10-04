package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeClaude answers `claude plugin list` with listed and fails the installs
// named in failing.
type fakeClaude struct {
	listed    string
	failing   map[string]bool
	installed []string
	onInstall func() // runs inside the guard, where the real CLI would write
}

func (f *fakeClaude) List() (string, error) { return f.listed, nil }

func (f *fakeClaude) Install(id string) error {
	f.installed = append(f.installed, id)
	if f.onInstall != nil {
		f.onInstall()
	}
	if f.failing[id] {
		return errors.New("marketplace unreachable")
	}
	return nil
}

func pluginRepo(t *testing.T) string {
	t.Helper()
	repo := twoConfigRepo(t)
	writeMirrorFixture(t, filepath.Join(repo, "ai", "claude", "plugins.json"),
		`{"plugins":["a@m","b@m"]}`)
	return repo
}

func TestDeployCmd_BareDeployInstallsMissingClaudePlugins(t *testing.T) {
	fake := &fakeClaude{listed: "a@m\n"}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, fake)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Join(fake.installed, ",") != "b@m" {
		t.Errorf("installed %v, want only the missing b@m", fake.installed)
	}
	if !strings.Contains(out, "claude-plugins 1 added, 1 already present") {
		t.Errorf("the step must report what it did:\n%s", out)
	}
}

// A second run on a converged box installs nothing: the deploy contract is
// that what is in sync is reported and not redone.
func TestDeployCmd_ClaudePluginsAreIdempotent(t *testing.T) {
	fake := &fakeClaude{listed: "a@m\nb@m\n"}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, fake)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(fake.installed) != 0 || !strings.Contains(out, "0 added, 2 already present") {
		t.Errorf("installed %v on a converged box:\n%s", fake.installed, out)
	}
}

func TestDeployCmd_AFailedPluginInstallIsNamedAndFailsTheRun(t *testing.T) {
	fake := &fakeClaude{failing: map[string]bool{"b@m": true}}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, fake)
	if err == nil {
		t.Fatalf("a failed install must fail the deploy:\n%s", out)
	}
	if !strings.Contains(out, "1 added") || !strings.Contains(out, "failed    claude-plugins b@m") {
		t.Errorf("the failed plugin must be named and not counted:\n%s", out)
	}
}

func TestDeployCmd_ClaudePluginsSkippedWithoutClaude(t *testing.T) {
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, nil)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "skipped   claude-plugins (claude not installed)") {
		t.Errorf("the skip must be reported:\n%s", out)
	}
}

// A named deploy repairs one config and leaves the converge steps alone.
func TestDeployCmd_NamedDeployDoesNotTouchClaudePlugins(t *testing.T) {
	fake := &fakeClaude{}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), []string{"one"}, fake)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(fake.installed) != 0 || strings.Contains(out, "claude-plugins") {
		t.Errorf("a named deploy ran the plugin step:\n%s", out)
	}
}

func TestDeployCmd_ClaudePluginsDryRunInstallsNothing(t *testing.T) {
	fake := &fakeClaude{}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), []string{"--dry-run"}, fake)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(fake.installed) != 0 || !strings.Contains(out, "2 would add") {
		t.Errorf("dry run installed %v:\n%s", fake.installed, out)
	}
}

// The guard must watch the .claude.json the CLI actually writes, which lives in
// CLAUDE_CONFIG_DIR. A guard pointed elsewhere finds no file, takes its
// unguarded branch, and every unit test still passes.
func TestDeployCmd_GuardWatchesTheClaudeConfigDir(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)
	claudeJSON := filepath.Join(cfg, ".claude.json")
	healthy := []byte(`{"oauthAccount":{},"pad":"` + strings.Repeat("a", 20000) + `"}`)
	if err := os.WriteFile(claudeJSON, healthy, 0o600); err != nil {
		t.Fatal(err)
	}
	fake := &fakeClaude{onInstall: func() { _ = os.WriteFile(claudeJSON, []byte(`{}`), 0o600) }}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, fake)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if got, _ := os.ReadFile(claudeJSON); string(got) != string(healthy) {
		t.Fatalf("%s was not restored after the install truncated it:\n%s", claudeJSON, out)
	}
	if !strings.Contains(out, "restored  claude-plugins "+claudeJSON) {
		t.Errorf("the restore must be reported with the path:\n%s", out)
	}
}

// The child claude runs with the CLAUDE_CONFIG_DIR the guard resolved, even
// when the parent's environment says otherwise; exec keeps the last duplicate.
func TestClaudeCLIPinsTheConfigDirOnTheChild(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/somewhere/else")
	env := claudeCLI{configDir: "/guarded"}.command("plugin", "list").Env
	var last string
	for _, kv := range env {
		if strings.HasPrefix(kv, "CLAUDE_CONFIG_DIR=") {
			last = kv
		}
	}
	if last != "CLAUDE_CONFIG_DIR=/guarded" {
		t.Fatalf("effective CLAUDE_CONFIG_DIR = %q, want the guarded dir", last)
	}
}
