package converge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/agy"
	"github.com/mlorentedev/dotfiles/cli/internal/claude"
)

// claudeBox is a Claude Code install: the servers and plugins it has, and the
// marketplaces it registers. An install in broken fails; one in deaf exits 0
// and leaves nothing installed. A server in stale is registered at an old URL
// until it is removed. mutated counts every call that changes it.
type claudeBox struct {
	servers, plugins, markets []string
	broken, deaf              map[string]bool
	stale                     map[string]string
	mutated                   int
}

func (b *claudeBox) List() (string, error) { return strings.Join(b.plugins, "\n"), nil }

func (b *claudeBox) Install(id string) error {
	b.mutated++
	switch {
	case b.broken[id]:
		return errors.New("marketplace unreachable")
	case !b.deaf[id]:
		b.plugins = append(b.plugins, id)
	}
	return nil
}

func (b *claudeBox) McpGet(name string) (string, error) {
	if !slices.Contains(b.servers, name) {
		return "", errors.New("No MCP server found")
	}
	url := "https://example.test/mcp"
	if old, ok := b.stale[name]; ok {
		url = old
	}
	return name + ":\n  Scope: User config (available in all your projects)\n  Type: http\n  URL: " + url + "\n", nil
}

func (b *claudeBox) McpAdd(name, _ string, _ []string) error {
	b.mutated++
	b.servers = append(b.servers, name)
	return nil
}

func (b *claudeBox) McpRemove(name string) error {
	b.mutated++
	delete(b.stale, name)
	b.servers = slices.DeleteFunc(b.servers, func(s string) bool { return s == name })
	return nil
}

func (b *claudeBox) Marketplaces() (string, error) {
	parts := make([]string, 0, len(b.markets))
	for _, m := range b.markets {
		parts = append(parts, `{"name":"`+m+`"}`)
	}
	return "[" + strings.Join(parts, ",") + "]", nil
}

func (b *claudeBox) RemoveMarketplace(string) error { b.mutated++; return nil }

func (b *claudeBox) AddMarketplace(source string) error {
	b.mutated++
	b.markets = append(b.markets, source[strings.LastIndex(source, "/")+1:])
	return nil
}

func (b *claudeBox) UpdateMarketplace(string) error { b.mutated++; return nil }

// extensionsRepo declares one server and two plugins from one marketplace.
func extensionsRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, claude.ServersRel),
		`{"servers":[{"name":"web","transport":"http","args":"https://example.test/mcp"}]}`)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(claude.PluginsRel)),
		`{"plugins":["a@m","b@m"],"retired_marketplaces":[]}`)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(claude.MarketplacesRel)),
		`{"marketplaces":{"m":"o/m"}}`)
	return repo
}

func withClaude(box *claudeBox) agentExtensions {
	return agentExtensions{claude: func(string) (claude.Syncer, bool) {
		if box == nil {
			return claude.Syncer{}, false
		}
		return claude.Syncer{
			Run:        box,
			ClaudeJSON: filepath.Join(os.TempDir(), "absent-claude-json-for-test"),
			Floor:      10240,
			HasCommand: func(string) bool { return true },
		}, true
	}}
}

// A machine without claude skips the step and says why; it never passes it.
func TestAgentExtensions_SkipsWithoutClaude(t *testing.T) {
	for name, step := range map[string]agentExtensions{"unwired": {}, "absent": withClaude(nil)} {
		res, err := step.Reconcile(Env{RepoRoot: extensionsRepo(t)}, false)
		if err != nil || res.Skip == "" {
			t.Errorf("%s: res %+v, err %v; want a skip", name, res, err)
		}
	}
}

