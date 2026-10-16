package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mergeStep rewrites the entry's source to body and deploys it with records.
func mergeStep(t *testing.T, c Config, root, home, records, body string) Outcome {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(c.Src)), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := DeployWith(c, root, home, noResolve, nil, false, records)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// #2570: a key or element the source stops declaring is removed from the
// destination, when the destination still holds what the repo wrote. What the
// repo never managed is left alone.
func TestDeploy_MergeRemovesWhatTheSourceStoppedManaging(t *testing.T) {
	root := repoWithCopilotSettings(t, `{}`)
	home, records := t.TempDir(), t.TempDir()
	c := copilotSettingsConfig()
	dst := writeDst(t, home, `{"user": true, "dirs": ["mine"]}`)

	mergeStep(t, c, root, home, records, `{"a": 1, "t": {"keep": 1, "gone": 2}, "dirs": ["x", "y"], "whole": {"n": 1}}`)
	out := mergeStep(t, c, root, home, records, `{"a": 1, "t": {"keep": 1}, "dirs": ["x"]}`)
	if !out.Changed {
		t.Error("removing a retired key is a change")
	}
	got := readObject(t, dst)
	if _, ok := got["t"].(map[string]any)["gone"]; ok {
		t.Errorf("t.gone survived its retirement: %v", got)
	}
	if _, ok := got["whole"]; ok {
		t.Errorf("a retired table, emptied, must go with its keys: %v", got)
	}
	if dirs := got["dirs"].([]any); len(dirs) != 2 || dirs[0] != "mine" || dirs[1] != "x" {
		t.Errorf("dirs = %v, want the user's element and the declared one", dirs)
	}
	if got["user"] != true || got["a"] != float64(1) {
		t.Errorf("an unmanaged or still-declared key was lost: %v", got)
	}
	if again := mergeStep(t, c, root, home, records, `{"a": 1, "t": {"keep": 1}, "dirs": ["x"]}`); again.Changed {
		t.Error("a second run after the removal must be in sync")
	}
}

