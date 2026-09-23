package secrets

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// itemsPayload wraps item JSON in bw serve's inner list envelope.
func itemsPayload(t *testing.T, items string) json.RawMessage {
	t.Helper()
	return json.RawMessage(`{"object":"list","data":[` + items + `]}`)
}

// THE TEST THIS FILE EXISTS FOR.
//
// The projection's whole claim is that a value cannot cross it. Asserting the
// fields we DO want is not that claim — it passes just as well if a password
// rides along in a struct nobody looked at. So this feeds a payload stuffed with
// distinctive secret material, marshals the entire result back to JSON, and
// asserts none of it survives anywhere in the output.
//
// Marshalling the result rather than checking named fields is deliberate: it
// fails if someone later adds a value-bearing field to ItemSummary, which is
// exactly the regression worth catching and exactly the one a field-by-field
// assertion would miss.
func TestDecodeItemsCannotCarryAValue(t *testing.T) {
	const (
		pw    = "PASSWORD-must-not-survive-8f3a"
		fld   = "FIELDVALUE-must-not-survive-2b7c"
		totp  = "TOTPSEED-must-not-survive-91de"
		card  = "4111111111111111"
		notes = "NOTEBODY-must-not-survive-5c0f"
	)
	raw := itemsPayload(t, fmt.Sprintf(`{
	  "name":"loaded","folderId":"f1","revisionDate":"2026-09-21T10:00:00.000Z",
	  "notes":%q,
	  "login":{"username":"someone","password":%q,"totp":%q},
	  "card":{"number":%q},
	  "fields":[{"name":"api-key","value":%q}],
	  "passwordHistory":[{"password":%q}]
	}`, notes, pw, totp, card, fld, pw))

	got, err := decodeItems(raw, map[string]string{"f1": "Dotfiles/apps"})
	if err != nil {
		t.Fatalf("decodeItems: %v", err)
	}
	blob, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{pw, fld, totp, card, notes} {
		if strings.Contains(string(blob), secret) {
			t.Errorf("a secret value survived the projection: %q found in %s", secret, blob)
		}
	}

	// And the shape it DOES carry is right, or the test above passes vacuously
	// on a decoder that returns nothing at all.
	if len(got) != 1 {
		t.Fatalf("want 1 item, got %d", len(got))
	}
	it := got[0]
	if it.Name != "loaded" || it.Folder != "Dotfiles/apps" {
		t.Errorf("shape lost: name=%q folder=%q", it.Name, it.Folder)
	}
	if !it.HasNotes || !it.HasLogin || !it.HasUsername {
		t.Errorf("presence booleans lost: notes=%v login=%v username=%v", it.HasNotes, it.HasLogin, it.HasUsername)
	}
	if len(it.Fields) != 1 || it.Fields[0] != "api-key" {
		t.Errorf("field NAMES must survive, got %v", it.Fields)
	}
}

// An item with no login block must not read as having a username: `field:
// username` resolving against it would fail at runtime, and the report exists to
// say so beforehand.
func TestDecodeItemsDistinguishesAnAbsentLogin(t *testing.T) {
	raw := itemsPayload(t, `{"name":"noteonly","notes":"x","fields":[]}`)
	got, err := decodeItems(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].HasLogin || got[0].HasUsername {
		t.Errorf("an item with no login block claims one: %+v", got[0])
	}
	if !got[0].HasNotes {
		t.Error("a non-empty note must register")
	}
}

