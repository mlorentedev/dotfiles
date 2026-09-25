package deploy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// AI-042 review round 3 (Blocker, REAL): agy writes settings.json at runtime —
// a trusted workspace and four permission grants that no manifest named were
// found on the box — and Copilot writes trustedFolders. A replace, or a
// top-level key replace, deleted them on every deploy. The merge is now
// granular where the tools write: lists union, objects recurse.
func TestMergeInto_KeepsWhatTheToolAddedAtRuntime(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "settings.json")
	existing := `{
  "trustedWorkspaces": ["C:/Users/u/Projects/*", "C:/Users/u/Projects/Workspace/fae-onboarding"],
  "permissions": {"allow": ["read", "mcp(hive-vault/vault_query)"], "deny": ["rm"]},
  "model": "old-model",
  "ui": {"theme": "dark"}
}`
	if err := os.WriteFile(dst, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	managed := []byte(`{
  "trustedWorkspaces": ["C:/Users/u/Projects/*", "C:/Users/u/Projects/Workspace/*"],
  "permissions": {"allow": ["read", "write"], "deny": ["rm"]},
  "model": "new-model"
}`)

	out, changed, err := mergeInto(dst, managed)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("a managed entry the destination lacks is a change")
	}
	got := string(out)
	for _, keep := range []string{
		`"C:/Users/u/Projects/Workspace/fae-onboarding"`, // runtime-trusted workspace survives
		`"mcp(hive-vault/vault_query)"`,                  // runtime grant survives
		`"theme": "dark"`,                                 // a key only the destination has survives
	} {
		if !strings.Contains(got, keep) {
			t.Errorf("runtime state lost: %s missing from\n%s", keep, got)
		}
	}
	for _, add := range []string{`"C:/Users/u/Projects/Workspace/*"`, `"write"`, `"new-model"`} {
		if !strings.Contains(got, add) {
			t.Errorf("managed entry not applied: %s missing from\n%s", add, got)
		}
	}
	if strings.Contains(got, "old-model") {
		t.Error("a managed scalar must replace the destination's")
	}

	// Idempotent: merging the same source into the merged result changes nothing.
	if err := os.WriteFile(dst, out, 0o644); err != nil {
		t.Fatal(err)
	}
	again, changed, err := mergeInto(dst, managed)
	if err != nil {
		t.Fatal(err)
	}
	if changed || string(again) != got {
		t.Fatalf("second merge must be a no-op\nfirst:\n%s\nsecond:\n%s", got, again)
	}
}

// Round-3 Major (theoretical): with json.Number on both sides, string equality
// would call 15 and 15.0 different forever. Numbers compare by value.
func TestMergeInto_ComparesNumbersByValue(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "c.json")
	if err := os.WriteFile(dst, []byte(`{"n": 15.0, "m": 1e1, "z": -0}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, changed, err := mergeInto(dst, []byte(`{"n": 15, "m": 10, "z": 0}`))
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("15.0/15, 1e1/10 and -0/0 are the same values; not a change")
	}
}

// Round-3 Blocker (regression from round 2): the UseNumber decoders must still
// refuse a second document, as the manifest reader does.
func TestRender_RefusesTrailingData(t *testing.T) {
	src := []byte(`{"trustedFolders":["{HOME}/x"]} {"injected":true}`)
	if _, err := expandPaths(src, PathsSlash, "/home/u", noResolve); err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Errorf("expandPaths must refuse trailing data, got %v", err)
	}
	dst := filepath.Join(t.TempDir(), "c.json")
	if err := os.WriteFile(dst, []byte(`{"a":1} {"b":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := mergeInto(dst, []byte(`{"a":1}`)); err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Errorf("mergeInto must refuse a destination with trailing data, got %v", err)
	}
	if _, _, err := mergeInto(dst, []byte(`{"a":1} {"b":2}`)); err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Errorf("mergeInto must refuse a source with trailing data, got %v", err)
	}
}

// Round-3 Minor: the renderer writes `<`, `>`, `&` as themselves — the bytes
// the tool would write for the same content.
func TestRender_DoesNotHTMLEscape(t *testing.T) {
	out, err := expandPaths([]byte(`{"cmd":"a && b > c","p":"{HOME}/x"}`), PathsSlash, "/home/u", noResolve)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), `\u0026`) || strings.Contains(string(out), `\u003e`) {
		t.Errorf("HTML escaping must be off:\n%s", out)
	}
	dst := filepath.Join(t.TempDir(), "c.json")
	if err := os.WriteFile(dst, []byte(`{"cmd":"a && b"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	merged, _, err := mergeInto(dst, []byte(`{"x":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(merged), `\u0026`) {
		t.Errorf("merge must not HTML-escape either:\n%s", merged)
	}
}

// Round-3 Minor: the token rule is symmetric. Under slash, a native home
// inside a longer string is rendered with forward slashes; under native, a
// slash home inside a longer string is rendered with the OS separator. Only a
// string that begins with a token has the rest of it converted too.
func TestExpandPaths_RendersTheTokenInTheDeclaredFormWhereverItSits(t *testing.T) {
	nativeHome := filepath.FromSlash("/Users/u")
	if runtime.GOOS == "windows" {
		nativeHome = `C:\Users\u`
	}
	out, err := expandPaths([]byte(`{"url":"https://h/{HOME}/x","p":"{HOME}/a\\b"}`), PathsSlash, nativeHome, noResolve)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"https://h/`+filepath.ToSlash(nativeHome)+`/x"`) {
		t.Errorf("slash form must render the token with forward slashes inside a URL:\n%s", s)
	}
	if runtime.GOOS == "windows" && strings.Contains(s, `\\`) {
		t.Errorf("slash form must leave no backslash in a leading-token path:\n%s", s)
	}
}
