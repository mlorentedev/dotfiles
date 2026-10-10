package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// fakeClaude answers `claude plugin list` with listed and fails the installs
// named in failing. For MCP, registered maps a server to its `mcp get` output
// (absent means get exits non-zero), and failing also names adds that fail.
type fakeClaude struct {
	listed     string
	failing    map[string]bool
	installed  []string
	onInstall  func() // runs inside the guard, where the real CLI would write
	registered map[string]string
	added      []string
	removed    []string
	calls      []string // step order: "mcp", "market" or "plugin", per call
	markets    []string // registered marketplaces; a removal drops one
	sticky     map[string]bool
	unmarketed []string // removals asked for
}

func (f *fakeClaude) Marketplaces() (string, error) {
	f.calls = append(f.calls, "market")
	parts := make([]string, 0, len(f.markets))
	for _, m := range f.markets {
		parts = append(parts, `{"name":"`+m+`"}`)
	}
	return "[" + strings.Join(parts, ",") + "]", nil
}

func (f *fakeClaude) RemoveMarketplace(name string) error {
	f.unmarketed = append(f.unmarketed, name)
	if !f.sticky[name] {
		f.markets = slices.DeleteFunc(f.markets, func(m string) bool { return m == name })
	}
	return nil
}

func (f *fakeClaude) McpGet(name string) (string, error) {
	f.calls = append(f.calls, "mcp")
	out, ok := f.registered[name]
	if !ok {
		return "", errors.New("No MCP server found")
	}
	return out, nil
}

func (f *fakeClaude) McpAdd(name, transport string, args []string) error {
	f.added = append(f.added, name)
	if f.failing[name] {
		return errors.New("add failed")
	}
	return nil
}

func (f *fakeClaude) McpRemove(name string) error {
	f.removed = append(f.removed, name)
	return nil
}

func (f *fakeClaude) List() (string, error) {
	f.calls = append(f.calls, "plugin")
	return f.listed, nil
}

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
		`{"plugins":["a@m","b@m"],"retired_marketplaces":["old"]}`)
	writeMirrorFixture(t, filepath.Join(repo, "mcp-servers.json"),
		`{"servers":[{"name":"web","transport":"http","args":"https://example.test/mcp"},`+
			`{"name":"hive","transport":"stdio","args":"hive client","prerequisite_binary":"uv","prerequisite_command":"uv tool install hive-vault"}]}`)
	return repo
}

// withPath answers the PATH lookups the MCP prerequisites make: only the named
// binaries are present.
func withPath(t *testing.T, have ...string) {
	t.Helper()
	orig := deployCommandAvailable
	t.Cleanup(func() { deployCommandAvailable = orig })
	deployCommandAvailable = func(name string) bool {
		for _, h := range have {
			if h == name {
				return true
			}
		}
		return false
	}
}

func TestDeployCmd_BareDeployRegistersMissingMCPServers(t *testing.T) {
	withPath(t, "uv")
	fake := &fakeClaude{registered: map[string]string{"web": "web:\n  Type: http\n"}, listed: "a@m b@m"}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, fake)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Join(fake.added, ",") != "hive" {
		t.Errorf("added %v, want only the missing hive", fake.added)
	}
	if !strings.Contains(out, row("installed", "claude-mcp")+"1 added, 1 already present") {
		t.Errorf("the step must report what it did:\n%s", out)
	}
	// Both twins register the servers before installing the plugins.
	if fake.calls[0] != "mcp" || fake.calls[len(fake.calls)-1] != "plugin" {
		t.Errorf("call order %v, want MCP before plugins", fake.calls)
	}
}

// A missing prerequisite is reported by name and is not a failure: the server
// cannot run, and installing its tool is not the deploy's job (#1993).
func TestDeployCmd_MCPServerWithoutItsPrerequisiteIsSkipped(t *testing.T) {
	withPath(t)
	fake := &fakeClaude{listed: "a@m b@m"}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, fake)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Join(fake.added, ",") != "web" || !strings.Contains(out, row("skipped", "claude-mcp")+"hive (uv not on PATH)") {
		t.Errorf("added %v; the skip must name the server and binary:\n%s", fake.added, out)
	}
}

