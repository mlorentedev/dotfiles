package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The top tier's no-fallback rule is a SHAPE rule here and a BEHAVIOUR rule in
// the dispatcher. This holds the shape half.
//
// The second case is the load-bearing one and is not obvious: naming `top`
// under `chains.properties` exempts it from `additionalProperties`, so a
// `maxItems` added without the sibling `$ref` would leave `chains.top` with no
// type, no minimum and no `pool:model` pattern — a strictly weaker key that
// LOOKS like an added constraint. It was written that way first, and this test
// is what the mistake earned.
func TestChainsTopIsCappedWithoutLosingTheChainShape(t *testing.T) {
	schema := readShippedSchema(t)

	tests := []struct {
		name    string
		chains  string
		wantErr string
	}{
		{
			name:   "one entry is the declared shape",
			chains: `{"top": ["claude:opus"], "mid": ["nan:qwen3.6"]}`,
		},
		{
			name:    "a second top entry is a declared fallback the top tier does not have",
			chains:  `{"top": ["claude:opus", "nan:deepseek-v4-flash"], "mid": ["nan:qwen3.6"]}`,
			wantErr: "maxItems",
		},
		{
			name:    "a top entry that is not pool:model is still rejected",
			chains:  `{"top": ["claude-opus"], "mid": ["nan:qwen3.6"]}`,
			wantErr: "pattern",
		},
		{
			name:    "a top that is not an array at all is still rejected",
			chains:  `{"top": "claude:opus", "mid": ["nan:qwen3.6"]}`,
			wantErr: "array",
		},
		{
			name:    "an empty top is not 'no fallback', it is nowhere to go",
			chains:  `{"top": [], "mid": ["nan:qwen3.6"]}`,
			wantErr: "minItems",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateModelMap([]byte(mapWithChains(tc.chains)), schema)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("valid map rejected: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("map accepted; want a rejection naming %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("rejection does not name %q: %v", tc.wantErr, err)
			}
		})
	}
}

func mapWithChains(chains string) string {
	return `{
  "$comment": ["fixture"],
  "version": 1,
  "pools": {
    "nan": {"auth": "subscription", "probe": "env:NAN_API_KEY"},
    "claude": {"auth": "subscription", "probe": "bin:claude"}
  },
  "harnesses": {"claude": {"pools": ["claude"], "render": "agent-md"}},
  "tiers": {"top": {"claude": "opus"}},
  "chains": ` + chains + `,
  "services": {}
}`
}

func readShippedSchema(t *testing.T) []byte {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		p := filepath.Join(dir, ModelMapSchemaFile)
		if b, err := os.ReadFile(p); err == nil {
			return b
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find %s walking up from the test directory", ModelMapSchemaFile)
		}
		dir = parent
	}
}
