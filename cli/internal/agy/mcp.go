// Package agy writes what Antigravity (agy) reads its MCP servers from (#2384):
// the master ~/.gemini/config/mcp_config.json and the hive-vault plugin under
// agy's app-data dir. It replaces the "Consolidate MCP servers" blocks of
// setup-linux.sh and setup-windows.ps1, so a machine converged by dotf alone
// (a Mac) gets them too.
package agy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/claude"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// MCPRel is agy's own server list, relative to the repo root: the hive-vault
// entry, whose VAULT_PATH is a placeholder because agy does not expand
// variables inside JSON values.
const MCPRel = "ai/agy/mcp_servers.json"

const hiveVault = "hive-vault"

// errNoSource is a checkout without MCPRel: only that file's absence means
// "nothing to write". A missing mcp-servers.json stays an error.
var errNoSource = errors.New(MCPRel + " not found")

// Paths is where ApplyMCP writes, and the vault hive-vault is pointed at.
type Paths struct {
	GeminiHome string // $GEMINI_HOME, default ~/.gemini
	AppData    string // $AGY_HOME, default <GeminiHome>/antigravity-cli
	Vault      string // VAULT_PATH as the env contract resolves it, default ~/Projects/knowledge
}

// ResolvePaths reads the variables the setups exported, with their defaults.
// resolve is env.ResolvePath; it answers "" when no contract is found.
func ResolvePaths(home string, getenv, resolve func(string) string) Paths {
	p := Paths{Vault: resolve("VAULT_PATH")}
	p.GeminiHome, p.AppData = Dirs(home, getenv)
	if p.Vault == "" {
		p.Vault = filepath.Join(home, "Projects", "knowledge")
	}
	return p
}

// Dirs is $GEMINI_HOME and $AGY_HOME with their defaults, the contract's
// names. doctor reads them through it too, so the writer and the checker agree
// on the paths by construction. agy itself reads neither variable; both are
// ours. The setups' old AGY_APP_DATA is no longer read (#2404).
func Dirs(home string, getenv func(string) string) (geminiHome, appData string) {
	if geminiHome = getenv("GEMINI_HOME"); geminiHome == "" {
		geminiHome = filepath.Join(home, ".gemini")
	}
	if appData = getenv("AGY_HOME"); appData == "" {
		appData = filepath.Join(geminiHome, "antigravity-cli")
	}
	return geminiHome, appData
}

// MCPReport is what one ApplyMCP wrote and removed, or under a plan would.
type MCPReport struct {
	Written, InSync, Removed []string
	// NoSource is set when the checkout carries no ai/agy/mcp_servers.json:
	// nothing to write.
	NoSource bool
	// VaultMissing is the vault path hive-vault was given, when no directory
	// exists there. hive-vault fails at its first tool call until it does; the
	// configs are still written, so creating the vault later needs no rerun.
	VaultMissing string
}

// Changes counts the files written and removed.
func (r MCPReport) Changes() int { return len(r.Written) + len(r.Removed) }

// ApplyMCP writes the master config, with every stdio server of
// mcp-servers.json except hive (agy runs hive-vault itself) merged into agy's
// own list; the hive-vault plugin pair; and removes the copy an older setup
// left in the app-data dir. A file whose bytes already match is left alone,
// and a symlink is replaced by a regular file (BUG-100). Under dryRun nothing
// is written or removed.
func ApplyMCP(repoRoot string, p Paths, dryRun bool) (MCPReport, error) {
	var rep MCPReport
	master, plugin, err := render(repoRoot, p.Vault)
	if errors.Is(err, errNoSource) {
		rep.NoSource = true
		return rep, nil
	}
	if err != nil {
		return rep, err
	}
	pluginDir := filepath.Join(p.AppData, "plugins", hiveVault)
	type file struct {
		path string
		data []byte
	}
	files := []file{{filepath.Join(p.GeminiHome, "config", "mcp_config.json"), master}}
	if plugin != nil {
		files = append(files,
			file{filepath.Join(pluginDir, "plugin.json"), []byte(`{"name": "hive-vault"}` + "\n")},
			file{filepath.Join(pluginDir, "mcp_config.json"), plugin})
	}
	for _, f := range files {
		changed, err := writeIfDifferent(f.path, f.data, dryRun)
		if err != nil {
			return rep, err
		}
		if changed {
			rep.Written = append(rep.Written, f.path)
		} else {
			rep.InSync = append(rep.InSync, f.path)
		}
	}
	stale := filepath.Join(p.AppData, "mcp_config.json")
	if _, err := os.Lstat(stale); err == nil {
		if !dryRun {
			if err := os.Remove(stale); err != nil {
				return rep, fmt.Errorf("removing the stale %s: %w", stale, err)
			}
		}
		rep.Removed = append(rep.Removed, stale)
	}
	if info, err := os.Stat(p.Vault); err != nil || !info.IsDir() {
		rep.VaultMissing = p.Vault
	}
	return rep, nil
}

// render builds the master config and the hive-vault plugin config (nil when
// agy's list has no hive-vault entry). The output is indented JSON with sorted
// keys, so the same checkout renders the same bytes on every OS.
func render(repoRoot, vault string) (master, plugin []byte, err error) {
	raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(MCPRel))) //nolint:gosec // checkout-relative, fixed name
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, errNoSource
	}
	if err != nil {
		return nil, nil, err
	}
	var doc struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", MCPRel, err)
	}
	servers := doc.MCPServers
	if servers == nil {
		servers = map[string]map[string]any{}
	}
	if hv, ok := servers[hiveVault]; ok {
		env, _ := hv["env"].(map[string]any)
		if env == nil {
			env = map[string]any{}
			hv["env"] = env
		}
		env["VAULT_PATH"] = vault
	}
	root, err := claude.LoadServers(filepath.Join(repoRoot, claude.ServersRel))
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", claude.ServersRel, err)
	}
	for _, s := range root {
		if s.Name == "hive" || s.Transport != "stdio" {
			continue
		}
		argv := strings.Fields(s.Args) // LoadServers rejects empty args
		servers[s.Name] = map[string]any{"command": argv[0], "args": append([]string{}, argv[1:]...)}
	}
	if master, err = encode(map[string]any{"mcpServers": servers}); err != nil {
		return nil, nil, err
	}
	if hv, ok := servers[hiveVault]; ok {
		if plugin, err = encode(map[string]any{"mcpServers": map[string]any{hiveVault: hv}}); err != nil {
			return nil, nil, err
		}
	}
	return master, plugin, nil
}

func encode(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// writeIfDifferent writes data to path unless a regular file there already
// holds it. A symlink is removed first, so the write lands a regular file and
// never follows the link.
func writeIfDifferent(path string, data []byte, dryRun bool) (bool, error) {
	info, err := os.Lstat(path)
	switch {
	case err == nil && info.Mode()&fs.ModeSymlink != 0:
		if dryRun {
			return true, nil
		}
		if err := os.Remove(path); err != nil {
			return false, fmt.Errorf("replacing the symlink %s: %w", path, err)
		}
	case err == nil:
		if have, rerr := os.ReadFile(path); rerr == nil && bytes.Equal(have, data) { //nolint:gosec // a fixed path under agy's config
			return false, nil
		}
	case !errors.Is(err, fs.ErrNotExist):
		return false, err
	}
	if dryRun {
		return true, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // agy's config dirs are not private
		return false, err
	}
	if err := secrets.AtomicWriteMode(path, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}
