package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// Sentinel values a curate precondition reads. None may reach any output.
const (
	curateSecret  = "sentinel-a-cmd-curate-never-print-5a1f"
	curateAccount = "sentinel-b-cmd-curate-never-print-0c3e"
	curateURI     = "https://token-in-query.example/?t=URI-never-print-77b2"
)

// curateVault models the store as curate sees it: item shapes, the values a
// precondition compares, and the writes. It answers every seam the command uses.
// Every item holds the same login, so any merge's comparisons read equal.
type curateVault struct {
	items   map[string]*secrets.ItemSummary
	folders map[string]string // id -> name
	writes  int
	// losePasskey makes the next write drop one passkey somewhere else in the
	// vault: the loss the before/after count exists to catch.
	losePasskey bool
}

func newCurateVault() *curateVault {
	v := &curateVault{items: map[string]*secrets.ItemSummary{}, folders: map[string]string{}}
	add := func(id, name string, passkeys int) {
		v.items[id] = &secrets.ItemSummary{ID: id, Name: name, Type: 1, HasLogin: true, HasUsername: true,
			FieldTypes: map[string]int{}, Passkeys: passkeys, URIs: []string{curateURI},
			Revised: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	}
	add("aaaa0001-keeper", "shop", 0)
	add("bbbb0002-dup", "shop", 0)
	add("cccc0003-pk", "bank", 3)
	add("dddd0004-reg", "dockerhub", 0)
	return v
}

func (v *curateVault) Sync() error                    { return nil }
func (v *curateVault) ListFolders() ([]string, error) { return nil, nil }
func (v *curateVault) ListItems() ([]secrets.ItemSummary, error) {
	var out []secrets.ItemSummary
	for _, it := range v.items {
		c := *it
		c.URIs = slices.Clone(it.URIs)
		out = append(out, c)
	}
	return out, nil
}

func (v *curateVault) Field(item, field string) (string, error) {
	if _, ok := v.items[item]; ok {
		switch field {
		case "username":
			return curateAccount, nil
		case "password":
			return curateSecret, nil
		}
	}
	return "", secrets.ErrBWItemNotFound
}

func (v *curateVault) touch(id string) {
	v.writes++
	v.items[id].Revised = v.items[id].Revised.Add(time.Minute)
	if v.losePasskey {
		v.items["cccc0003-pk"].Passkeys--
		v.losePasskey = false
	}
}

func (v *curateVault) Curate(id string, m secrets.Mutation) error {
	it := v.items[id]
	switch m.Kind {
	case secrets.MutReprompt:
		it.Reprompt = true
	case secrets.MutAddURI:
		it.URIs = append(it.URIs, m.Value)
	case secrets.MutMove:
		it.Folder = v.folders[m.Value]
	default:
		return errors.New("unmodelled mutation")
	}
	v.touch(id)
	return nil
}

func (v *curateVault) DeleteItem(id string) error {
	v.writes++
	delete(v.items, id)
	return nil
}

func (v *curateVault) ResolveFolder(name string) (string, error) {
	v.folders["f-"+name] = name
	return "f-" + name, nil
}

func (v *curateVault) SetField(string, string, string) error           { return errors.New("unused") }
func (v *curateVault) CreateItem(string, string, string, string) error { return errors.New("unused") }
func (v *curateVault) MoveItem(string, string) error                   { return errors.New("unused") }
func (v *curateVault) RemoveField(string, string) error                { return errors.New("unused") }

const curateRegistry = `
version: 1
secrets:
  - {id: DOCKERHUB_TOKEN, plane: app, backend: bw, bw: {item: dockerhub, field: PAT, folder: Dotfiles/apps}, expose: {env: DOCKERHUB_TOKEN}}
`

func runCurate(t *testing.T, v *curateVault, plan string, args ...string) (string, error) {
	t.Helper()
	useTempRegistry(t, curateRegistry)
	path := filepath.Join(t.TempDir(), "plan.tsv")
	if err := os.WriteFile(path, []byte(plan), 0o600); err != nil {
		t.Fatal(err)
	}
	oldL, oldR, oldW := bwLister, bwReader, bwWriter
	t.Cleanup(func() { bwLister, bwReader, bwWriter = oldL, oldR, oldW })
	bwLister, bwReader, bwWriter = v, v, v

	c := newSecretsCurateCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(append([]string{"--plan", path}, args...))
	err := c.Execute()
	if err != nil {
		out.WriteString("\nERR: " + err.Error())
	}
	return out.String(), err
}

func assertNoValue(t *testing.T, out string) {
	t.Helper()
	for _, s := range []string{curateSecret, curateAccount, curateURI, "URI-never-print", "shop", "bank"} {
		if strings.Contains(out, s) {
			t.Fatalf("output carries %q, a value or a personal name:\n%s", s, out)
		}
	}
}

var digestRE = regexp.MustCompile(`--digest ([0-9a-f]{16})`)

const curatePlan = "merge-delete\tbbbb0002\taaaa0001\tsame account\n" +
	"reprompt\taaaa0001\t-\tsensitive\n" +
	"folder\taaaa0001\tHomelab\thome host\n"

// AC1 + AC3 + AC5: the dry run writes nothing and prints a digest; --apply with it
// converges; nothing printed on the way carries a value or an item name.
func TestCurateDryRunThenApplyWithTheDigest(t *testing.T) {
	v := newCurateVault()
	out, err := runCurate(t, v, curatePlan)
	if err != nil || v.writes != 0 {
		t.Fatalf("dry run: err=%v writes=%d\n%s", err, v.writes, out)
	}
	assertNoValue(t, out)
	m := digestRE.FindStringSubmatch(out)
	if m == nil || !strings.Contains(out, "Plan: 3 to apply") {
		t.Fatalf("the dry run must print the plan and its digest:\n%s", out)
	}

	out, err = runCurate(t, v, curatePlan, "--apply", "--digest", m[1])
	if err != nil {
		t.Fatalf("apply: %v\n%s", err, out)
	}
	assertNoValue(t, out)
	if !strings.Contains(out, "Converged: 3 row(s) applied") {
		t.Errorf("apply must confirm convergence:\n%s", out)
	}
	if _, dup := v.items["bbbb0002-dup"]; dup || !v.items["aaaa0001-keeper"].Reprompt {
		t.Errorf("store not curated: %+v", v.items)
	}

	before := v.writes
	out, err = runCurate(t, v, curatePlan)
	if err != nil || v.writes != before || !strings.Contains(out, "Plan: 0 to apply, 3 done") {
		t.Errorf("a second plan must be all done: err=%v\n%s", err, out)
	}
}

// AC4: --apply refuses a missing digest and one made before the store changed.
func TestCurateApplyRefusesAStaleOrMissingDigest(t *testing.T) {
	v := newCurateVault()
	out, _ := runCurate(t, v, curatePlan)
	d := digestRE.FindStringSubmatch(out)[1]

	if out, err := runCurate(t, v, curatePlan, "--apply"); err == nil || v.writes != 0 {
		t.Fatalf("--apply without --digest must refuse:\n%s", out)
	}
	v.items["aaaa0001-keeper"].Revised = v.items["aaaa0001-keeper"].Revised.Add(time.Hour) // edited elsewhere
	out, err := runCurate(t, v, curatePlan, "--apply", "--digest", d)
	if err == nil || v.writes != 0 {
		t.Fatalf("a digest from before the edit must refuse:\n%s", out)
	}
	assertNoValue(t, out)
}

// AC2 via the command: a registry-declared item and a passkey item block, and
// the error path prints no value either.
func TestCurateBlocksRegistryAndPasskeyItems(t *testing.T) {
	v := newCurateVault()
	out, err := runCurate(t, v, "folder\tdddd0004\tDev\tr\ndelete\tcccc0003\t-\tr\n")
	if err == nil || !strings.Contains(out, "reconcile owns it") || !strings.Contains(out, "passkey") {
		t.Fatalf("want both rows blocked:\n%s", out)
	}
	assertNoValue(t, out)
}

// AC6: a write that loses a passkey anywhere fails the apply, loudly.
func TestCurateApplyFailsOnAPasskeyCountChange(t *testing.T) {
	v := newCurateVault()
	plan := "reprompt\taaaa0001\t-\tr\n"
	out, _ := runCurate(t, v, plan)
	d := digestRE.FindStringSubmatch(out)[1]
	v.losePasskey = true
	out, err := runCurate(t, v, plan, "--apply", "--digest", d)
	if err == nil || !strings.Contains(out, "PASSKEY COUNT CHANGED: 3 before, 2 after") {
		t.Fatalf("a lost passkey must fail the apply:\n%s", out)
	}
}
