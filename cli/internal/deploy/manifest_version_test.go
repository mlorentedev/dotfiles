package deploy

import (
	"strings"
	"testing"
)

// A binary must refuse a manifest it cannot fully read. Before this, a dotf
// predating `strategy`/`requires` decoded the AI-039 manifest with both fields
// silently dropped — every entry "replace", no gate — and `dotf deploy` on any
// box still running release 0.51.0 would have overwritten ~/.copilot/settings.json
// and wiped the box's own keys. Measured on the Windows work box with the stale
// binary: `would deploy copilot-settings`, `would deploy copilot-config`.
func TestParseManifest_RefusesWhatItCannotFullyRead(t *testing.T) {
	cases := []struct{ name, manifest, want string }{
		{"older schema", `{"version":1,"configs":[]}`, "version 1 unsupported"},
		{"newer schema", `{"version":3,"configs":[]}`, "version 3 unsupported"},
		{"unknown entry field", `{"version":2,"configs":[{"name":"x","src":"a","dst":"b","future":true}]}`, `unknown field "future"`},
		{"unknown top-level field", `{"version":2,"future":1,"configs":[]}`, `unknown field "future"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tc.manifest))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want an error containing %q, got %v", tc.want, err)
			}
			if err != nil && !strings.Contains(err.Error(), "dotf") {
				t.Errorf("the error must tell the operator what to update: %v", err)
			}
		})
	}
	// $comment is documentation, not an unknown field: the shipped manifest carries one.
	if _, err := ParseManifest([]byte(`{"$comment":["x"],"version":2,"configs":[]}`)); err != nil {
		t.Errorf("$comment must stay allowed: %v", err)
	}
}
