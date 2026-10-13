package agy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const agySource = `{"mcpServers": {"hive-vault": {"command": "uvx", "args": ["hive-vault"],
  "env": {"VAULT_PATH": "${VAULT_PATH}"}, "timeout": 30000, "trust": false}}}`

const rootServers = `{"servers": [
  {"name": "sequential-thinking", "transport": "stdio", "args": "npx -y @modelcontextprotocol/server-sequential-thinking"},
  {"name": "context7", "transport": "http", "args": "https://mcp.context7.com/mcp"},
  {"name": "hive", "transport": "stdio", "args": "hive client"},
  {"name": "solo", "transport": "stdio", "args": "solo"}
]}`

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixture is a checkout carrying both server lists and an empty home whose
// vault exists.
func fixture(t *testing.T) (repo string, p Paths) {
	t.Helper()
	repo, home := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, MCPRel), agySource)
	writeFile(t, filepath.Join(repo, "mcp-servers.json"), rootServers)
	p = ResolvePaths(home, func(string) string { return "" }, func(string) string { return "" })
	if err := os.MkdirAll(p.Vault, 0o755); err != nil {
		t.Fatal(err)
	}
	return repo, p
}

func servers(t *testing.T, path string) map[string]map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return doc.MCPServers
}

func TestResolvePaths_DefaultsFollowTheSetups(t *testing.T) {
	home := "/h"
	p := ResolvePaths(home, func(string) string { return "" }, func(string) string { return "" })
	want := Paths{
		GeminiHome: filepath.Join(home, ".gemini"),
		AppData:    filepath.Join(home, ".gemini", "antigravity-cli"),
		Vault:      filepath.Join(home, "Projects", "knowledge"),
	}
	if p != want {
		t.Fatalf("defaults = %+v, want %+v", p, want)
	}
	set := map[string]string{"GEMINI_HOME": "/g"}
	p = ResolvePaths(home, func(k string) string { return set[k] }, func(string) string { return "/v" })
	if p.AppData != filepath.Join("/g", "antigravity-cli") || p.Vault != "/v" {
		t.Fatalf("AGY_HOME must default under GEMINI_HOME and the contract must win: %+v", p)
	}
}

// AGY_HOME is the contract's name and wins; the old AGY_APP_DATA is read only
// when AGY_HOME is unset (#2395).
func TestDirs_ReadsTheContractNameBeforeTheLegacyOne(t *testing.T) {
	for name, tc := range map[string]struct {
		env  map[string]string
		want string
	}{
		"contract only": {map[string]string{"AGY_HOME": "/a"}, "/a"},
		"legacy only":   {map[string]string{"AGY_APP_DATA": "/l"}, "/l"},
		"both":          {map[string]string{"AGY_HOME": "/a", "AGY_APP_DATA": "/l"}, "/a"},
		"neither":       {nil, filepath.Join("/h", ".gemini", "antigravity-cli")},
	} {
		if _, got := Dirs("/h", func(k string) string { return tc.env[k] }); got != tc.want {
			t.Errorf("%s: Dirs = %q, want %q", name, got, tc.want)
		}
	}
}

