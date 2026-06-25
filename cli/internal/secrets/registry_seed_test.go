package secrets

import (
	"os"
	"testing"
)

// TestRealRegistry_RoundTripsEnvMapping is the drift guard: the committed
// secrets/registry.yaml must resolve to exactly the same secret set as the
// committed sensitive/env-mapping.conf, for as long as both exist (env-mapping is
// retired with the load-secrets twins in #493 PR-C). Same fixed home both sides.
func TestRealRegistry_RoundTripsEnvMapping(t *testing.T) {
	const home = "/h"
	const root = "../../.." // cli/internal/secrets → repo root

	regBytes, err := os.ReadFile(root + "/secrets/registry.yaml")
	if err != nil {
		t.Fatalf("read registry.yaml: %v", err)
	}
	reg, err := ParseRegistry(regBytes)
	if err != nil {
		t.Fatalf("ParseRegistry(real): %v", err)
	}

	mapFile, err := os.Open(root + "/sensitive/env-mapping.conf")
	if err != nil {
		t.Fatalf("open env-mapping.conf: %v", err)
	}
	defer func() { _ = mapFile.Close() }()
	mapEntries, err := ParseMapping(mapFile, home)
	if err != nil {
		t.Fatalf("ParseMapping(real): %v", err)
	}

	want := entriesByVar(mapEntries)
	got := entriesByVar(reg.Entries(home))

	for v, we := range want {
		ge, ok := got[v]
		if !ok {
			t.Errorf("registry is missing env var %q (mapping has it → %s)", v, we.File)
			continue
		}
		if ge.File != we.File || ge.IsFile != we.IsFile || ge.Dest != we.Dest {
			t.Errorf("var %q drift:\n  registry=%+v\n  mapping =%+v", v, ge, we)
		}
	}
	for v := range got {
		if _, ok := want[v]; !ok {
			t.Errorf("registry has extra env var %q not in env-mapping.conf", v)
		}
	}
	if len(want) != len(got) {
		t.Errorf("entry count: registry=%d env-mapping=%d", len(got), len(want))
	}
}
