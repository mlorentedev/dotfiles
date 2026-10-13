package doctor

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/claude"
)

// readOnlyClaude answers claude's read calls and fails the test on any call
// that would change Claude Code: doctor reports, it never registers.
type readOnlyClaude struct {
	t          *testing.T
	registered map[string]string // `claude mcp get` output by name; absent exits non-zero
	listed     string            // `claude plugin list` output
	markets    string            // `claude plugin marketplace list --json` output
}

func (c readOnlyClaude) List() (string, error)         { return c.listed, nil }
func (c readOnlyClaude) Marketplaces() (string, error) { return c.markets, nil }
func (c readOnlyClaude) McpGet(name string) (string, error) {
	if out, ok := c.registered[name]; ok {
		return out, nil
	}
	return "", errors.New("No MCP server found")
}
func (c readOnlyClaude) Install(id string) error { c.t.Fatalf("doctor installed %s", id); return nil }
func (c readOnlyClaude) McpAdd(n, _ string, _ []string) error {
	c.t.Fatalf("doctor added %s", n)
	return nil
}
func (c readOnlyClaude) McpRemove(n string) error { c.t.Fatalf("doctor removed %s", n); return nil }
func (c readOnlyClaude) RemoveMarketplace(n string) error {
	c.t.Fatalf("doctor unregistered %s", n)
	return nil
}
func (c readOnlyClaude) AddMarketplace(s string) error {
	c.t.Fatalf("doctor registered %s", s)
	return nil
}
func (c readOnlyClaude) UpdateMarketplace(n string) error {
	c.t.Fatalf("doctor refreshed %s", n)
	return nil
}

// claudeGet is `claude mcp get` as Claude Code prints it for a user-scope
// http server.
func claudeGet(name, url string) string {
	return name + ":\n  Scope: User config (available in all your projects)\n  Status: ✔ Connected\n  Type: http\n  URL: " + url + "\n"
}

// claudeRepo declares one http server and one plugin, and retires one
// marketplace.
func claudeRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, claude.ServersRel), `{"servers":[{"name":"web","transport":"http","args":"https://example.test/mcp"}]}`)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(claude.PluginsRel)), `{"plugins":["a@m"],"retired_marketplaces":["old"]}`)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(claude.MarketplacesRel)), `{"marketplaces":{"m":"o/m"}}`)
	return repo
}

func runClaudeCheck(t *testing.T, repo string, run claude.Runner) string {
	t.Helper()
	sys := newSys(map[string]string{"HOME": t.TempDir()}, nil, nil)
	var buf bytes.Buffer
	syncer := func(string) (claude.Syncer, bool) {
		return claude.Syncer{Run: run, ClaudeJSON: filepath.Join(t.TempDir(), "absent.json"), Floor: 10240, HasCommand: func(string) bool { return true }}, true
	}
	checkClaudeExtensions(sys, &Config{RepoDir: repo}, capture(&buf), syncer)
	return buf.String()
}

// A converged box passes; one missing a server, a plugin, or still carrying a
// retired marketplace, or registering a server with other arguments, warns
// with the converge command, and the read-only fake proves nothing changed.
func TestCheckClaudeExtensions(t *testing.T) {
	converged := readOnlyClaude{t: t, registered: map[string]string{"web": claudeGet("web", "https://example.test/mcp")},
		listed: "a@m\n", markets: `[{"name":"m"}]`}
	cases := []struct {
		name   string
		run    readOnlyClaude
		want   Status
		needle string
	}{
		{"converged → PASS", converged, StatusPass, "1 MCP server(s) and 1 plugin(s) as declared"},
		{"server missing → WARN", readOnlyClaude{t: t, listed: "a@m\n", markets: `[{"name":"m"}]`}, StatusWarn,
			"MCP server web is not registered (run `dotf converge --only agent-extensions`)"},
		{"server drifted → WARN with both", readOnlyClaude{t: t, registered: map[string]string{"web": claudeGet("web", "https://old.test/mcp")},
			listed: "a@m\n", markets: `[{"name":"m"}]`}, StatusWarn,
			"MCP server web is registered as http https://old.test/mcp, declared as http https://example.test/mcp (run `dotf converge"},
		{"server unreadable → WARN, converge leaves it", readOnlyClaude{t: t, registered: map[string]string{"web": "web:\n"},
			listed: "a@m\n", markets: `[{"name":"m"}]`}, StatusWarn, "converge leaves it as it is"},
		{"plugin missing → WARN", readOnlyClaude{t: t, registered: converged.registered, markets: `[{"name":"m"}]`}, StatusWarn,
			"plugin a@m is not installed (run `dotf converge --only agent-extensions`)"},
		{"retired marketplace still there → WARN", readOnlyClaude{t: t, registered: converged.registered, listed: "a@m\n",
			markets: `[{"name":"m"},{"name":"old"}]`}, StatusWarn, "retired marketplace old is still registered"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.run.t = t
			out := runClaudeCheck(t, claudeRepo(t), tc.run)
			if got := statusOfLine(out, tc.needle); got != tc.want {
				t.Fatalf("line mentioning %q: status %q, want %q\n%s", tc.needle, tagOf(got), tagOf(tc.want), out)
			}
			if tc.want == StatusWarn && strings.Contains(out, "as declared") {
				t.Fatalf("a finding must not also report a pass\n%s", out)
			}
		})
	}
}

// Without claude, outside a checkout, or unwired, the section skips.
func TestCheckClaudeExtensions_Skips(t *testing.T) {
	sys := newSys(map[string]string{"HOME": t.TempDir()}, nil, nil)
	absent := func(string) (claude.Syncer, bool) { return claude.Syncer{}, false }
	for name, c := range map[string]struct {
		repo   string
		syncer func(string) (claude.Syncer, bool)
		needle string
	}{
		"unwired":     {claudeRepo(t), nil, "no claude runner"},
		"no checkout": {"", absent, "not run from a dotfiles checkout"},
		"no claude":   {claudeRepo(t), absent, "claude is not installed"},
	} {
		var buf bytes.Buffer
		checkClaudeExtensions(sys, &Config{RepoDir: c.repo}, capture(&buf), c.syncer)
		if got := statusOfLine(buf.String(), c.needle); got != StatusSkip {
			t.Errorf("%s: status %q, want SKIP\n%s", name, tagOf(got), buf.String())
		}
	}
}