// The default Dirs computes is the one env-contract.json declares for
// AGY_HOME, on every OS key it carries: two places answering one question.
func TestDirs_DefaultIsTheContractDefault(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "env-contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		EnvVars []struct {
			Name    string            `json:"name"`
			Default map[string]string `json:"default"`
		} `json:"env_vars"`
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	var defaults map[string]string
	for _, v := range c.EnvVars {
		if v.Name == "AGY_HOME" {
			defaults = v.Default
		}
	}
	if len(defaults) == 0 {
		t.Fatal("env-contract.json declares no default for AGY_HOME")
	}
	_, got := Dirs("HOME", func(string) string { return "" })
	got = filepath.ToSlash(got)
	for goos, d := range defaults {
		d = strings.NewReplacer(`$env:USERPROFILE`, "HOME", "$HOME", "HOME", `\`, "/").Replace(d)
		if d != got {
			t.Errorf("contract default for %s is %q, Dirs computes %q", goos, d, got)
		}
	}
}

func TestApplyMCP_WritesMasterAndPluginThenConverges(t *testing.T) {
	repo, p := fixture(t)
	rep, err := ApplyMCP(repo, p, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Changes() != 3 || rep.VaultMissing != "" {
		t.Fatalf("first apply = %+v, want 3 files written and the vault found", rep)
	}

	master := servers(t, filepath.Join(p.GeminiHome, "config", "mcp_config.json"))
	if got := master["hive-vault"]["env"].(map[string]any)["VAULT_PATH"]; got != p.Vault {
		t.Errorf("hive-vault VAULT_PATH = %v, want the placeholder replaced by %s", got, p.Vault)
	}
	if _, ok := master["hive"]; ok {
		t.Error("hive merged into agy's list; agy runs hive-vault itself")
	}
	if _, ok := master["context7"]; ok {
		t.Error("an http server merged; only stdio servers are agy's to run")
	}
	if got := master["sequential-thinking"]; !reflect.DeepEqual(got, map[string]any{
		"command": "npx", "args": []any{"-y", "@modelcontextprotocol/server-sequential-thinking"},
	}) {
		t.Errorf("sequential-thinking = %v", got)
	}
	// The setups' `cut -d' ' -f2-` handed a one-word command itself as its args.
	if got := master["solo"]["args"]; !reflect.DeepEqual(got, []any{}) {
		t.Errorf("a one-word command got args %v, want none", got)
	}
	plugin := servers(t, filepath.Join(p.AppData, "plugins", "hive-vault", "mcp_config.json"))
	if len(plugin) != 1 || !reflect.DeepEqual(plugin["hive-vault"], master["hive-vault"]) {
		t.Errorf("the plugin config must hold exactly the master's hive-vault entry: %v", plugin)
	}

	again, err := ApplyMCP(repo, p, false)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changes() != 0 || len(again.InSync) != 3 {
		t.Fatalf("second apply = %+v, want 0 changes", again)
	}
}

func TestApplyMCP_DryRunWritesNothing(t *testing.T) {
	repo, p := fixture(t)
	stale := filepath.Join(p.AppData, "mcp_config.json")
	writeFile(t, stale, "{}")
	rep, err := ApplyMCP(repo, p, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Written) != 3 || len(rep.Removed) != 1 {
		t.Fatalf("plan = %+v, want 3 to write and the stale copy to remove", rep)
	}
	if _, err := os.Stat(filepath.Join(p.GeminiHome, "config")); !os.IsNotExist(err) {
		t.Errorf("a plan created %s/config", p.GeminiHome)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Errorf("a plan removed the stale copy: %v", err)
	}
}

func TestApplyMCP_ReplacesASymlinkWithoutFollowingIt(t *testing.T) {
	repo, p := fixture(t)
	target := filepath.Join(t.TempDir(), "elsewhere.json")
	writeFile(t, target, "untouched")
	master := filepath.Join(p.GeminiHome, "config", "mcp_config.json")
	if err := os.MkdirAll(filepath.Dir(master), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, master); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	if _, err := ApplyMCP(repo, p, false); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(master); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("master is still a symlink (BUG-100): %v", err)
	}
	if raw, _ := os.ReadFile(target); string(raw) != "untouched" {
		t.Errorf("the write followed the symlink into %s", target)
	}
}

func TestApplyMCP_RemovesTheStaleAppDataCopy(t *testing.T) {
	repo, p := fixture(t)
	stale := filepath.Join(p.AppData, "mcp_config.json")
	writeFile(t, stale, "{}")
	rep, err := ApplyMCP(repo, p, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(stale); !os.IsNotExist(err) || len(rep.Removed) != 1 {
		t.Errorf("stale copy left behind (%v), report %+v", err, rep)
	}
}

func TestApplyMCP_AMissingVaultIsANoteNotAFailure(t *testing.T) {
	repo, p := fixture(t)
	p.Vault = filepath.Join(t.TempDir(), "absent")
	rep, err := ApplyMCP(repo, p, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.VaultMissing != p.Vault || rep.Changes() != 3 {
		t.Errorf("report = %+v, want the configs written and the vault named missing", rep)
	}
}

func TestApplyMCP_SourceAbsence(t *testing.T) {
	repo, p := fixture(t)
	if err := os.Remove(filepath.Join(repo, "mcp-servers.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyMCP(repo, p, false); err == nil {
		t.Error("a checkout without mcp-servers.json applied; it must fail, not pass as nothing to do")
	}
	if err := os.Remove(filepath.Join(repo, MCPRel)); err != nil {
		t.Fatal(err)
	}
	rep, err := ApplyMCP(repo, p, false)
	if err != nil || !rep.NoSource || rep.Changes() != 0 {
		t.Errorf("a checkout without %s = %+v, %v; want NoSource and nothing written", MCPRel, rep, err)
	}
}
