package docpaths

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadPolicy(t *testing.T) {
	t.Run("valid version one", func(t *testing.T) {
		path := writePolicy(t, `{
			"version": 1,
			"governed_basenames": ["AGENTS.md", "README.md"],
			"exclude_prefixes": ["docs/", "specs/"],
			"known_extensions": [".md", ".sh"],
			"required_members": ["AGENTS.md"],
			"source_extensions": [".conf", ".sh", ".zsh"]
		}`)
		got, err := LoadPolicy(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.Version != 1 || len(got.GovernedBasenames) != 2 || got.RequiredMembers[0] != "AGENTS.md" {
			t.Fatalf("LoadPolicy() = %+v", got)
		}
	})

	cases := []struct {
		name string
		body string
		want string
	}{
		{"unknown field", `{"version":1,"governed_basenames":["AGENTS.md"],"unknown":true}`, "unknown field"},
		{"unsupported version", `{"version":2,"governed_basenames":["AGENTS.md"]}`, "unsupported version"},
		{"missing governed names", `{"version":1}`, "governed_basenames"},
		{"escaping exclude", `{"version":1,"governed_basenames":["AGENTS.md"],"exclude_prefixes":["../docs/"]}`, "exclude_prefix"},
		{"empty exclude", `{"version":1,"governed_basenames":["AGENTS.md"],"exclude_prefixes":[""]}`, "exclude_prefix"},
		{"governed path instead of basename", `{"version":1,"governed_basenames":["ai/AGENTS.md"]}`, "governed_basename"},
		{"invalid known extension", `{"version":1,"governed_basenames":["AGENTS.md"],"known_extensions":["md"]}`, "known_extension"},
		{"escaping required member", `{"version":1,"governed_basenames":["AGENTS.md"],"required_members":["../AGENTS.md"]}`, "required_member"},
		{"invalid source extension", `{"version":1,"governed_basenames":["AGENTS.md"],"source_extensions":["sh"]}`, "source_extension"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadPolicy(writePolicy(t, tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("LoadPolicy() error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

func TestCurrentPolicyLoads(t *testing.T) {
	policy, err := LoadPolicy(filepath.Join("..", "..", "..", "harness", "doc-paths.json"))
	if err != nil {
		t.Fatal(err)
	}
	wantBasenames := []string{
		"AGENTS.md",
		"CLAUDE.md",
		"AGY.md",
		"GEMINI.md",
		"copilot-instructions.md",
		"README.md",
	}
	if !reflect.DeepEqual(policy.GovernedBasenames, wantBasenames) {
		t.Fatalf("governed basenames = %v, want %v", policy.GovernedBasenames, wantBasenames)
	}
	if len(policy.RequiredMembers) < 5 || len(policy.KnownExtensions) < 10 {
		t.Fatalf("current policy is incomplete: %+v", policy)
	}
}

func writePolicy(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "doc-paths.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
