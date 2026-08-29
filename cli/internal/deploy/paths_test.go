package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoWithTrust(t *testing.T, rel, body string) string {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// AC1 (AI-042, #1334): tokens are expanded inside JSON string values only, each
// rendered in the declared separator form, and the result is valid JSON — the
// encoder escapes a native Windows path, nobody escapes by hand.
func TestExpandPaths_RendersTheDeclaredForm(t *testing.T) {
	home := `C:\Users\u`
	if runtime.GOOS != "windows" {
		home = "/home/u"
	}
	src := []byte(`{"trustedFolders":["{HOME}/Projects","{HOME}/Projects/*"],"model":"m","nested":{"keep":"a/b","t":"{HOME}/x"}}`)

	native, err := expandPaths(src, PathsNative, home, noResolve)
	if err != nil {
		t.Fatal(err)
	}
	slash, err := expandPaths(src, PathsSlash, home, noResolve)
	if err != nil {
		t.Fatal(err)
	}
	var n, s map[string]any
	if err := json.Unmarshal(native, &n); err != nil {
		t.Fatalf("native output is not JSON: %v\n%s", err, native)
	}
	if err := json.Unmarshal(slash, &s); err != nil {
		t.Fatalf("slash output is not JSON: %v\n%s", err, slash)
	}
	wantNative := filepath.FromSlash(filepath.ToSlash(home) + "/Projects/*")
	wantSlash := filepath.ToSlash(home) + "/Projects/*"
	if got := n["trustedFolders"].([]any)[1]; got != wantNative {
		t.Errorf("native: want %q, got %q", wantNative, got)
	}
	if got := s["trustedFolders"].([]any)[1]; got != wantSlash {
		t.Errorf("slash: want %q, got %q", wantSlash, got)
	}
	if n["model"] != "m" || n["nested"].(map[string]any)["keep"] != "a/b" {
		t.Errorf("strings without a token must be untouched: %s", native)
	}
	if runtime.GOOS == "windows" && !strings.Contains(string(native), `\\Users\\`) {
		t.Errorf("a native Windows path must be JSON-escaped by the encoder: %s", native)
	}
}

func TestExpandPaths_RejectsWhatItCannotRender(t *testing.T) {
	if _, err := expandPaths([]byte(`{"a":"{HOME}"}`), "backslash", "/h", noResolve); err == nil || !strings.Contains(err.Error(), "unknown paths form") {
		t.Errorf("unknown form: %v", err)
	}
	if _, err := expandPaths([]byte(`not json`), PathsSlash, "/h", noResolve); err == nil || !strings.Contains(err.Error(), "JSON source") {
		t.Errorf("non-JSON source: %v", err)
	}
	if _, err := expandPaths([]byte(`{"a":"{NOPE}/x"}`), PathsSlash, "/h", noResolve); err == nil || !strings.Contains(err.Error(), "NOPE") {
		t.Errorf("unresolvable token must be named: %v", err)
	}
}

// AC2: paths composes with merge (the source is rendered, the destination's
// own keys survive) and PlanConfig sees a rendered destination as in sync.
func TestDeploy_PathsComposeWithMergeAndReportInSync(t *testing.T) {
	root := repoWithTrust(t, "ai/copilot/config.json", `{"trustedFolders":["{HOME}/Projects/*"]}`)
	home := t.TempDir()
	dst := filepath.Join(home, ".copilot", "config.json")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(`{"firstLaunchAt":"2026-03-11T00:00:00.000Z","trustedFolders":["/old"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := Config{Name: "copilot-config", Src: "ai/copilot/config.json", Dst: "{HOME}/.copilot/config.json", Mode: "0644", Strategy: StrategyMerge, Paths: PathsNative}

	res, err := Deploy(c, root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Fatal("a differing rendered list is a change")
	}
	got := readObject(t, dst)
	if got["firstLaunchAt"] != "2026-03-11T00:00:00.000Z" {
		t.Errorf("unmanaged key lost: %v", got)
	}
	want := filepath.FromSlash(filepath.ToSlash(home) + "/Projects/*")
	if fs := got["trustedFolders"].([]any); len(fs) != 1 || fs[0] != want {
		t.Errorf("want rendered %q, got %v", want, fs)
	}
	p, err := PlanConfig(c, root, home, noResolve)
	if err != nil {
		t.Fatal(err)
	}
	if p.Changed {
		t.Error("a rendered destination must read as in sync")
	}
}

func TestParseManifest_ValidatesPathsByName(t *testing.T) {
	_, err := ParseManifest([]byte(`{"version":3,"configs":[{"name":"x","src":"a","dst":"b","paths":"backslash"}]}`))
	if err == nil || !strings.Contains(err.Error(), `config "x"`) || !strings.Contains(err.Error(), "paths") {
		t.Errorf("an unknown paths form must be rejected naming the entry: %v", err)
	}
	if _, err := ParseManifest([]byte(`{"version":3,"configs":[{"name":"x","src":"a","dst":"b","paths":"slash"}]}`)); err != nil {
		t.Errorf("slash must parse: %v", err)
	}
}
