package secrets

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestCommittedAgeBlobsAreClaimed pins CLI-036: every sensitive/*.secret.age in
// the repository is named by an age-backed registry entry. The DR floor is the
// escrow in sensitive/dr/ (ADR-028 §5), so an unclaimed per-secret blob is not a
// backup. It is a stale, decryptable copy of a value, in a public repository.
// The 31 that ADR-002 left behind went unnoticed for months because the doctor
// check that could see them had been downgraded to a WARN (#971).
func TestCommittedAgeBlobsAreClaimed(t *testing.T) {
	raw, err := os.ReadFile("../../../secrets/registry.yaml")
	if err != nil {
		t.Fatalf("read the repository registry: %v", err)
	}
	reg, err := ParseRegistry(raw)
	if err != nil {
		t.Fatalf("parse the repository registry: %v", err)
	}
	// Entries resolves per-var age sources, so the claim set is the one the
	// resolver and doctor's checkSecrets use, not a re-derivation of it.
	claimed := map[string]bool{}
	for _, e := range reg.Entries(t.TempDir()) {
		if e.Backend == BackendAge || e.Backend == BackendAgeOffline {
			claimed[e.File] = true
		}
	}

	blobs, err := filepath.Glob("../../../sensitive/*.secret.age")
	if err != nil {
		t.Fatal(err)
	}
	// Non-vacuity: the floor's own blob must be seen, or the glob is looking in
	// the wrong place and the loop below proves nothing.
	if len(blobs) == 0 {
		t.Fatal("no sensitive/*.secret.age found; the path is wrong or the SSH_KEY floor blob is gone")
	}
	var unclaimed []string
	for _, b := range blobs {
		base := strings.TrimSuffix(filepath.Base(b), ".secret.age")
		if !claimed[base] {
			unclaimed = append(unclaimed, filepath.Base(b))
		}
	}
	sort.Strings(unclaimed)
	if len(unclaimed) > 0 {
		t.Errorf("%d committed blob(s) claimed by no age-backed registry entry; git rm them "+
			"(the escrow is the floor, ADR-028 §5):\n  %s", len(unclaimed), strings.Join(unclaimed, "\n  "))
	}
}