// A fresh Claude Code gets the declared server and plugins, the probe then
// passes, and a second run changes nothing: what a Mac from zero lacked (#2387).
func TestAgentExtensions_AFreshClaudeGetsServersAndPlugins(t *testing.T) {
	box := &claudeBox{}
	step, env := withClaude(box), Env{RepoRoot: extensionsRepo(t)}
	res, err := step.Reconcile(env, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changes != 3 || !strings.Contains(res.Detail, "1 MCP server(s) added") || !strings.Contains(res.Detail, "2 plugin(s) added") {
		t.Errorf("first run: %+v", res)
	}
	if err := step.Probe(env); err != nil {
		t.Errorf("probe after the apply: %v", err)
	}
	again, err := step.Reconcile(env, false)
	if err != nil || again.Changes != 0 {
		t.Errorf("second run: %+v, %v; want no change", again, err)
	}
}

// A plan reports what an apply would add and changes nothing.
func TestAgentExtensions_APlanChangesNothing(t *testing.T) {
	box := &claudeBox{}
	res, err := withClaude(box).Reconcile(Env{RepoRoot: extensionsRepo(t)}, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changes != 3 || !strings.Contains(res.Detail, "to add") || box.mutated != 0 {
		t.Errorf("plan: %+v, %d mutating call(s)", res, box.mutated)
	}
}

// A server registered with arguments the SSOT no longer declares is planned as
// a re-registration, replaced on apply, and in sync on the next run (#2397).
func TestAgentExtensions_ADriftedServerIsReRegistered(t *testing.T) {
	box := &claudeBox{servers: []string{"web"}, plugins: []string{"a@m", "b@m"}, markets: []string{"m"},
		stale: map[string]string{"web": "https://old.test/mcp"}}
	step, env := withClaude(box), Env{RepoRoot: extensionsRepo(t)}
	const move = "web %s (http https://old.test/mcp -> http https://example.test/mcp)"
	plan, err := step.Reconcile(env, true)
	if err != nil || plan.Changes != 1 || !strings.Contains(plan.Detail, fmt.Sprintf(move, "to re-register")) || box.mutated != 0 {
		t.Fatalf("plan: %+v, %v, %d mutating call(s)", plan, err, box.mutated)
	}
	res, err := step.Reconcile(env, false)
	if err != nil || res.Changes != 1 || !strings.Contains(res.Detail, fmt.Sprintf(move, "re-registered")) {
		t.Fatalf("apply: %+v, %v", res, err)
	}
	again, err := step.Reconcile(env, false)
	if err != nil || again.Changes != 0 {
		t.Errorf("second run: %+v, %v; want no change", again, err)
	}
}

// A plugin that fails to install fails the step, and the error names it with
// the CLI's cause: converge prints one line per step (BUG-121, #2336).
func TestAgentExtensions_AFailedInstallFailsWithItsCause(t *testing.T) {
	box := &claudeBox{broken: map[string]bool{"b@m": true}}
	_, err := withClaude(box).Reconcile(Env{RepoRoot: extensionsRepo(t)}, false)
	if err == nil || !strings.Contains(err.Error(), "b@m (marketplace unreachable)") {
		t.Errorf("err = %v; want b@m named with its cause", err)
	}
}

// An install that exits 0 and installs nothing is caught by the probe, by name.
func TestAgentExtensions_TheProbeNamesWhatIsStillMissing(t *testing.T) {
	box := &claudeBox{deaf: map[string]bool{"b@m": true}}
	step, env := withClaude(box), Env{RepoRoot: extensionsRepo(t)}
	if _, err := step.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if err := step.Probe(env); err == nil || !strings.Contains(err.Error(), "b@m") {
		t.Errorf("probe = %v; want it to name b@m", err)
	}
}

// withAgy writes agy's source into repo and wires paths under home.
func withAgy(t *testing.T, step agentExtensions, repo, home string) agentExtensions {
	t.Helper()
	writeFile(t, filepath.Join(repo, filepath.FromSlash(agy.MCPRel)),
		`{"mcpServers":{"hive-vault":{"command":"uvx","args":["hive-vault"],"env":{"VAULT_PATH":"${VAULT_PATH}"}}}}`)
	step.agyPaths = func(h string) agy.Paths {
		return agy.ResolvePaths(h, func(string) string { return "" }, func(string) string { return "" })
	}
	return step
}

// agy's configs are written whether or not claude is installed, and a machine
// without claude no longer skips the whole step (CLI-105, #2384).
func TestAgentExtensions_AgyIsWrittenWithoutClaude(t *testing.T) {
	repo, home := extensionsRepo(t), t.TempDir()
	step, env := withAgy(t, withClaude(nil), repo, home), Env{RepoRoot: repo, Home: home}
	res, err := step.Reconcile(env, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skip != "" || res.Changes != 3 || !strings.Contains(res.Detail, "agy: 3 MCP config(s) written") ||
		!strings.Contains(res.Detail, "claude is not installed") {
		t.Errorf("first run: %+v", res)
	}
	if err := step.Probe(env); err != nil {
		t.Errorf("probe after the apply: %v", err)
	}
	again, err := step.Reconcile(env, false)
	if err != nil || again.Changes != 0 || !strings.Contains(again.Detail, "3 in sync") {
		t.Errorf("second run: %+v, %v; want no change", again, err)
	}
}

// A master agy rewrote by hand is caught by the probe, by path.
func TestAgentExtensions_TheProbeNamesADriftedAgyConfig(t *testing.T) {
	repo, home := extensionsRepo(t), t.TempDir()
	step, env := withAgy(t, withClaude(&claudeBox{}), repo, home), Env{RepoRoot: repo, Home: home}
	if _, err := step.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	master := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	writeFile(t, master, "{}")
	if err := step.Probe(env); err == nil || !strings.Contains(err.Error(), master) {
		t.Errorf("probe = %v; want it to name %s", err, master)
	}
}

// herdrRepo is extensionsRepo plus the shipped routing map and its schema,
// which name the harnesses herdr's integrations go to.
func herdrRepo(t *testing.T) string {
	t.Helper()
	repo := extensionsRepo(t)
	for _, rel := range []string{"harness/model-map.json", "harness/model-map.schema.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(repo, filepath.FromSlash(rel)), string(data))
	}
	return repo
}

// herdrBox is herdr's integration state: every target it knows, and whether
// each is current. installs counts every install it ran.
type herdrBox struct {
	current  map[string]bool
	installs int
}

func (b *herdrBox) run(name string, args ...string) ([]byte, error) {
	if name != "herdr" || len(args) < 2 {
		return nil, fmt.Errorf("unexpected: %s %v", name, args)
	}
	if args[1] == "install" {
		b.installs++
		b.current[args[2]] = true
		return nil, nil
	}
	var out strings.Builder
	for target, ok := range b.current {
		state := "not installed"
		if ok {
			state = "current (v1)"
		}
		fmt.Fprintf(&out, "%s: %s (/h/%s)\n", target, state, target)
	}
	return []byte(out.String()), nil
}

// herdr's integration goes to every installed harness the routing map names,
// the probe then passes, and a second run installs nothing. A harness that is
// not installed gets none: herdr would create its config directory (#1312).
func TestAgentExtensions_HerdrIntegratesEveryInstalledHarness(t *testing.T) {
	box := &herdrBox{current: map[string]bool{
		"claude": false, "copilot": false, "opencode": false, "pi": true, "antigravity-cli": false,
	}}
	installed := map[string]bool{"herdr": true, "claude": true, "pi": true, "agy": true}
	step := agentExtensions{herdr: box.run, has: func(c string) bool { return installed[c] }}
	env := Env{RepoRoot: herdrRepo(t)}
	res, err := step.Reconcile(env, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changes != 2 || !strings.Contains(res.Detail, "herdr: 2 integration(s) installed, 1 current") {
		t.Errorf("first run: %+v", res)
	}
	if box.current["copilot"] || box.current["opencode"] {
		t.Errorf("an absent harness got an integration: %v", box.current)
	}
	if err := step.Probe(env); err != nil {
		t.Errorf("probe after the apply: %v", err)
	}
	box.installs = 0
	if again, err := step.Reconcile(env, false); err != nil || again.Changes != 0 || box.installs != 0 {
		t.Errorf("second run: %+v, %v, %d install(s); want none", again, err, box.installs)
	}
}

// Without herdr the half is skipped and named; the step is not failed.
func TestAgentExtensions_HerdrAbsentIsASkip(t *testing.T) {
	box := &herdrBox{current: map[string]bool{}}
	step := agentExtensions{herdr: box.run, has: func(string) bool { return false }}
	res, err := step.Reconcile(Env{RepoRoot: herdrRepo(t)}, false)
	if err != nil || !strings.Contains(res.Skip, "herdr is not installed") {
		t.Errorf("got %+v, %v; want a skip naming herdr", res, err)
	}
}
