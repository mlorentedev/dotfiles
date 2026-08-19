package secrets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ManifestFileName sits beside the escrow. Exact name, not a pattern: the escrow
// directory's .gitignore is deny-by-default so that a plaintext `bw export` can
// never be staged, and it un-ignores this file by name for the same reason — a
// glob would reopen the door the guard exists to hold shut.
const ManifestFileName = "escrow-manifest.json"

// EscrowManifest is a value-free description of what an escrow contained at the
// moment it was written, so a later check can ask whether the escrow still
// describes the vault rather than merely how old it is.
//
// Those two questions diverge in the case that loses data. A deleted item vanishes
// from the vault, every surviving item's revision can predate the escrow, and an
// age comparison reports healthy about a vault that has lost a secret. Count and
// digest see it; a timestamp cannot (#1077).
//
// Nothing here is a secret. Item ids are UUIDs, revisions are timestamps, and the
// digest is over those two alone — no name, no field, no value, no folder. That is
// what makes the file committable, and it is a property the tests assert rather
// than a promise this comment makes.
type EscrowManifest struct {
	Count       int    `json:"count"`
	MaxRevision string `json:"max_revision"`
	Digest      string `json:"digest"`
}

// ManifestFrom derives the manifest from an exported vault document. It reads the
// plaintext Backup already holds in memory, so it costs no extra API call and adds
// no session dependency — the whole reason this could ship beside the escrow rather
// than as a separate command.
//
// Declared blind spots, because an escrow that hides them is worse than one that
// names them: `bw export` has never carried attachments, it does not include the
// trash, and folders are not covered here. A folder rename moves nothing this
// digest sees. The escrow's blind spots are the manifest's, by construction.
func ManifestFrom(export []byte) (EscrowManifest, error) {
	var doc struct {
		Items []ItemRevision `json:"items"`
	}
	if err := json.Unmarshal(export, &doc); err != nil {
		return EscrowManifest{}, fmt.Errorf("manifest: export is not the expected JSON document: %w", err)
	}
	return ManifestFromItems(doc.Items)
}

// ItemRevision is the pair both producers reduce to: the escrowed export and the
// live vault listing. Having ONE reduction is the point — two spellings of "what a
// manifest is over" is the two-file agreement nobody checks, and here it would
// produce permanent false drift between two descriptions of the same vault.
//
// Measured 2026-08-19 before this was built, because the equivalence is an
// assumption until it is not: the escrow held 177 items, `/list/object/items`
// returned 178, and the sets differed by exactly one id — an addition, not a
// systematic divergence. Both carry `id` and `revisionDate`.
type ItemRevision struct {
	ID           string `json:"id"`
	RevisionDate string `json:"revisionDate"`
}

// ManifestFromItems is the reduction itself.
func ManifestFromItems(items []ItemRevision) (EscrowManifest, error) {
	doc := struct{ Items []ItemRevision }{items}
	if len(doc.Items) == 0 {
		// Refusing beats describing nothing: a manifest claiming zero items would
		// later read as "everything was deleted" against any real vault, which is
		// the loudest possible wrong answer.
		return EscrowManifest{}, fmt.Errorf("manifest: export contains no items — refusing to describe an empty vault " +
			"(the escrow itself was written and verified; only this file is missing)")
	}

	lines := make([]string, 0, len(doc.Items))
	max := ""
	for _, it := range doc.Items {
		if it.ID == "" {
			return EscrowManifest{}, fmt.Errorf("manifest: an item has no id, so the digest could not be stable")
		}
		lines = append(lines, it.ID+":"+it.RevisionDate)
		// Lexicographic max is correct only because bw emits ISO-8601 in UTC with
		// a fixed width, where string order IS chronological order. Stated rather
		// than assumed: the day that format changes, this silently picks a wrong
		// maximum instead of failing.
		if it.RevisionDate > max {
			max = it.RevisionDate
		}
	}
	// Sorted, because `bw export` gives no ordering guarantee and an unsorted
	// digest would report drift on every run — an alarm that always fires is one
	// nobody reads, which is the failure this file exists to avoid.
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))

	return EscrowManifest{
		Count:       len(doc.Items),
		MaxRevision: max,
		Digest:      hex.EncodeToString(sum[:]),
	}, nil
}

// Differs reports how the stored manifest and the live vault disagree, in words a
// reader can act on, or "" when they describe the same vault.
//
// The parameters are NAMED stored and live rather than left to convention, because a
// swapped call site would invert "added" and "DELETED" — a signal answering a
// question it was not asked, which is this repository's recurring defect. A test
// asserts the direction so the swap cannot survive the suite.
func (stored EscrowManifest) Differs(live EscrowManifest) string {
	if stored.Digest == live.Digest {
		return ""
	}
	asOf := ""
	if stored.MaxRevision != "" {
		asOf = fmt.Sprintf(" The escrow describes the vault as of %s.", stored.MaxRevision)
	}
	switch {
	case live.Count > stored.Count:
		return fmt.Sprintf("%d item(s) added since the escrow was written.%s", live.Count-stored.Count, asOf)
	case live.Count < stored.Count:
		return fmt.Sprintf("%d item(s) DELETED since the escrow was written — restoring from it would lose them.%s",
			stored.Count-live.Count, asOf)
	default:
		// Equal counts do NOT mean no deletion. One removed and one added in the
		// same window lands here, and ranking this below the DELETED case would
		// hide exactly what the digest exists to catch. The remedy is identical
		// either way — re-run backup — so the message says so rather than
		// pretending to attribute what count and digest cannot.
		return fmt.Sprintf("%d item(s), same count, but the vault changed since the escrow was written — "+
			"this can include a deletion paired with an addition.%s", live.Count, asOf)
	}
}
