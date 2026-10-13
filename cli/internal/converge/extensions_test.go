package converge

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/claude"
)

// claudeBox is a Claude Code install: the servers and plugins it has, and the
// marketplaces it registers. An install in broken fails; one in deaf exits 0
// and leaves nothing installed. mutated counts every call that changes it.
type claudeBox struct {
	servers, plugins, markets []string
	broken, deaf              map[string]bool
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
	return name + ": https://example.test/mcp", nil
}

func (b *claudeBox) McpAdd(name, _ string, _ []string) error {
	b.mutated++
	b.servers = append(b.servers, name)
	return nil
}

func (b *claudeBox) McpRemove(string) error { b.mutated++; return nil }

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