// A failed add fails the run, and the plugin step still runs: one broken
// server must not leave the plugins unconverged.
func TestDeployCmd_AFailedMCPAddFailsTheRunButPluginsStillRun(t *testing.T) {
	withPath(t, "uv")
	fake := &fakeClaude{failing: map[string]bool{"web": true}, listed: "a@m"}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, fake)
	if err == nil || !strings.Contains(err.Error(), "web") {
		t.Fatalf("a failed add must fail the deploy and name the server: %v\n%s", err, out)
	}
	if !strings.Contains(out, row("failed", "claude-mcp")+"web did not register") {
		t.Errorf("the failed server must be named:\n%s", out)
	}
	if strings.Join(fake.installed, ",") != "b@m" {
		t.Errorf("plugins did not converge after the MCP failure: installed %v", fake.installed)
	}
}

func TestDeployCmd_StaleHiveRegistrationIsMigrated(t *testing.T) {
	withPath(t, "uv")
	fake := &fakeClaude{registered: map[string]string{"web": "ok", "hive": "Command: uvx hive-vault"}, listed: "a@m b@m"}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, fake)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Join(fake.removed, ",") != "hive" || strings.Join(fake.added, ",") != "hive" {
		t.Errorf("removed %v added %v, want hive removed then re-added", fake.removed, fake.added)
	}
	if !strings.Contains(out, row("migrated", "claude-mcp")+"hive") {
		t.Errorf("the migration must be reported:\n%s", out)
	}
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
	if !strings.Contains(out, row("installed", "claude-plugins")+"1 added, 1 already present") {
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
	if !strings.Contains(out, "1 added") || !strings.Contains(out, row("failed", "claude-plugins")+"b@m") {
		t.Errorf("the failed plugin must be named and not counted:\n%s", out)
	}
}

func TestDeployCmd_ClaudePluginsSkippedWithoutClaude(t *testing.T) {
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, nil)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, row("skipped", "claude-plugins")+"(claude not installed)") ||
		!strings.Contains(out, row("skipped", "claude-mcp")+"(claude not installed)") {
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
	if len(fake.installed) != 0 || len(fake.calls) != 0 || strings.Contains(out, "claude-") {
		t.Errorf("a named deploy ran a claude step:\n%s", out)
	}
}

func TestDeployCmd_ClaudePluginsDryRunInstallsNothing(t *testing.T) {
	withPath(t, "uv")
	fake := &fakeClaude{}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), []string{"--dry-run"}, fake)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(fake.installed) != 0 || !strings.Contains(out, row("would add", "claude-plugins")+"2 missing") {
		t.Errorf("dry run installed %v:\n%s", fake.installed, out)
	}
	if len(fake.added) != 0 || !strings.Contains(out, row("would add", "claude-mcp")+"2 missing") {
		t.Errorf("dry run registered %v:\n%s", fake.added, out)
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
	if !strings.Contains(out, row("restored", "claude-plugins")+claudeJSON) {
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

func TestMcpAddArgvMatchesTheTwins(t *testing.T) {
	got := strings.Join(mcpAddArgv("seq", "stdio", []string{"npx", "-y", "pkg"}), " ")
	if want := "mcp add --transport stdio seq --scope user -- npx -y pkg"; got != want {
		t.Fatalf("argv %q, want %q", got, want)
	}
}

// A retired marketplace still registered is removed, and the row says so.
func TestDeployCmd_RemovesARetiredMarketplace(t *testing.T) {
	withPath(t, "uv")
	fake := &fakeClaude{listed: "a@m b@m", markets: []string{"m", "old"}}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, fake)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Join(fake.unmarketed, ",") != "old" || !strings.Contains(out, row("removed", "claude-plugins")+"retired marketplace old") {
		t.Errorf("removed %v; the row must name it:\n%s", fake.unmarketed, out)
	}
}

// The #1431 class: the CLI exits 0 and the registration survives. The run
// fails and names it, and the plugin installs still run.
func TestDeployCmd_ARetiredMarketplaceThatSurvivesFailsTheRun(t *testing.T) {
	withPath(t, "uv")
	fake := &fakeClaude{listed: "a@m", markets: []string{"old"}, sticky: map[string]bool{"old": true}}
	out, err := runDeployWithClaude(t, pluginRepo(t), t.TempDir(), nil, fake)
	if err == nil || !strings.Contains(err.Error(), "old") {
		t.Fatalf("a surviving registration must fail the deploy: %v\n%s", err, out)
	}
	if !strings.Contains(out, row("failed", "claude-plugins")+"retired marketplace old is still registered") {
		t.Errorf("the failure must be named:\n%s", out)
	}
	if strings.Join(fake.installed, ",") != "b@m" {
		t.Errorf("plugins did not converge after the retire failure: installed %v", fake.installed)
	}
}
