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
			// A verb the harness declared it has no equivalent for is ANSWERED,
			// not missing, and resolving it alone would legitimately render
			// nothing. Assert the declaration instead — the thing that must
			// never happen is a verb that is neither mapped nor declared, and
			// the loader already refuses to load that map at all.
			unsup, err := UnsupportedFor(m, []string{v}, name)
			if err != nil {
				t.Errorf("shipped map cannot report support for %q on %q: %v", v, name, err)
				continue
			}
			if len(unsup) == 1 {
				continue
			}
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

// TestShippedMapGrantsTheSkillCapabilityWhereItExists pins the fix for #1420 to
// the SHIPPED map rather than a fixture. The defect was not a wrong value: it was
// a verb that did not exist, so every persona deployed without the ability to
// invoke a skill and none could satisfy its own gate. A fixture would have proved
// the mechanism works while the real map still had no `skill` in it.
func TestShippedMapGrantsTheSkillCapabilityWhereItExists(t *testing.T) {
	m, err := LoadCapabilityMap(repoRootForTest(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !contains(toStrings(m["vocabulary"]), "skill") {
		t.Fatal("the shipped vocabulary declares no `skill` verb — no persona can be granted one")
	}

	line, err := ResolveCapabilities(m, []string{"read", "skill"}, "claude")
	if err != nil {
		t.Fatalf("resolve skill for claude: %v", err)
	}
	if !strings.Contains(line, "Skill") {
		t.Errorf("claude resolves read+skill to %q, which grants no skill invocation; claude's\n"+
			"`tools:` is an allow-list, so a tool not named is unavailable and the persona\n"+
			"deadlocks under enforce: block", line)
	}

	// The other half of AC2: a harness with no equivalent must be distinguishable
	// from one nobody has mapped. It reports, and it does not grant.
	unsup, err := UnsupportedFor(m, []string{"read", "skill"}, "opencode")
	if err != nil {
		t.Fatalf("unsupported for opencode: %v", err)
	}
	if len(unsup) != 1 || unsup[0] != "skill" {
		t.Fatalf("opencode should declare `skill` unsupported, got %v", unsup)
	}
	line, err = ResolveCapabilities(m, []string{"read", "skill"}, "opencode")
	if err != nil {
		t.Fatalf("resolve for opencode: %v", err)
	}
	if strings.Contains(strings.ToLower(line), "skill") {
		t.Errorf("opencode rendered %q — a verb declared unsupported must not appear as a grant", line)
	}
}

// TestCapabilityMapRejectsRenderBreakingNames is the guard for a defect measured
// on 2026-08-22: the resolver escapes nothing, so before the schema restricted
// these fields to identifier tokens, a native name carrying a comma or a brace
// did not appear in the rendered value — it ALTERED it.
//
//	"Read,Bash"    rendered as `tools: Read,Bash`   -> Bash granted to an agent
//	                                                   that only asked for `read`
//	"read},{bash"  rendered as `permission: {read},{bash: allow}` -> broke out of
//	                                                   the flow mapping entirely
//
// The first is privilege escalation through data. The render is deliberately
// unescaped — a frontmatter line is not a shell command — so the CONTRACT is
// where this has to be prevented, and this test is what keeps it there.
func TestCapabilityMapRejectsRenderBreakingNames(t *testing.T) {
	tests := []struct {
		name string
		doc  string
	}{
		{"a comma in a native name would grant a second tool", commaNativeCapabilityMap},
		{"a brace in a native name would escape the flow mapping", braceNativeCapabilityMap},
		{"a comma in the field name would split the line", commaFieldCapabilityMap},
		{"a decision-map harness with no grant cannot grant anything", noGrantCapabilityMap},
	}
	root := repoRootForTest(t)
	schema, err := os.ReadFile(filepath.Join(root, CapabilityMapSchemaFile))
	if err != nil {
		t.Fatalf("read shipped schema: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCapFixture(t, dir, CapabilityMapSchemaFile, string(schema))
			writeCapFixture(t, dir, CapabilityMapFile, tt.doc)
			if _, err := LoadCapabilityMap(dir); err == nil {
				t.Fatal("expected the map to be rejected at LOAD; " +
					"reaching the resolver means the rendered value can be altered by its own data")
			}
		})
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

const commaNativeCapabilityMap = `{
  "$comment": ["fixture"], "version": 1, "vocabulary": ["read"],
  "harnesses": {"claude": {"field": "tools", "form": "csv", "capabilities": {"read": ["Read,Bash"]}}}
}`

const braceNativeCapabilityMap = `{
  "$comment": ["fixture"], "version": 1, "vocabulary": ["read"],
  "harnesses": {"oc": {"field": "permission", "form": "decision-map", "grant": "allow", "capabilities": {"read": ["read},{bash"]}}}
}`

const commaFieldCapabilityMap = `{
  "$comment": ["fixture"], "version": 1, "vocabulary": ["read"],
  "harnesses": {"claude": {"field": "tools, model", "form": "csv", "capabilities": {"read": ["Read"]}}}
}`

// A decision map with no decision to write grants nothing. Caught at load so an
// unusable map never reaches a render.
const noGrantCapabilityMap = `{
  "$comment": ["fixture"], "version": 1, "vocabulary": ["read"],
  "harnesses": {"oc": {"field": "permission", "form": "decision-map", "capabilities": {"read": ["read"]}}}
}`