// An empty username is not a username. bw returns the login block with "" for an
// item that has only a password, and reporting that as present would send an
// operator looking for a value that is not there.
func TestDecodeItemsTreatsAnEmptyUsernameAsAbsent(t *testing.T) {
	raw := itemsPayload(t, `{"name":"pwonly","login":{"username":""},"fields":[]}`)
	got, err := decodeItems(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got[0].HasLogin {
		t.Error("the login block is present and must register")
	}
	if got[0].HasUsername {
		t.Error("an empty username must read as absent")
	}
}

// An unfoldered item resolves to "", not to the literal id, so a report never
// prints a UUID at a human.
func TestDecodeItemsResolvesFolderNamesAndLeavesUnfolderedEmpty(t *testing.T) {
	raw := itemsPayload(t, `{"name":"a","folderId":"f1","fields":[]},{"name":"b","fields":[]}`)
	got, err := decodeItems(raw, map[string]string{"f1": "Dotfiles/infra"})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Folder != "Dotfiles/infra" {
		t.Errorf("folder id not resolved: %q", got[0].Folder)
	}
	if got[1].Folder != "" {
		t.Errorf("unfoldered item must be empty, got %q", got[1].Folder)
	}
}

// A malformed timestamp on one row must not abort the inventory: the report is
// about layout, and refusing to produce it over one bad date would be the same
// "all or nothing" failure that makes health checks get skipped.
func TestDecodeItemsToleratesAnUnparseableRevisionDate(t *testing.T) {
	raw := itemsPayload(t, `{"name":"a","revisionDate":"not-a-date","fields":[]}`)
	got, err := decodeItems(raw, nil)
	if err != nil {
		t.Fatalf("one bad date aborted the whole inventory: %v", err)
	}
	if !got[0].Revised.IsZero() {
		t.Error("an unparseable date must yield the zero time, reported as unknown")
	}
}

// An unparseable payload reports a BYTE COUNT and never the body — this endpoint
// answers with the entire vault, so an error string that quoted it would be the
// leak the projection exists to prevent.
func TestDecodeItemsErrorNeverQuotesTheBody(t *testing.T) {
	const secret = "SECRET-must-not-appear-in-an-error"
	// Both error paths. The second one formats the byte count of the item array
	// itself, the bytes that carry every value, and round 3 of CLI-078's review
	// showed a mutation quoting it there survived when only the first was covered.
	for name, body := range map[string]string{
		"outer envelope unparseable": `{"data":[{"name":` + secret + `}]}`,
		"item array unparseable":     `{"object":"list","data":"` + secret + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := decodeItems(json.RawMessage(body), nil)
			if err == nil {
				t.Fatal("want an error for malformed item JSON")
			}
			if strings.Contains(err.Error(), secret) {
				t.Errorf("the error quoted the body: %v", err)
			}
			if !strings.Contains(err.Error(), "bytes") {
				t.Errorf("the error must say how much it could not parse: %v", err)
			}
		})
	}
}

func decl(secret, item, field, folder string, dormant bool) BWDecl {
	return BWDecl{Secret: secret, Var: secret, Item: item, Field: field, Folder: folder, Dormant: dormant}
}

func item(name, folder string, fields ...string) ItemSummary {
	return ItemSummary{Name: name, Folder: folder, Fields: fields}
}

func kinds(fs []LayoutFinding) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.Kind)
	}
	return out
}

// A declared folder that no folder carries BY NAME is the finding that motivated
// this check: ResolveFolder creates on miss, so the next `set` splits the store.
func TestLayoutDriftReportsAFolderNameNothingCarries(t *testing.T) {
	got := LayoutDrift(
		[]BWDecl{decl("A", "thing", "api-key", "apps", false)},
		[]ItemSummary{item("thing", "Dotfiles/apps", "api-key")},
		[]string{"Dotfiles/apps"},
	)
	if len(got) == 0 || got[0].Kind != DriftFolderMissing {
		t.Fatalf("want folder-missing first, got %v", kinds(got))
	}
	if !strings.Contains(got[0].Detail, "CREATE") {
		t.Errorf("the finding must say what happens next, got %q", got[0].Detail)
	}
}

// The case that hid for months: a declaration the read path does not use is still
// a declaration, and its target did not exist.
func TestLayoutDriftChecksDormantDeclarations(t *testing.T) {
	got := LayoutDrift(
		[]BWDecl{decl("GITHUB_PERSONAL_ACCESS_TOKEN", "github-cli-pat", "api-token", "Dotfiles/apps", true)},
		[]ItemSummary{item("something-else", "Dotfiles/apps")},
		[]string{"Dotfiles/apps"},
	)
	if len(got) != 1 || got[0].Kind != DriftItemMissing {
		t.Fatalf("a dormant declaration must still be checked, got %v", kinds(got))
	}
	if !strings.Contains(got[0].Detail, "dormant") {
		t.Errorf("the finding must say the read path does not use it yet: %q", got[0].Detail)
	}
}

// An item in the wrong folder is reported once, naming both places.
func TestLayoutDriftReportsAMisfiledItem(t *testing.T) {
	got := LayoutDrift(
		[]BWDecl{decl("D", "dockerhub", "PAT", "Dotfiles/apps", false)},
		[]ItemSummary{item("dockerhub", "", "PAT")},
		[]string{"Dotfiles/apps"},
	)
	if len(got) != 1 || got[0].Kind != DriftItemMisfiled {
		t.Fatalf("want one item-misfiled, got %v", kinds(got))
	}
	if !strings.Contains(got[0].Detail, "(no folder)") || !strings.Contains(got[0].Detail, "Dotfiles/apps") {
		t.Errorf("must name where it is AND where it was declared: %q", got[0].Detail)
	}
}

