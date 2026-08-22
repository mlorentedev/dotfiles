package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveCapabilitiesAgainstShippedMap exercises the resolver against the
// map the repository actually ships, so a fixture cannot drift away from it.
func TestResolveCapabilitiesAgainstShippedMap(t *testing.T) {
	root := repoRootForTest(t)
	m, err := LoadCapabilityMap(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	tests := []struct {
		name        string
		caps        []string
		harness     string
		want        string
		wantErrSubs []string
	}{
		{
			// An allow-list: what is named is granted, what is not is denied.
			name:    "claude renders a csv allow-list",
			caps:    []string{"read", "shell"},
			harness: "claude",
			want:    "tools: Read, Glob, Bash",
		},
		{
			// Glob serves both read and search; it must appear once, at the
			// position its first requester put it.
			name:    "overlapping natives are deduped, first occurrence wins",
			caps:    []string{"read", "search"},
			harness: "claude",
			want:    "tools: Read, Glob, Grep",
		},
		{
			// A decision map grants without denying, and renders as a YAML flow
			// mapping so it still fits one frontmatter line.
			name:    "opencode renders a decision map",
			caps:    []string{"shell", "web"},
			harness: "opencode",
			want:    "permission: {bash: allow, webfetch: allow, websearch: allow}",
		},
		{
			name:    "the whole vocabulary resolves for every declared harness",
			caps:    []string{"read", "search", "edit", "shell", "web"},
			harness: "claude",
			want:    "tools: Read, Glob, Grep, Edit, Write, Bash, WebFetch, WebSearch",
		},
		{
			name:        "an unmapped capability names itself and the harness",
			caps:        []string{"telepathy"},
			harness:     "claude",
			wantErrSubs: []string{"telepathy", "claude"},
		},
		{
			// Absent on purpose, not overlooked: guessing native names would
			// render a definition granting tools that may not exist.
			name:        "an undeclared harness names what IS declared",
			caps:        []string{"read"},
			harness:     "copilot",
			wantErrSubs: []string{"copilot", "claude", "opencode"},
		},
		{
			name:        "an empty request refuses rather than granting nothing",
			caps:        []string{""},
			harness:     "claude",
			wantErrSubs: []string{"claude"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveCapabilities(m, tt.caps, tt.harness)
			if len(tt.wantErrSubs) > 0 {
				if err == nil {
					t.Fatalf("expected an error, got %q", got)
				}
				if got != "" {
					t.Errorf("failed resolution returned %q; it must return nothing", got)
				}
				for _, sub := range tt.wantErrSubs {
					if !strings.Contains(err.Error(), sub) {
						t.Errorf("error does not name %q: %v", sub, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

// TestCapabilityMapFailsLoudWhenUnreadable is C15 for this registry: an absent,
// unschema'd or invalid map errors rather than resolving to a permissive default.
// An empty capability value is not "no opinion" — for a csv allow-list it is a
// definition granting nothing.
func TestCapabilityMapFailsLoudWhenUnreadable(t *testing.T) {
	shippedSchema := func(t *testing.T) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(repoRootForTest(t), CapabilityMapSchemaFile))
		if err != nil {
			t.Fatalf("read shipped schema: %v", err)
		}
		return string(b)
	}

	tests := []struct {
		name    string
		seed    func(t *testing.T) string
		wantSub string
	}{
		{
			name: "no map at all",
			seed: func(t *testing.T) string { return t.TempDir() },
		},
		{
			name: "a map with no schema beside it",
			seed: func(t *testing.T) string {
				dir := t.TempDir()
				writeCapFixture(t, dir, CapabilityMapFile, minimalCapabilityMap)
				return dir
			},
		},
		{
			name: "a map that is not JSON",
			seed: func(t *testing.T) string {
				dir := t.TempDir()
				writeCapFixture(t, dir, CapabilityMapFile, "{ not json")
				writeCapFixture(t, dir, CapabilityMapSchemaFile, "{}")
				return dir
			},
		},
		{
			// The cross-block rule a stock schema cannot express. It validates
			// against every standard keyword and would then render a claude
			// definition missing exactly the tool the persona asked for.
			name: "a harness that does not cover the whole vocabulary",
			seed: func(t *testing.T) string {
				dir := t.TempDir()
				writeCapFixture(t, dir, CapabilityMapSchemaFile, shippedSchema(t))
				writeCapFixture(t, dir, CapabilityMapFile, partialCapabilityMap)
				return dir
			},
			wantSub: "shell",
		},
		{
			name: "a harness that maps a verb the vocabulary does not declare",
			seed: func(t *testing.T) string {
				dir := t.TempDir()
				writeCapFixture(t, dir, CapabilityMapSchemaFile, shippedSchema(t))
				writeCapFixture(t, dir, CapabilityMapFile, extraVerbCapabilityMap)
				return dir
			},
			wantSub: "telepathy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadCapabilityMap(tt.seed(t))
			if err == nil {
				t.Fatal("expected an error; an unreadable capability map must never load")
			}
			if tt.wantSub != "" && !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("error does not name %q: %v", tt.wantSub, err)
			}
		})
	}
}

// TestShippedCapabilityMapCoversEveryDeclaredHarness is the assertion that would
// have caught a half-filled registry before it reached a deploy.
func TestShippedCapabilityMapCoversEveryDeclaredHarness(t *testing.T) {
	root := repoRootForTest(t)
	m, err := LoadCapabilityMap(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	vocab := toStrings(m["vocabulary"])
	harnesses, _ := m["harnesses"].(map[string]any)
	if len(harnesses) == 0 {
		t.Fatal("shipped map declares no harnesses")
	}
	for _, name := range sortedKeys(harnesses) {
		for _, v := range vocab {
			line, err := ResolveCapabilities(m, []string{v}, name)
			if err != nil {
				t.Errorf("shipped map cannot resolve %q for %q: %v", v, name, err)
				continue
			}
			if strings.TrimSpace(line) == "" {
				t.Errorf("shipped map resolves %q for %q to an empty line", v, name)
			}
		}
	}
}

func writeCapFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

const minimalCapabilityMap = `{
  "$comment": ["fixture"],
  "version": 1,
  "vocabulary": ["read"],
  "harnesses": {"claude": {"field": "tools", "form": "csv", "capabilities": {"read": ["Read"]}}}
}`

// Type-correct and schema-valid; only the cross-block coverage rule rejects it.
const partialCapabilityMap = `{
  "$comment": ["fixture"],
  "version": 1,
  "vocabulary": ["read", "shell"],
  "harnesses": {"claude": {"field": "tools", "form": "csv", "capabilities": {"read": ["Read"]}}}
}`

const extraVerbCapabilityMap = `{
  "$comment": ["fixture"],
  "version": 1,
  "vocabulary": ["read"],
  "harnesses": {"claude": {"field": "tools", "form": "csv", "capabilities": {"read": ["Read"], "telepathy": ["Mind"]}}}
}`
