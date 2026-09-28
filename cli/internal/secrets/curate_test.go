package secrets

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// fakeCurateStore holds whole item JSON, keyed by id, and answers every seam
// curate reads or writes. Its inventory goes through the real projection
// (decodeItems) and its edits through the real mutation (applyMutation), so the
// tests exercise the code the live store runs, not a model of it.
type fakeCurateStore struct {
	items   map[string]map[string]any
	folders map[string]string // id -> name
	writes  int
}

func newCurateStore(items ...string) *fakeCurateStore {
	s := &fakeCurateStore{items: map[string]map[string]any{}, folders: map[string]string{"fh": "Homelab"}}
	for _, raw := range items {
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			panic(fmt.Sprintf("fixture %s: %v", raw, err))
		}
		s.items[m["id"].(string)] = m
	}
	return s
}

func (s *fakeCurateStore) inventory(t *testing.T) []ItemSummary {
	t.Helper()
	ids := make([]string, 0, len(s.items))
	for id := range s.items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var parts []string
	for _, id := range ids {
		b, _ := json.Marshal(s.items[id])
		parts = append(parts, string(b))
	}
	got, err := decodeItems(itemsPayload(t, strings.Join(parts, ",")), s.folders)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func (s *fakeCurateStore) Field(item, field string) (string, error) {
	m, ok := s.items[item]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrBWItemNotFound, item)
	}
	b, _ := json.Marshal(m)
	return fieldFromItem(b, field)
}

func (s *fakeCurateStore) Curate(id string, mu Mutation) error {
	s.writes++
	b, _ := json.Marshal(s.items[id])
	out, err := applyMutation(b, mu)
	if err != nil {
		return err
	}
	var m map[string]any
	_ = json.Unmarshal(out, &m)
	// Every edit bumps the revision date, as Bitwarden does.
	m["revisionDate"] = fmt.Sprintf("2026-09-27T20:00:%02d.000Z", s.writes)
	s.items[id] = m
	return nil
}

func (s *fakeCurateStore) DeleteItem(id string) error {
	s.writes++
	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("%w: %s", ErrBWItemNotFound, id)
	}
	delete(s.items, id)
	return nil
}

func (s *fakeCurateStore) ResolveFolder(name string) (string, error) {
	for id, n := range s.folders {
		if n == name {
			return id, nil
		}
	}
	s.writes++
	id := "f-" + name
	s.folders[id] = name
	return id, nil
}

const (
	curatePass  = "PASS-sentinel-never-printed-31c9"
	curateOther = "OTHER-sentinel-never-printed-8e20"
	curateUser  = "USER-sentinel-never-printed-a4f1"
)

func login(id, user, pass string, extra string) string {
	return fmt.Sprintf(`{"id":%q,"name":"n-%s","type":1,"revisionDate":"2026-09-01T00:00:00.000Z",
	  "login":{"username":%q,"password":%q,"uris":[{"uri":"https://%s.example"}]}%s}`, id, id, user, pass, id, extra)
}