// hasField mirrors fieldFromItem's three special cases. If that dispatch ever
// changes, this is the test that should go red — it is the pin holding the
// deliberate duplication honest.
func TestLayoutDriftResolvesFieldsTheWayTheReaderDoes(t *testing.T) {
	it := ItemSummary{
		Name: "svc", Folder: "Dotfiles/apps",
		Fields: []string{"api-key"}, HasNotes: true, HasLogin: true, HasUsername: true,
	}
	for _, field := range []string{"api-key", "notes", "username", "password"} {
		got := LayoutDrift([]BWDecl{decl("S", "svc", field, "Dotfiles/apps", false)},
			[]ItemSummary{it}, []string{"Dotfiles/apps"})
		if len(got) != 0 {
			t.Errorf("field %q resolves at runtime but was reported missing: %v", field, got)
		}
	}
	// And the negative side, or the above passes on a checker that never reports.
	bare := ItemSummary{Name: "svc", Folder: "Dotfiles/apps"}
	for _, field := range []string{"api-key", "notes", "username", "password"} {
		got := LayoutDrift([]BWDecl{decl("S", "svc", field, "Dotfiles/apps", false)},
			[]ItemSummary{bare}, []string{"Dotfiles/apps"})
		if len(got) != 1 || got[0].Kind != DriftFieldMissing {
			t.Errorf("field %q is absent and must be reported, got %v", field, kinds(got))
		}
	}
}

// Seven vars naming one absent item is one absent item. Reporting per var turns
// a four-line report into a twenty-line one and buries the count.
func TestLayoutDriftReportsOneFindingPerProblemNotPerVar(t *testing.T) {
	var decls []BWDecl
	for i := 0; i < 7; i++ {
		decls = append(decls, decl(fmt.Sprintf("X_%d", i), "x-twitter-api", fmt.Sprintf("f%d", i), "Dotfiles/apps", false))
	}
	got := LayoutDrift(decls, nil, []string{"Dotfiles/apps"})
	if len(got) != 1 {
		t.Fatalf("want exactly 1 finding for 7 vars on one absent item, got %d: %v", len(got), kinds(got))
	}
}

// A missing item swallows its field checks: its fields cannot be inspected, and
// saying so per var would bury the one fact that matters.
func TestLayoutDriftDoesNotReportFieldsOfAnAbsentItem(t *testing.T) {
	got := LayoutDrift(
		[]BWDecl{decl("A", "ghost", "api-key", "Dotfiles/apps", false)},
		nil, []string{"Dotfiles/apps"},
	)
	if len(got) != 1 || got[0].Kind != DriftItemMissing {
		t.Fatalf("want only item-missing, got %v", kinds(got))
	}
}

// Findings come out in the order they must be fixed: a folder cannot hold an item
// before it exists, and a field cannot be checked on an item that is absent.
func TestLayoutDriftOrdersFindingsByFixOrder(t *testing.T) {
	got := LayoutDrift([]BWDecl{
		decl("C", "present", "missing-field", "Dotfiles/apps", false),
		decl("B", "absent", "api-key", "Dotfiles/apps", false),
		decl("A", "present", "api-key", "nowhere", false),
	}, []ItemSummary{item("present", "Dotfiles/apps", "api-key")}, []string{"Dotfiles/apps"})

	// Four, not three: declaring "present" into the absent folder "nowhere" makes
	// it misfiled as well as making the folder missing. Both are true and both are
	// reported — what this pins is that the two ITEM findings sit together between
	// the folder one and the field one.
	want := []string{DriftFolderMissing, DriftItemMissing, DriftItemMisfiled, DriftFieldMissing}
	gotKinds := kinds(got)
	if len(gotKinds) != len(want) {
		t.Fatalf("want %v, got %v", want, gotKinds)
	}
	for i := range want {
		if gotKinds[i] != want[i] {
			t.Fatalf("want %v, got %v", want, gotKinds)
		}
	}
}

