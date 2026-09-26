package spec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
)

// ContractDigests returns the normalised SHA-256 of each contract file in
// specDir, keyed by file name; an absent file maps to "".
//
// This is what makes review freshness a question about CONTENT rather than
// about commits (SDD-042, #1566). The launcher records these digests in
// review-request.json when the reviewer starts, and archive recomputes them
// from disk, so the answer does not change when a squash-merge, a rebase or a
// fresh clone discards the reviewed commit. Normalisation folds away only what
// is bookkeeping rather than contract — see normaliseContract.
func ContractDigests(specDir string) map[string]string {
	return contractDigests(specDir, true)
}

// legacyContractDigests is the digest before the lifecycle status was folded
// (HARNESS-151). A review launched earlier recorded this form, and it must stay
// fresh after the upgrade rather than demand a re-review of an unchanged spec.
func legacyContractDigests(specDir string) map[string]string {
	return contractDigests(specDir, false)
}

func contractDigests(specDir string, foldStatus bool) map[string]string {
	digests := make(map[string]string, len(contractFiles))
	for _, name := range contractFiles {
		data, err := os.ReadFile(filepath.Join(specDir, name))
		if err != nil {
			digests[name] = ""
			continue
		}
		sum := sha256.Sum256(normaliseContract(name, data, foldStatus))
		digests[name] = hex.EncodeToString(sum[:])
	}
	return digests
}

// listCheckbox matches the marker of a markdown list checkbox, bulleted or
// ordered (`1.` / `1)`), at any depth.
var listCheckbox = regexp.MustCompile(`(?m)^(\s*(?:[-*+]|\d+[.)])\s+)\[[ xX]\]`)

// harnessFeatureFields are the features.json fields the harness writes after a
// review (a feature's lifecycle state and the output that proved it). They are
// progress, not contract, so they are blanked before digesting.
var harnessFeatureFields = []string{"state", "evidence"}

// normaliseContract folds exactly these kinds of bookkeeping, and nothing else:
//
//   - line endings, so a Windows checkout (`* text=auto`, no eol for .md) does
//     not read as an edit;
//   - list checkbox state, so ticking `- [ ]` to `- [x]` is progress and a
//     review whose own finding was "tick the boxes" does not invalidate itself
//     (#998 part 2);
//   - the harness-owned fields of features.json;
//   - the frontmatter's lifecycle `status:`, which the archive itself writes
//     (HARNESS-151), when foldStatus is set.
//
// A reworded criterion, a new task or a changed feature behavior still moves
// the digest; the tests pin both halves.
func normaliseContract(name string, data []byte, foldStatus bool) []byte {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if filepath.Ext(name) == ".json" {
		return normaliseFeatures(data)
	}
	// The frontmatter status is lifecycle, not contract (HARNESS-151): `dotf spec
	// archive` writes `status: archived` after the freshness check passes, so
	// reading it would stale every archived review against its own archive.
	// setStatus touches the first frontmatter block only; a body line is prose.
	if foldStatus {
		data = []byte(setStatus(string(data), "-"))
	}
	return listCheckbox.ReplaceAll(data, []byte("${1}[ ]"))
}

// normaliseFeatures re-encodes features.json with the harness-owned fields
// removed. encoding/json sorts map keys, so the result is canonical whatever
// the source's key order or indentation. Unparseable input digests by its
// bytes: a malformed file is still content, and still has to match.
func normaliseFeatures(data []byte) []byte {
	var features []map[string]any
	if err := json.Unmarshal(data, &features); err != nil {
		return data
	}
	for _, f := range features {
		for _, field := range harnessFeatureFields {
			delete(f, field)
		}
	}
	canonical, err := json.Marshal(features)
	if err != nil {
		return data
	}
	return canonical
}
