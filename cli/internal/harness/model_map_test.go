package harness

import (
	"strings"
	"testing"
)

// The one rule the schema exists to enforce that a naive schema would miss:
// cross-block reference integrity. ADR-032 §3's reference schema shipped a
// `harnesses.codex.pools: ["codex"]` naming a pool the `pools` block never
// declares, and ADR-035 records it as an amendment rather than copying it in.
// A JSON Schema without a custom rule validates that document happily — every
// type is right and every required key is present — which is why this is the
// first test written.
func TestSchemaRejectsDanglingPoolReference(t *testing.T) {
	schema := []byte(`{
	  "type": "object",
	  "required": ["pools", "harnesses"],
	  "properties": {
	    "pools":     {"type": "object"},
	    "harnesses": {"type": "object"}
	  },
	  "x-poolReferencesResolve": true
	}`)

	cases := []struct {
		name       string
		doc        string
		wantErr    bool
		wantDetail string
	}{
		{
			name: "a harness naming an undeclared pool is rejected",
			doc: `{
			  "pools":     {"nan": {}},
			  "harnesses": {"codex": {"pools": ["codex"]}}
			}`,
			wantErr:    true,
			wantDetail: "codex",
		},
		{
			name: "a harness naming a declared pool is accepted",
			doc: `{
			  "pools":     {"nan": {}},
			  "harnesses": {"pi": {"pools": ["nan"]}}
			}`,
			wantErr: false,
		},
		{
			name: "one bad reference among several good ones is still rejected",
			doc: `{
			  "pools":     {"nan": {}, "claude": {}},
			  "harnesses": {"opencode": {"pools": ["nan", "openrouter"]}}
			}`,
			wantErr:    true,
			wantDetail: "openrouter",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateModelMap([]byte(tc.doc), schema)
			if tc.wantErr && err == nil {
				t.Fatalf("expected validation to reject the document, got nil error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected the document to validate, got: %v", err)
			}
			// The error has to name the offending pool. "invalid document" sends
			// the reader back to diff the file by hand, which is the failure mode
			// an error message exists to prevent.
			if tc.wantErr && !strings.Contains(err.Error(), tc.wantDetail) {
				t.Errorf("error must name the undeclared pool %q, got: %v", tc.wantDetail, err)
			}
		})
	}
}

// C15's teeth. A validator that silently ignores a schema construct it does not
// implement reports health it never established — the exact class this whole
// spec exists to close. Unknown constructs must be loud.
func TestValidatorRejectsUnimplementedSchemaConstructs(t *testing.T) {
	schema := []byte(`{
	  "type": "object",
	  "properties": {"pools": {"type": "object"}},
	  "patternProperties": {"^x-": {"type": "string"}}
	}`)

	err := ValidateModelMap([]byte(`{"pools": {}}`), schema)
	if err == nil {
		t.Fatal("a schema construct the validator does not implement must be a loud error, not a silent pass")
	}
	if !strings.Contains(err.Error(), "patternProperties") {
		t.Errorf("the error must name the unimplemented construct, got: %v", err)
	}
}