// A store that matches its declaration reports nothing. Without this the whole
// suite could pass on a function that always finds something.
func TestLayoutDriftIsSilentOnAMatchingStore(t *testing.T) {
	got := LayoutDrift(
		[]BWDecl{decl("A", "svc", "api-key", "Dotfiles/apps", false)},
		[]ItemSummary{item("svc", "Dotfiles/apps", "api-key")},
		[]string{"Dotfiles/apps"},
	)
	if len(got) != 0 {
		t.Errorf("a matching store must produce no findings, got %v", got)
	}
}

// Unmanaged items are counted, never reported as findings: a personal vault
// legitimately holds what this repo does not manage.
func TestUnmanagedItemsListsOnlyWhatNoDeclarationNames(t *testing.T) {
	got := UnmanagedItems(
		[]BWDecl{decl("A", "svc", "api-key", "Dotfiles/apps", false)},
		[]ItemSummary{item("svc", "Dotfiles/apps"), item("bank", ""), item("airline", "")},
	)
	if len(got) != 2 || got[0] != "airline" || got[1] != "bank" {
		t.Errorf("want [airline bank] sorted, got %v", got)
	}
}

// BWDeclarations must walk age-backed secrets too. Tested through ParseRegistry
// rather than by handing LayoutDrift a struct, because the bug this guards is in
// the WALK: an earlier version filtered on `Backend != BackendBW` and so never
// emitted a dormant declaration at all, while a hand-built fixture with
// Dormant:true went on passing. Mutation testing found that hole, not review.
func TestBWDeclarationsIncludesDormantBlocksOnAgeBackedSecrets(t *testing.T) {
	const yml = "version: 1\nsecrets:\n" +
		"  - {id: LIVE, plane: app, backend: bw, bw: {item: live-item, field: api-key, folder: Dotfiles/apps}, expose: {env: LIVE}}\n" +
		"  - {id: DORMANT, plane: app, backend: age, age: some.blob, bw: {item: future-item, field: api-token, folder: Dotfiles/apps}, expose: {env: DORMANT}}\n" +
		"  - {id: NOBW, plane: app, backend: age, age: other.blob, expose: {env: NOBW}}\n"
	reg, err := ParseRegistry([]byte(yml))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	got := reg.BWDeclarations()
	if len(got) != 2 {
		t.Fatalf("want 2 declarations (the live one and the dormant one), got %d: %+v", len(got), got)
	}
	byItem := map[string]BWDecl{}
	for _, d := range got {
		byItem[d.Item] = d
	}
	if d, ok := byItem["live-item"]; !ok || d.Dormant {
		t.Errorf("a bw-backed secret must be declared live, got %+v", d)
	}
	if d, ok := byItem["future-item"]; !ok || !d.Dormant {
		t.Errorf("an age-backed secret's bw block must be declared DORMANT, got %+v", d)
	}
	if _, ok := byItem[""]; ok {
		t.Error("a secret with no bw block must contribute no declaration")
	}
}

// Per-var field overrides are resolved by the walk. Without this, the seven X_*
// vars would all be checked against the secret-level field and six would be
// reported missing on an item that carries every one of them.
func TestBWDeclarationsResolvesPerVarFieldOverrides(t *testing.T) {
	const yml = "version: 1\nsecrets:\n" +
		"  - {id: MULTI, plane: app, backend: bw, bw: {item: multi, field: fallback, folder: Dotfiles/apps}," +
		" expose: {env: {A: {field: a-field}, B: {}}}}\n"
	reg, err := ParseRegistry([]byte(yml))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	got := reg.BWDeclarations()
	if len(got) != 2 {
		t.Fatalf("want one declaration per var, got %d", len(got))
	}
	byVar := map[string]string{}
	for _, d := range got {
		byVar[d.Var] = d.Field
	}
	if byVar["A"] != "a-field" {
		t.Errorf("a per-var field override must win, got %q", byVar["A"])
	}
	if byVar["B"] != "fallback" {
		t.Errorf("a var with no override must fall back to the secret's field, got %q", byVar["B"])
	}
}

// Dedupe applies to MISFILED as well as MISSING. Seven vars on one item in the
// wrong folder is one item in the wrong folder; the first version deduped only
// the missing case and mutation testing walked straight through it.
func TestLayoutDriftDedupesAMisfiledItemAcrossVars(t *testing.T) {
	var decls []BWDecl
	for i := 0; i < 7; i++ {
		decls = append(decls, decl(fmt.Sprintf("X_%d", i), "x-twitter-api", fmt.Sprintf("f%d", i), "Dotfiles/apps", false))
	}
	got := LayoutDrift(decls,
		[]ItemSummary{item("x-twitter-api", "", "f0", "f1", "f2", "f3", "f4", "f5", "f6")},
		[]string{"Dotfiles/apps"})
	if len(got) != 1 || got[0].Kind != DriftItemMisfiled {
		t.Fatalf("want exactly 1 item-misfiled for 7 vars, got %d: %v", len(got), kinds(got))
	}
}

