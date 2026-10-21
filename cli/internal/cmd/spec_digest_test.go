package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// A reviewer checks review-request.json's digests with this command, so it
// must print exactly the normalised map the launcher records, not the raw
// sha256 a reviewer would otherwise reach for and misread as drift (#2684).
func TestSpecDigestPrintsTheRecordedForm(t *testing.T) {
	root := makeRepo(t)
	dir := filepath.Join(root, "specs", "X-001-y")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	proposal := "---\nstatus: verifying\n---\n\n- [x] **AC1** — refuses\n"
	if err := os.WriteFile(filepath.Join(dir, "proposal.md"), []byte(proposal), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _, err := execute(t, "spec", "digest", "X-001-y")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		ContractDigests map[string]string `json:"contract_digests"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not the review request's JSON shape: %v\n%s", err, out)
	}
	want := spec.ContractDigests(dir)
	for name, sum := range want {
		if got.ContractDigests[name] != sum {
			t.Errorf("%s: printed %q, recorded form %q", name, got.ContractDigests[name], sum)
		}
	}
	if len(got.ContractDigests) != len(want) {
		t.Errorf("printed %v, want one entry per contract file %v", got.ContractDigests, want)
	}
	raw := sha256.Sum256([]byte(proposal))
	if got.ContractDigests["proposal.md"] == hex.EncodeToString(raw[:]) {
		t.Error("printed the raw sha256: a ticked box or a status change would read as drift")
	}
}

func TestSpecDigestRefusesAMissingSpec(t *testing.T) {
	makeRepo(t)
	if _, _, err := execute(t, "spec", "digest", "X-404-none"); err == nil || !strings.Contains(err.Error(), "spec not found") {
		t.Fatalf("want a spec-not-found refusal, got %v", err)
	}
}
