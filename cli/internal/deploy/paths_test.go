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
	// Union (review round 3): the folder Copilot itself trusted at runtime
	// ("/old") survives, and the rendered entry is present beside it.
	want := filepath.FromSlash(filepath.ToSlash(home) + "/Projects/*")
	if fs := got["trustedFolders"].([]any); len(fs) != 2 || fs[0] != "/old" || fs[1] != want {
		t.Errorf("want [/old %q] (runtime entry kept, rendered entry added), got %v", want, fs)
	}
	p, err := PlanConfig(c, root, home, noResolve)
	if err != nil {
		t.Fatal(err)
	}
	if p.Changed {
		t.Error("a rendered destination must read as in sync")
	}
}

// The replace half of AC2 (AI-042 review round 1, Minor): paths expansion runs
// before the whole-file replace, so agy's entry — replace + slash — lands a
// rendered list, reads as in sync afterwards, and a foreign key in the
// destination does NOT survive, which is what replace means and how it differs
// from the merge case above.
func TestDeploy_PathsComposeWithReplace(t *testing.T) {
	root := repoWithTrust(t, "ai/agy/settings.json", `{"trustedWorkspaces":["{HOME}/Projects/*"]}`)
	home := t.TempDir()
	dst := filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(`{"stale":true,"trustedWorkspaces":["/old"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := Config{Name: "agy-settings", Src: "ai/agy/settings.json", Dst: "{HOME}/.gemini/antigravity-cli/settings.json", Mode: "0644", Strategy: StrategyReplace, Paths: PathsSlash}

	res, err := Deploy(c, root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Fatal("a differing rendered list is a change")
	}
	got := readObject(t, dst)
	if _, stale := got["stale"]; stale {
		t.Errorf("replace must not keep a foreign key: %v", got)
	}
	want := filepath.ToSlash(home) + "/Projects/*"
	if ws := got["trustedWorkspaces"].([]any); len(ws) != 1 || ws[0] != want {
		t.Errorf("want rendered %q, got %v", want, ws)
	}
	p, err := PlanConfig(c, root, home, noResolve)
	if err != nil {
		t.Fatal(err)
	}
	if p.Changed {
		t.Error("a rendered destination must read as in sync")
	}
}

// AI-042 review round 2 (Blocker, REAL): decoding into `any` made every number
// a float64, so an integer above 2^53 came back rounded in a file that never
// declared an interest in numbers. Both renderers keep the digits they read.
func TestRender_PreservesLargeIntegersVerbatim(t *testing.T) {
	const big = `1234567890123456789`
	src := []byte(`{"id":` + big + `,"nested":{"ts":` + big + `},"trustedFolders":["{HOME}/Projects/*"]}`)

	out, err := expandPaths(src, PathsSlash, "/home/u", noResolve)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), big) || strings.Count(string(out), big) != 2 {
		t.Errorf("expandPaths rounded an integer:\n%s", out)
	}

	dst := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(dst, []byte(`{"firstLaunchAt":`+big+`}`), 0o644); err != nil {
		t.Fatal(err)
	}
	merged, _, err := mergeInto(dst, []byte(`{"id":`+big+`}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(merged), big) != 2 {
		t.Errorf("mergeInto rounded an integer on one side:\n%s", merged)
	}
}

// AI-042 review round 2 (Major, THEORETICAL): `native` converted every slash
// in any string that carried a token, so a tokenized URL would have come out
// with backslashes. A string is a path when it BEGINS with a token; a token
// elsewhere is expanded and the string is otherwise left alone.
func TestExpandPaths_ConvertsOnlyStringsThatBeginWithAToken(t *testing.T) {
	src := []byte(`{"path":"{HOME}/Projects/*","url":"https://api.example.com/{HOME}/x","plain":"a/b/c"}`)
	out, err := expandPaths(src, PathsNative, "/home/u", noResolve)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if want := filepath.FromSlash("/home/u/Projects/*"); got["path"] != want {
		t.Errorf("leading-token path: got %q, want %q", got["path"], want)
	}
	// Round 3 made the rule symmetric: the token's expansion is rendered in the
	// declared form wherever it sits; only the URL's own slashes are left alone.
	if want := "https://api.example.com/" + filepath.FromSlash("/home/u") + "/x"; got["url"] != want {
		t.Errorf("a tokenized URL keeps its own slashes under native, got %q, want %q", got["url"], want)
	}
	if got["plain"] != "a/b/c" {
		t.Errorf("a string without a token is untouched: %q", got["plain"])
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