// bw serve lists a pseudo-folder, "No Folder", with a null id, and every
// unfoldered item carries a null folderId. Indexing folders by id without
// skipping it maps "" to "No Folder", so every unfoldered item reads as filed in a
// folder of that name. ListFolders already dropped it; the index ListItems built
// did not. Measured on the first live run: `dockerhub is in No Folder`.
func TestFolderIndexExcludesTheNoFolderPseudoFolder(t *testing.T) {
	folders, err := decodeFolders(json.RawMessage(
		`{"object":"list","data":[{"object":"folder","id":null,"name":"No Folder"},{"object":"folder","id":"f1","name":"Dotfiles/apps"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeItems(itemsPayload(t, `{"name":"loose","folderId":null,"fields":[]}`), folderIndex(folders))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Folder != "" {
		t.Errorf("an unfoldered item must resolve to \"\", got %q", got[0].Folder)
	}
}

// A secret exposed as a FILE declares a bw target exactly as an env secret does;
// only the consumer contract differs. The walk read expose.env alone, so eight
// file-exposed secrets (KUBECONFIG, SSH_KEY, the recovery codes) were never
// compared against the store at all.
func TestBWDeclarationsIncludesFileExposedSecrets(t *testing.T) {
	const yml = "version: 1\nsecrets:\n" +
		"  - {id: KCFG, plane: infra, backend: bw, bw: {item: kube-item, field: notes, folder: Dotfiles/infra}, expose: {file: {var: KUBECONFIG, path: \"~/.kube/x\"}}}\n"
	reg, err := ParseRegistry([]byte(yml))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	got := reg.BWDeclarations()
	if len(got) != 1 {
		t.Fatalf("want 1 declaration for the file-exposed secret, got %d: %+v", len(got), got)
	}
	want := BWDecl{Secret: "KCFG", Var: "KUBECONFIG", Item: "kube-item", Field: "notes", Folder: "Dotfiles/infra"}
	if got[0] != want {
		t.Errorf("got %+v, want %+v", got[0], want)
	}
}

// A declaration with no folder states no placement: the taxonomy covers the app
// and infra planes only, and the personal plane's is deferred (#586). Reading ""
// as "must be unfoldered" would report every personal item the operator filed by
// hand as misfiled, and hand reconcile an instruction to unfile it.
func TestLayoutDriftLeavesAnUndeclaredFolderUngoverned(t *testing.T) {
	got := LayoutDrift(
		[]BWDecl{decl("G", "gmail-backup-code", "notes", "", false)},
		[]ItemSummary{{Name: "gmail-backup-code", Folder: "Personal", HasNotes: true}},
		[]string{"Personal"},
	)
	if len(got) != 0 {
		t.Errorf("an item with no declared folder must not be reported misfiled, got %v", got)
	}
}

// Bitwarden allows two items with one name, and the reader refuses to pick between
// them (TestBWServeReader_Field_AmbiguousNameErrors). A by-name index kept only the
// last one, so a declared item could be judged by a same-named personal item —
// misfiled, missing fields — while the real one sat correct. Ambiguity is its own
// finding, raised instead of any judgement about either item (round 2, Blocker).
func TestLayoutDriftReportsAnAmbiguousItemInsteadOfJudgingEither(t *testing.T) {
	got := LayoutDrift(
		[]BWDecl{decl("D", "dockerhub", "PAT", "Dotfiles/apps", false)},
		[]ItemSummary{item("dockerhub", "Dotfiles/apps", "PAT"), item("dockerhub", "Personal")},
		[]string{"Dotfiles/apps", "Personal"},
	)
	if len(got) != 1 || got[0].Kind != DriftItemAmbiguous {
		t.Fatalf("want exactly one item-ambiguous, got %v", kinds(got))
	}
	if !strings.Contains(got[0].Detail, "2 items") {
		t.Errorf("the finding must say how many items share the name: %q", got[0].Detail)
	}
}

// One finding per problem (AC5), for fields too: two vars declaring the same absent
// field on one item are one absent field.
func TestLayoutDriftDedupesAMissingFieldAcrossVars(t *testing.T) {
	got := LayoutDrift(
		[]BWDecl{decl("A", "svc", "key", "", false), decl("B", "svc", "key", "", false)},
		[]ItemSummary{item("svc", "")},
		nil,
	)
	if len(got) != 1 || got[0].Kind != DriftFieldMissing {
		t.Fatalf("want one field-missing, got %v", kinds(got))
	}
}

// CLI-078 review round 3, Major: a declaration that names no field was counted
// and never checked. hasField("") answered true while fieldFromItem("") refuses,
// so drift certified as converged a target the reader cannot resolve. The live
// registry carried exactly one (AGE_KEY_PERSONAL's bw copy).
func TestLayoutDriftReportsADeclarationThatNamesNoField(t *testing.T) {
	it := item("AGE-SECRET-KEY-PERSONAL", "")
	it.HasNotes = true
	got := LayoutDrift([]BWDecl{decl("AGE_KEY_PERSONAL", "AGE-SECRET-KEY-PERSONAL", "", "", true)},
		[]ItemSummary{it}, nil)
	if len(got) != 1 || got[0].Kind != DriftFieldMissing {
		t.Fatalf("a field-less declaration must be a finding, got %v", kinds(got))
	}
	if !strings.Contains(got[0].Detail, "no field") {
		t.Errorf("the finding must say no field is declared, not that the item lacks one: %q", got[0].Detail)
	}
	// And the reader really does refuse it: this is the agreement the finding rests on.
	if _, err := fieldFromItem([]byte(`{"name":"AGE-SECRET-KEY-PERSONAL","notes":"x"}`), ""); err == nil {
		t.Fatal("fieldFromItem accepted an empty field; the finding above would be wrong")
	}
}

// AC4, pinned as AGREEMENT rather than as hasField alone. CLI-078 review round 3
// showed the older test passing while fieldFromItem's notes dispatch was inverted:
// it never ran the reader. Each row hands ONE raw item to both sides — the reader
// that resolves values, and the value-free projection drift judges — and requires
// them to agree on whether the field yields a usable (non-empty) value.
//
// Two rows diverge on purpose, and say why. The projection never decodes a
// password or a custom field's value (that is what keeps it value-free), so an
// empty one there reads as present. Both divergences lean the same way: drift may
// miss an empty value, and it never reports a usable one as missing.
func TestLayoutDriftAgreesWithTheReaderOnEveryItemShape(t *testing.T) {
	rows := []struct {
		raw, field string
		exception  string // non-empty: the documented reason the two sides differ
	}{
		{`{"name":"svc","notes":"n"}`, "notes", ""},
		{`{"name":"svc","notes":""}`, "notes", ""},
		{`{"name":"svc"}`, "notes", ""},
		{`{"name":"svc","login":{"username":"u","password":"p"}}`, "username", ""},
		{`{"name":"svc","login":{"username":""}}`, "username", ""},
		{`{"name":"svc"}`, "username", ""},
		{`{"name":"svc","login":{"password":"p"}}`, "password", ""},
		{`{"name":"svc","login":{"password":""}}`, "password", "the password is never decoded; login presence is the value-free answer"},
		{`{"name":"svc"}`, "password", ""},
		{`{"name":"svc","fields":[{"name":"k","value":"v"}]}`, "k", ""},
		{`{"name":"svc","fields":[{"name":"k","value":""}]}`, "k", "custom field values are never decoded; a named field is taken as present"},
		{`{"name":"svc"}`, "k", ""},
		{`{"name":"svc","notes":"n"}`, "", ""},
	}
	for _, r := range rows {
		value, err := fieldFromItem([]byte(r.raw), r.field)
		usable := err == nil && value != ""

		items, derr := decodeItems(json.RawMessage(`{"object":"list","data":[`+r.raw+`]}`), nil)
		if derr != nil || len(items) != 1 {
			t.Fatalf("%s: decode: %v", r.raw, derr)
		}
		present := hasField(items[0], r.field)

		switch {
		case r.exception == "" && present != usable:
			t.Errorf("%s field %q: drift says present=%v, the reader says usable=%v", r.raw, r.field, present, usable)
		case r.exception != "" && (!present || usable):
			t.Errorf("%s field %q: the documented divergence (%s) no longer holds; update the row", r.raw, r.field, r.exception)
		}
	}
}
