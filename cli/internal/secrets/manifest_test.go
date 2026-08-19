package secrets

import (
	"encoding/json"
	"strings"
	"testing"
)

// The escrow manifest (#1077). It exists because "how old is the escrow" is a
// cheaper question than "does the escrow still describe the vault", and the two
// answers diverge in the case that loses data.

func exportWith(items ...[2]string) []byte {
	type item struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		RevisionDate string `json:"revisionDate"`
	}
	var doc struct {
		Encrypted bool   `json:"encrypted"`
		Items     []item `json:"items"`
	}
	for _, it := range items {
		doc.Items = append(doc.Items, item{ID: it[0], RevisionDate: it[1], Name: "name-of-" + it[0]})
	}
	b, _ := json.Marshal(doc)
	return b
}

func mustManifest(t *testing.T, export []byte) EscrowManifest {
	t.Helper()
	m, err := ManifestFrom(export)
	if err != nil {
		t.Fatalf("ManifestFrom: %v", err)
	}
	return m
}

func TestManifest_CarriesNoSecretValue(t *testing.T) {
	// The type's comment claims the file is committable because it holds no value.
	// A comment is a promise; this is the assertion. The sentinel rides in a field
	// a real export carries (`name`), which is exactly the kind of thing that leaks
	// when a struct grows a field nobody re-read.
	const sentinel = "SUPERSECRETVALUEDONOTCOMMIT"
	raw := []byte(`{"items":[{"id":"a","revisionDate":"2026-01-01T00:00:00.000Z","name":"` + sentinel +
		`","login":{"password":"` + sentinel + `"}}]}`)

	blob, err := json.Marshal(mustManifest(t, raw))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), sentinel) {
		t.Fatalf("the manifest leaked a value from the export: %s", blob)
	}
	for _, forbidden := range []string{"name", "login", "password", "folder"} {
		if strings.Contains(string(blob), forbidden) {
			t.Errorf("manifest carries a %q key; it must hold ids, revisions and a digest only: %s", forbidden, blob)
		}
	}
}

func TestManifest_DigestIsOrderIndependent(t *testing.T) {
	// `bw export` gives no ordering guarantee. An unsorted digest would report drift
	// on every run, and an alarm that always fires is one nobody reads — which is
	// this file's own failure mode arriving through its implementation.
	a := mustManifest(t, exportWith([2]string{"a", "2026-01-01T00:00:00.000Z"}, [2]string{"b", "2026-02-01T00:00:00.000Z"}))
	b := mustManifest(t, exportWith([2]string{"b", "2026-02-01T00:00:00.000Z"}, [2]string{"a", "2026-01-01T00:00:00.000Z"}))
	if a.Digest != b.Digest {
		t.Fatal("the same vault in a different order must produce the same digest")
	}
}

func TestManifest_MaxRevisionIsTheNewest(t *testing.T) {
	m := mustManifest(t, exportWith(
		[2]string{"a", "2026-01-01T00:00:00.000Z"},
		[2]string{"b", "2026-08-19T00:00:00.000Z"},
		[2]string{"c", "2026-03-01T00:00:00.000Z"}))
	if m.MaxRevision != "2026-08-19T00:00:00.000Z" {
		t.Fatalf("max_revision is %q, expected the newest", m.MaxRevision)
	}
}

// --- Differs: the case a timestamp comparison cannot see ------------------------

func TestDiffers_DeletionWhoseSurvivorsAllPredateTheEscrow(t *testing.T) {
	// AC5's specific scenario, and the reason this feature exists. Two items are
	// escrowed; one is later deleted. Every survivor's revisionDate PREDATES the
	// escrow, so a "is anything newer than the escrow" check passes cleanly while a
	// secret has been lost.
	stored := mustManifest(t, exportWith(
		[2]string{"a", "2026-01-01T00:00:00.000Z"},
		[2]string{"b", "2026-01-02T00:00:00.000Z"}))
	live := mustManifest(t, exportWith([2]string{"a", "2026-01-01T00:00:00.000Z"}))

	if live.MaxRevision >= stored.MaxRevision {
		t.Fatal("fixture is wrong: the survivor must predate the escrow for this case to mean anything")
	}
	got := stored.Differs(live)
	if !strings.Contains(got, "DELETED") {
		t.Fatalf("a deletion must be named as one, got: %q", got)
	}
	if !strings.Contains(got, "1 item") {
		t.Errorf("the message must say how many, got: %q", got)
	}
}

func TestDiffers_DirectionCannotBeSwapped(t *testing.T) {
	// stored.Differs(live). A swapped call site would invert "added" and "DELETED" —
	// a signal answering a question it was not asked. This pins the direction so the
	// swap cannot survive the suite.
	two := mustManifest(t, exportWith([2]string{"a", "2026-01-01T00:00:00.000Z"}, [2]string{"b", "2026-01-02T00:00:00.000Z"}))
	one := mustManifest(t, exportWith([2]string{"a", "2026-01-01T00:00:00.000Z"}))

	if got := two.Differs(one); !strings.Contains(got, "DELETED") {
		t.Errorf("stored=2 live=1 is a deletion, got: %q", got)
	}
	if got := one.Differs(two); !strings.Contains(got, "added") {
		t.Errorf("stored=1 live=2 is an addition, got: %q", got)
	}
}

func TestDiffers_EqualCountDoesNotRankBelowADeletion(t *testing.T) {
	// One removed and one added in the same window lands in the equal-count branch.
	// Count and digest cannot attribute it, and the remedy is the same either way —
	// but the message must not let a reader rank this below the DELETED case.
	stored := mustManifest(t, exportWith([2]string{"a", "2026-01-01T00:00:00.000Z"}, [2]string{"b", "2026-01-02T00:00:00.000Z"}))
	live := mustManifest(t, exportWith([2]string{"a", "2026-01-01T00:00:00.000Z"}, [2]string{"c", "2026-01-03T00:00:00.000Z"}))

	got := stored.Differs(live)
	if !strings.Contains(got, "deletion") {
		t.Fatalf("the equal-count message must warn that a deletion may be hidden here, got: %q", got)
	}
}

func TestDiffers_SameVaultIsSilent(t *testing.T) {
	// A check that fires on an unchanged vault is worse than none.
	m := mustManifest(t, exportWith([2]string{"a", "2026-01-01T00:00:00.000Z"}))
	if got := m.Differs(m); got != "" {
		t.Fatalf("an unchanged vault must produce no message, got: %q", got)
	}
}

// --- refusals -------------------------------------------------------------------

func TestManifest_RefusesAnEmptyVault(t *testing.T) {
	// A manifest claiming zero items would later read as "everything was deleted"
	// against any real vault — the loudest possible wrong answer.
	_, err := ManifestFrom([]byte(`{"items":[]}`))
	if err == nil {
		t.Fatal("an export with no items must be refused")
	}
	// And the operator must learn the escrow itself is fine.
	if !strings.Contains(err.Error(), "written and verified") {
		t.Errorf("the refusal must say the escrow survived, got: %v", err)
	}
}

func TestManifest_RefusesAnItemWithNoID(t *testing.T) {
	_, err := ManifestFrom([]byte(`{"items":[{"revisionDate":"2026-01-01T00:00:00.000Z"}]}`))
	if err == nil {
		t.Fatal("an item without an id makes the digest unstable and must be refused")
	}
}

func TestManifest_RefusesSomethingThatIsNotAnExport(t *testing.T) {
	if _, err := ManifestFrom([]byte("not json at all")); err == nil {
		t.Fatal("a non-JSON document must be refused")
	}
}
