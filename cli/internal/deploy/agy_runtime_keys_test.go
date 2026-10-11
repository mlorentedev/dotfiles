package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AI-043 (#1390). agy rewrites its own settings.json while it runs, and this
// entry used to be `replace`, so every deploy discarded whatever the user had
// trusted or granted. Measured on 2026-09-01: `/home/manu/Projects/ts-bridge`,
// a workspace trusted at runtime, was gone after a deploy.
//
// Why this test exists ALONGSIDE the two bats guards, and is not a duplicate of
// them: a reviewer pointed out that asserting `strategy == "merge"` in
// ai/deploy.json is static configuration validation, not a functional guard. It
// passes even if the deploy engine misreads the strategy. TestDeploy_Merge-
// PreservesUnmanagedKeys covers the engine property, but with the COPILOT
// config, so the claim "therefore agy's runtime keys survive" was an inference
// across two tests with nothing joining them.
//
// This joins them: it loads the SHIPPED ai/deploy.json and the SHIPPED
// ai/agy/settings.json and deploys the real entry into a temp HOME. It fails if
// the entry regresses to replace, if the template re-adds a key agy writes, or
// if the engine stops honouring merge -- the three ways the data loss can come
// back, none of which the static assertions catch on their own.
func TestDeploy_AgySettingsPreservesRuntimeKeys(t *testing.T) {
	const root = "../../.."

	raw, err := os.ReadFile(filepath.Join(root, ManifestRel))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Configs []Config `json:"configs"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	var cfg Config
	for _, c := range m.Configs {
		if c.Name == "agy-settings" {
			cfg = c
			break
		}
	}
	if cfg.Name == "" {
		t.Fatal("agy-settings is not declared in " + ManifestRel)
	}

	// The runtime state at risk, in the shape the incident had: an entry the
	// user added to a list, and a grant inside a nested object.
	home := t.TempDir()
	dst := filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	const runtime = `{
	  "model": "stale-value-the-template-should-win",
	  "trustedWorkspaces": ["/home/u/Projects/ts-bridge"],
	  "permissions": {"allow": ["mcp(hive-vault/*)"], "deny": ["command(rm -rf /)"]}
	}`
	if err := os.WriteFile(dst, []byte(runtime), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Deploy(cfg, root, home, noResolve, nil, false); err != nil {
		t.Fatal(err)
	}

	got := readObject(t, dst)

	// The template's grants and trust paths are the baseline every machine
	// gets. The merge unions them into the machine's lists, so an entry the
	// machine added survives and no baseline entry is missing.
	ws, _ := got["trustedWorkspaces"].([]any)
	if !containsAll(ws, "/home/u/Projects/ts-bridge") {
		t.Errorf("a trust path the machine added was lost: %v", got["trustedWorkspaces"])
	}
	if !containsAll(ws, filepath.ToSlash(home)+"/Projects/*") {
		t.Errorf("the baseline trust path is missing: %v", got["trustedWorkspaces"])
	}

	perms, _ := got["permissions"].(map[string]any)
	allow, _ := perms["allow"].([]any)
	if !containsAll(allow, "mcp(hive-vault/*)") {
		t.Errorf("a grant the machine added was lost: %v", perms)
	}
	if !containsAll(allow, baselineGrants...) {
		t.Errorf("the baseline grants are missing: %v", perms)
	}

	// The other half of the contract: dotfiles still owns what it declares.
	// A deploy that preserved everything by doing nothing would pass the two
	// assertions above, so pin that the template still wins on its own keys.
	tmplRaw, err := os.ReadFile(filepath.Join(root, cfg.Src))
	if err != nil {
		t.Fatal(err)
	}
	var tmpl map[string]any
	if err := json.Unmarshal(tmplRaw, &tmpl); err != nil {
		t.Fatal(err)
	}
	if got["model"] == "stale-value-the-template-should-win" {
		t.Error("the template's model did not win: merge is not writing managed keys")
	}
	if got["model"] != tmpl["model"] {
		t.Errorf("model = %v, template declares %v", got["model"], tmpl["model"])
	}
}

// baselineGrants is the minimum agy needs to work on every machine (owner,
// #2236). A machine may add to it; the merge never leaves it holding less.
var baselineGrants = []string{"command(*)", "read_file(*)", "write_file(*)", "read_url(*)"}

// TestAgyTemplateShipsTheBaselineGrants pins the owner decision on #2236: the
// grants and the trust paths are one baseline shipped to every machine.
func TestAgyTemplateShipsTheBaselineGrants(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("../../..", "ai", "agy", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tmpl map[string]any
	if err := json.Unmarshal(raw, &tmpl); err != nil {
		t.Fatal(err)
	}
	ws, _ := tmpl["trustedWorkspaces"].([]any)
	if !containsAll(ws, "{HOME}/Projects/*", "{HOME}/Projects/Workspace/*") {
		t.Errorf("ai/agy/settings.json must ship the baseline trust paths: %v", tmpl["trustedWorkspaces"])
	}
	perms, _ := tmpl["permissions"].(map[string]any)
	allow, _ := perms["allow"].([]any)
	if !containsAll(allow, baselineGrants...) {
		t.Errorf("ai/agy/settings.json must ship the baseline grants: %v", perms)
	}
	for _, g := range allow {
		if !keptGrantKind(fmt.Sprint(g)) {
			t.Errorf("ai/agy/settings.json grants %v, a kind agy drops on its next save, so converge never settles (#2312)", g)
		}
	}
}

// agyKeptGrantKinds are the grant kinds agy keeps when it rewrites its
// settings, measured on macOS with agy 1.3.3 on 2026-10-10: each appeared in a
// file agy itself had saved, mcp(...) included. It drops any other kind on its
// next save: search_web(*) was dropped, and web search needs no grant (#2312).
var agyKeptGrantKinds = []string{"command(", "read_file(", "write_file(", "read_url(", "mcp("}

// keptGrantKind reports whether grant is of a kind agy keeps.
func keptGrantKind(grant string) bool {
	for _, k := range agyKeptGrantKinds {
		if strings.HasPrefix(grant, k) {
			return true
		}
	}
	return false
}

// containsAll reports whether list holds every one of want.
func containsAll(list []any, want ...string) bool {
	for _, w := range want {
		found := false
		for _, v := range list {
			if v == w {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