// A retired value the user has since changed is theirs now: removal needs the
// destination to still hold what the repo recorded. A retired table that holds
// a key of the user's keeps that key.
func TestDeploy_MergeKeepsARetiredValueTheUserChanged(t *testing.T) {
	root := repoWithCopilotSettings(t, `{}`)
	home, records := t.TempDir(), t.TempDir()
	c := copilotSettingsConfig()
	dst := writeDst(t, home, `{}`)

	mergeStep(t, c, root, home, records, `{"k": 1, "t": {"x": 1}, "u": {"p": 1, "q": 2}, "a": 1}`)
	if err := os.WriteFile(dst, []byte(`{"k": 5, "t": {"x": 1, "y": 2}, "u": {"p": 1}, "a": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mergeStep(t, c, root, home, records, `{"a": 1}`)
	got := readObject(t, dst)
	if got["k"] != float64(5) {
		t.Errorf("k = %v, want the user's 5 kept", got["k"])
	}
	if tbl, _ := got["t"].(map[string]any); tbl == nil || tbl["y"] != float64(2) || tbl["x"] != nil {
		t.Errorf("t = %v, want x removed and the user's y kept", got["t"])
	}
	if _, ok := got["u"]; ok {
		t.Errorf("u = %v: the user deleted one of its keys, the rest were the repo's, so the emptied table goes", got["u"])
	}
}

// A destination deployed before records existed has no record: the first run
// removes nothing and writes the baseline the next one compares with.
func TestDeploy_MergeWithoutARecordRemovesNothing(t *testing.T) {
	root := repoWithCopilotSettings(t, `{"a": 1}`)
	home, records := t.TempDir(), t.TempDir()
	c := copilotSettingsConfig()
	dst := writeDst(t, home, `{"old": 1, "a": 1, "b": 1}`)

	if _, err := DeployWith(c, root, home, noResolve, nil, true, records); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(records, c.Name+".json")); err == nil {
		t.Error("an in-sync dry run wrote a record")
	}

	mergeStep(t, c, root, home, records, `{"a": 1}`)
	if readObject(t, dst)["old"] != float64(1) {
		t.Error("with no record, a key the source does not name must stay")
	}
	if _, err := os.Stat(filepath.Join(records, c.Name+".json")); err != nil {
		t.Errorf("an in-sync run must still write the baseline: %v", err)
	}

	// A record for another destination is not this destination's history.
	other := c
	other.Dst = "{HOME}/.copilot/other.json"
	mergeStep(t, other, root, home, records, `{"a": 1, "b": 1}`)
	mergeStep(t, c, root, home, records, `{"a": 1}`)
	if got := readObject(t, dst); got["old"] != float64(1) || got["b"] != float64(1) {
		t.Errorf("a record written for another destination must remove nothing here: %v", got)
	}
}

// A record that exists and does not parse must not read as "no record" (which
// removes nothing, so would be safe) or as an empty one: it is an error that
// names the file and the way out.
func TestDeploy_MergeRefusesACorruptRecord(t *testing.T) {
	root := repoWithCopilotSettings(t, `{"a": 1}`)
	home, records := t.TempDir(), t.TempDir()
	c := copilotSettingsConfig()
	writeDst(t, home, `{}`)
	path := filepath.Join(records, c.Name+".json")
	if err := os.WriteFile(path, []byte(`{"version": 1, "dst": `), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := DeployWith(c, root, home, noResolve, nil, false, records)
	if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "delete it") {
		t.Errorf("want an error naming %s and the recovery, got %v", path, err)
	}
}

// A dry run reports the removal and writes neither the destination nor the
// record; doctor's plan reads the same record, so it calls the retired key
// drift instead of in sync.
func TestPlanConfig_ReportsARetiredKeyAndADryRunRecordsNothing(t *testing.T) {
	root := repoWithCopilotSettings(t, `{}`)
	home, records := t.TempDir(), t.TempDir()
	c := copilotSettingsConfig()
	dst := writeDst(t, home, `{}`)
	mergeStep(t, c, root, home, records, `{"a": 1, "b": 1}`)
	before, err := os.ReadFile(filepath.Join(records, c.Name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(c.Src)), []byte(`{"a": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := PlanConfigWith(c, root, home, noResolve, records)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Changed {
		t.Error("a plan must report a retired key the destination still holds")
	}
	if p0, _ := PlanConfig(c, root, home, noResolve); p0.Changed {
		t.Error("without records the plan sees no history, as before")
	}
	out, err := DeployWith(c, root, home, noResolve, nil, true, records)
	if err != nil || !out.Changed {
		t.Errorf("dry run: changed=%v err=%v, want a reported change", out.Changed, err)
	}
	if readObject(t, dst)["b"] != float64(1) {
		t.Error("a dry run removed the key")
	}
	if after, _ := os.ReadFile(filepath.Join(records, c.Name+".json")); string(after) != string(before) {
		t.Error("a dry run rewrote the record")
	}
}

// With listKeys a retired binding is found by its identity, and removed only
// while it is still the table the repo wrote; a binding of the user's, or one
// the user edited, stays.
func TestDeploy_MergeRemovesARetiredListElementByIdentity(t *testing.T) {
	root := repoWithHerdrConfig(t, ``)
	home, records := t.TempDir(), t.TempDir()
	c := herdrByKey()
	dst := writeHerdrDst(t, home, "[[keys.command]]\nkey = \"mine\"\ncommand = \"top\"\n")

	mergeStep(t, c, root, home, records, `[[keys.command]]
key = "a"
command = "lazygit"

[[keys.command]]
key = "b"
command = "htop"

[[keys.command]]
key = "c"
command = "btop"
port = 9007199254740993
`)
	raw, _ := os.ReadFile(dst)
	edited := strings.Replace(string(raw), "'btop'", "'btop --tree'", 1)
	if edited == string(raw) {
		t.Fatalf("fixture: could not edit binding c in\n%s", raw)
	}
	if err := os.WriteFile(dst, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	mergeStep(t, c, root, home, records, "[[keys.command]]\nkey = \"a\"\ncommand = \"lazygit\"\n")
	byKey, _ := commandsByKey(t, dst)
	if _, ok := byKey["b"]; ok {
		t.Errorf("binding b survived its retirement: %v", byKey)
	}
	if byKey["c"]["command"] != "btop --tree" {
		t.Errorf("binding c = %v, want the user's edit kept", byKey["c"])
	}
	if byKey["mine"] == nil || byKey["a"] == nil {
		t.Errorf("the user's binding or the declared one was lost: %v", byKey)
	}

	// An edited binding is still declared by its key: replaced in its slot,
	// not retired and appended again.
	mergeStep(t, c, root, home, records, "[[keys.command]]\nkey = \"a\"\ncommand = \"lazygit --debug\"\n")
	byKey, order := commandsByKey(t, dst)
	if byKey["a"]["command"] != "lazygit --debug" || strings.Join(order, ",") != "mine,a,c" {
		t.Errorf("order = %v, a = %v; want the edited binding in its slot", order, byKey["a"])
	}
}

// A TOML value is compared as TOML: an integer above 2^53 recorded by the
// repo is still "what the repo wrote" when the destination holds it.
func TestDeploy_MergeRemovesARetiredTOMLValueExactly(t *testing.T) {
	root := repoWithHerdrConfig(t, ``)
	home, records := t.TempDir(), t.TempDir()
	c := herdrConfig()
	dst := writeHerdrDst(t, home, "")
	mergeStep(t, c, root, home, records, "big = 9007199254740993\nkeep = 1\n")
	mergeStep(t, c, root, home, records, "keep = 1\n")
	if _, ok := readTOML(t, dst)["big"]; ok {
		t.Error("a retired int64 was not recognised as the recorded value")
	}
}
