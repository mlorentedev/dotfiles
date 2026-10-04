package cmd

import (
	"errors"
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
}

func (f *fakeClaude) List() (string, error) { return f.listed, nil }

func (f *fakeClaude) Install(id string) error {
	f.installed = append(f.installed, id)
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