func mustParse(t *testing.T, plan string) []CurateRow {
	t.Helper()
	rows, err := ParseCuratePlan(strings.NewReader(plan))
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func planOn(t *testing.T, s *fakeCurateStore, plan string, cleared ...string) CuratePlan {
	t.Helper()
	c := map[string]bool{}
	for _, g := range cleared {
		c[g] = true
	}
	return PlanCurate(mustParse(t, plan), CurateInputs{
		Items: s.inventory(t), Values: s, Cleared: c,
		Owned: func(name string) bool { return name == "registry-item" },
	})
}

func TestParseCuratePlan(t *testing.T) {
	rows := mustParse(t, "# comment\n\n"+
		"folder\taaaa1111\tHomelab\thomelab host\n"+
		"merge-delete\tbbbb\tcccc\tsame account\talias,drop=notes,gate=g1\n"+
		"hide\tdddd:api-key\t-\tsecret-named\n")
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	if r := rows[1]; !r.Alias || len(r.Drop) != 1 || r.Drop[0] != "notes" || r.Gate != "g1" || r.Line != 4 {
		t.Errorf("flags lost: %+v", r)
	}
	if r := rows[2]; r.Target != "dddd" || r.Field != "api-key" || r.Arg != "" {
		t.Errorf("id:field not split: %+v", r)
	}

	for _, bad := range []string{
		"explode\taaaa\t-\twhy",               // unknown op
		"delete\taaaa\t-\t",                   // no reason
		"hide\taaaa\t-\twhy",                  // hide needs id:field
		"folder\taaaa\t-\twhy",                // folder needs a folder
		"delete\taaaa\t-\twhy\tbogus",         // unknown flag
		"delete\taaaa\t-\twhy\talias",         // alias on a non-merge
		"delete-field\taaaa:password\t-\twhy", // curate removes custom fields only
		"delete\taaaa",                        // too few columns
	} {
		if _, err := ParseCuratePlan(strings.NewReader(bad)); err == nil {
			t.Errorf("want a parse error for %q", bad)
		}
	}
}

// AC1 + AC3: every op plans `apply` on a store that needs it, the apply converges
// it, and the same plan against the result is all `done`, with no write.
func TestCurateAppliesEveryOpAndConverges(t *testing.T) {
	s := newCurateStore(
		login("aaaa0001", curateUser, curatePass, `,"fields":[{"name":"token","type":0,"value":"x"},{"name":"old","type":0,"value":"y"}]`),
		login("bbbb0002", curateUser, curatePass, ""),  // duplicate of aaaa0001
		login("cccc0003", "u2", curateOther, ""),       // to delete
		login("dddd0004", curateUser, curateOther, ""), // username source
	)
	plan := "folder\taaaa0001\tHomelab\tr\n" +
		"hide\taaaa0001:token\t-\tr\n" +
		"reprompt\taaaa0001\t-\tr\n" +
		"delete-field\taaaa0001:old\t-\tr\n" +
		"add-uri\taaaa0001\thttps://extra.example\tr\n" +
		"copy-username\taaaa0001:alias\tdddd0004\tr\n" +
		"merge-delete\tbbbb0002\taaaa0001\tr\n" +
		"delete\tcccc0003\t-\tr\n"

	p := planOn(t, s, plan)
	if got := p.Count(CurateApply); got != 8 {
		for _, st := range p.Steps {
			t.Logf("%s %s %s", st.State, st.Row.Op, st.Detail)
		}
		t.Fatalf("want 8 to apply, got %d", got)
	}
	if s.writes != 0 {
		t.Fatalf("planning wrote %d time(s)", s.writes)
	}
	if err := ApplyCurate(p, s, nil); err != nil {
		t.Fatal(err)
	}

	keeper := s.inventory(t)
	var k ItemSummary
	for _, it := range keeper {
		if it.ID == "aaaa0001" {
			k = it
		}
	}
	if k.Folder != "Homelab" || !k.Reprompt || k.FieldTypes["token"] != 1 || k.FieldTypes["alias"] != 0 {
		t.Errorf("keeper not curated: %+v", k)
	}
	if _, has := k.FieldTypes["old"]; has {
		t.Error("delete-field left the field")
	}
	for _, u := range []string{"https://extra.example", "https://bbbb0002.example"} {
		if !contains(k.URIs, u) {
			t.Errorf("keeper lacks %s: %v", u, k.URIs)
		}
	}
	if v, _ := s.Field("aaaa0001", "alias"); v != curateUser {
		t.Error("copy-username did not store the source's username")
	}
	if _, ok := s.items["bbbb0002"]; ok {
		t.Error("merge-delete left the duplicate")
	}
	if _, ok := s.items["cccc0003"]; ok {
		t.Error("delete left the item")
	}

	before := s.writes
	again := planOn(t, s, plan)
	if again.Count(CurateDone) != 8 {
		for _, st := range again.Steps {
			t.Logf("%s %s %s", st.State, st.Row.Op, st.Detail)
		}
		t.Fatal("a second plan of the same rows must be all done")
	}
	if err := ApplyCurate(again, s, nil); err != nil || s.writes != before {
		t.Errorf("a converged plan must write nothing: err=%v writes=%d", err, s.writes-before)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// AC2: each blocking rule, alone.
func TestCurateBlocks(t *testing.T) {
	passkey := `,"login":{"username":"u","password":"p","fido2Credentials":[{"credentialId":"c"}]}`
	cases := []struct {
		name, plan, want string
		items            []string
		cleared          []string
	}{
		{"passkey item is never edited", "reprompt\taaaa0001\t-\tr", "passkey",
			[]string{`{"id":"aaaa0001","type":1` + passkey + `}`}, nil},
		{"passkey item is never deleted", "delete\taaaa0001\t-\tr", "passkey",
			[]string{`{"id":"aaaa0001","type":1` + passkey + `}`}, nil},
		{"attachments block a delete", "delete\taaaa0001\t-\tr", "attachment",
			[]string{`{"id":"aaaa0001","type":2,"attachments":[{"id":"x"}]}`}, nil},
		{"merge passwords differ", "merge-delete\taaaa0001\tbbbb0002\tr", "passwords differ",
			[]string{login("aaaa0001", "u", curatePass, ""), login("bbbb0002", "u", curateOther, "")}, nil},
		{"merge usernames differ", "merge-delete\taaaa0001\tbbbb0002\tr", "usernames differ",
			[]string{login("aaaa0001", "u1", curatePass, ""), login("bbbb0002", "u2", curatePass, "")}, nil},
		{"duplicate with attachments", "merge-delete\taaaa0001\tbbbb0002\tr", "attachment",
			[]string{login("aaaa0001", "u", "p", `,"attachments":[{"id":"x"}]`), login("bbbb0002", "u", "p", "")}, nil},
		{"duplicate with TOTP", "merge-delete\taaaa0001\tbbbb0002\tr", "TOTP",
			[]string{`{"id":"aaaa0001","type":1,"login":{"username":"u","password":"p","totp":"seed"}}`, login("bbbb0002", "u", "p", "")}, nil},
		{"duplicate fields not dropped", "merge-delete\taaaa0001\tbbbb0002\tr", "custom field",
			[]string{login("aaaa0001", "u", "p", `,"fields":[{"name":"k","type":0,"value":"v"}]`), login("bbbb0002", "u", "p", "")}, nil},
		{"duplicate notes not dropped", "merge-delete\taaaa0001\tbbbb0002\tr", "notes",
			[]string{login("aaaa0001", "u", "p", `,"notes":"n"`), login("bbbb0002", "u", "p", "")}, nil},
		{"duplicate with a passkey", "merge-delete\taaaa0001\tbbbb0002\tr", "passkey",
			[]string{`{"id":"aaaa0001","type":1` + passkey + `}`, login("bbbb0002", "u", "p", "")}, nil},
		{"passkey keeper needing a URI", "merge-delete\taaaa0001\tbbbb0002\tr", "keeper carries a passkey",
			[]string{login("aaaa0001", "u", "p", ""), `{"id":"bbbb0002","type":1,"login":{"username":"u","password":"p","uris":[],"fido2Credentials":[{"c":1}]}}`}, nil},
		{"registry-owned item", "add-uri\taaaa0001\thttps://x\tr", "registry",
			[]string{`{"id":"aaaa0001","name":"registry-item","type":1}`}, nil},
		{"Dotfiles folders are the registry's", "folder\taaaa0001\tDotfiles/apps\tr", "registry",
			[]string{login("aaaa0001", "u", "p", "")}, nil},
		{"ambiguous prefix", "reprompt\taaaa\t-\tr", "more than one",
			[]string{login("aaaa0001", "u", "p", ""), login("aaaa0002", "u", "p", "")}, nil},
		{"absent edit target", "reprompt\taaaa0001\t-\tr", "absent", nil, nil},
		{"gate not cleared", "delete\taaaa0001\t-\tr\tgate=pexels", "gate", []string{login("aaaa0001", "u", "p", "")}, nil},
		{"copy never overwrites", "copy-username\taaaa0001:alias\tbbbb0002\tr", "never overwrites",
			[]string{login("aaaa0001", "u", "p", `,"fields":[{"name":"alias","type":0,"value":"other"}]`), login("bbbb0002", "src", "p", "")}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCurateStore(c.items...)
			p := planOn(t, s, c.plan, c.cleared...)
			st := p.Steps[0]
			if st.State != CurateBlocked || !strings.Contains(st.Detail, c.want) {
				t.Fatalf("want blocked with %q, got %s: %s", c.want, st.State, st.Detail)
			}
			if err := ApplyCurate(p, s, nil); err == nil || s.writes != 0 {
				t.Errorf("a blocked plan must refuse whole: err=%v writes=%d", err, s.writes)
			}
		})
	}
}

// hide and reprompt change presentation only, which the registry does not govern,
// so they apply to a registry-declared item that every other op refuses.
func TestCuratePresentationOpsReachRegistryItems(t *testing.T) {
	s := newCurateStore(`{"id":"aaaa0001","name":"registry-item","type":1,"login":{},
	  "fields":[{"name":"PAT","type":0,"value":"v"}]}`)
	p := planOn(t, s, "hide\taaaa0001:PAT\t-\tr\nreprompt\taaaa0001\t-\tr\n")
	if p.Count(CurateApply) != 2 {
		t.Fatalf("hide and reprompt must plan on a registry item: %+v", p.Steps)
	}
}

// drop=uris is the author's claim that the duplicate's URIs add nothing (an
// equivalent host under base-domain matching). Nothing is carried, so a keeper
// with a passkey needs no edit and the merge can proceed.
func TestCurateDropURIsLetsAPasskeyKeeperStandUntouched(t *testing.T) {
	s := newCurateStore(login("aaaa0001", "u", "p", ""),
		`{"id":"bbbb0002","type":1,"login":{"username":"u","password":"p","uris":[],"fido2Credentials":[{"c":1}]}}`)
	p := planOn(t, s, "merge-delete\taaaa0001\tbbbb0002\tr\tdrop=uris")
	if st := p.Steps[0]; st.State != CurateApply || len(st.Carry) != 0 {
		t.Fatalf("want apply with nothing carried, got %s (%d carried): %s", st.State, len(st.Carry), st.Detail)
	}
	if err := ApplyCurate(p, s, nil); err != nil || s.writes != 1 {
		t.Fatalf("want the delete alone: err=%v writes=%d", err, s.writes)
	}
}

func TestCurateGateClears(t *testing.T) {
	s := newCurateStore(login("aaaa0001", "u", "p", ""))
	p := planOn(t, s, "delete\taaaa0001\t-\tr\tgate=pexels", "pexels")
	if p.Steps[0].State != CurateApply {
		t.Fatalf("a cleared gate must let the row apply: %s", p.Steps[0].Detail)
	}
}

// AC4: the digest pins the store state the plan was made against.
func TestCurateDigestTracksRevisions(t *testing.T) {
	s := newCurateStore(login("aaaa0001", "u", "p", ""), login("bbbb0002", "u", "p", ""))
	plan := "reprompt\taaaa0001\t-\tr\n"
	d1 := planOn(t, s, plan).Digest
	if d2 := planOn(t, s, plan).Digest; d1 != d2 {
		t.Fatal("the digest must be deterministic")
	}
	s.items["bbbb0002"]["revisionDate"] = "2026-09-27T21:00:00.000Z" // an untouched item
	if planOn(t, s, plan).Digest != d1 {
		t.Error("an item the plan does not touch must not change the digest")
	}
	s.items["aaaa0001"]["revisionDate"] = "2026-09-27T21:00:00.000Z"
	if planOn(t, s, plan).Digest == d1 {
		t.Error("an edit to a touched item must change the digest")
	}
}

func TestTotalPasskeys(t *testing.T) {
	s := newCurateStore(`{"id":"a","type":1,"login":{"fido2Credentials":[{},{}]}}`, `{"id":"b","type":1,"login":{}}`)
	if n := TotalPasskeys(s.inventory(t)); n != 2 {
		t.Errorf("want 2, got %d", n)
	}
}
