package deploy

import (
	"strings"
	"testing"
)

// herdrByKey is the herdr entry with its bindings identified by `key`, the
// declaration #2455 adds once a release carries this reader.
func herdrByKey() Config {
	c := herdrConfig()
	c.ListKeys = map[string]string{"keys.command": "key"}
	return c
}

// commandsByKey reads keys.command from dst as key -> the table's fields.
func commandsByKey(t *testing.T, dst string) (map[string]map[string]any, []string) {
	t.Helper()
	keys, _ := readTOML(t, dst)["keys"].(map[string]any)
	cmds, _ := keys["command"].([]any)
	byKey := map[string]map[string]any{}
	var order []string
	for _, c := range cmds {
		table, ok := c.(map[string]any)
		if !ok {
			t.Fatalf("keys.command element is %T, want a table", c)
		}
		k, _ := table["key"].(string)
		if _, dup := byKey[k]; dup {
			t.Errorf("key %q is bound twice", k)
		}
		byKey[k] = table
		order = append(order, k)
	}
	return byKey, order
}

const lazygitSource = `[[keys.command]]
key = "prefix+d"
type = "popup"
command = "lazygit"
width = "80%"
`

// #2455: editing one field of a managed binding replaced nothing under the
// union; the old table stayed beside the new one, two bindings on one key.
func TestDeploy_MergeReplacesAListElementByIdentity(t *testing.T) {
	root := repoWithHerdrConfig(t, lazygitSource)
	home := t.TempDir()
	dst := writeHerdrDst(t, home, `[[keys.command]]
key = "prefix+alt+t"
type = "pane"
command = "htop"

[[keys.command]]
key = "prefix+d"
type = "popup"
command = "lazygit"
width = "90%"
`)
	out, err := Deploy(herdrByKey(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Changed {
		t.Error("a changed field of a managed element is a change")
	}
	byKey, order := commandsByKey(t, dst)
	if got := byKey["prefix+d"]["width"]; got != "80%" {
		t.Errorf("prefix+d width = %v, want the source's 80%%", got)
	}
	if strings.Join(order, ",") != "prefix+alt+t,prefix+d" {
		t.Errorf("order = %v, want the user's binding kept in place and the managed one in its slot", order)
	}
	again, err := Deploy(herdrByKey(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed {
		t.Error("a second merge must be in sync")
	}
}

// The damage the union already left on a machine: two tables under the managed
// key. The merge keeps one, the source's, and says it changed something. The
// first table already equals the source, so the change is the dropped one.
func TestDeploy_MergeCollapsesDuplicatesOfAManagedIdentity(t *testing.T) {
	root := repoWithHerdrConfig(t, lazygitSource)
	home := t.TempDir()
	dst := writeHerdrDst(t, home, lazygitSource+`
[[keys.command]]
key = "prefix+d"
type = "popup"
command = "lazygit"
width = "90%"
`)
	out, err := Deploy(herdrByKey(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	byKey, order := commandsByKey(t, dst)
	if len(order) != 1 || byKey["prefix+d"]["width"] != "80%" {
		t.Errorf("keys.command = %v, want one prefix+d with width 80%%", order)
	}
	if !out.Changed {
		t.Error("dropping a duplicate is a change")
	}
}

// A managed element the destination does not hold yet is appended.
func TestDeploy_MergeAppendsANewIdentity(t *testing.T) {
	root := repoWithHerdrConfig(t, lazygitSource)
	home := t.TempDir()
	dst := writeHerdrDst(t, home, `[[keys.command]]
key = "prefix+alt+t"
type = "pane"
command = "htop"
`)
	if _, err := Deploy(herdrByKey(), root, home, noResolve, nil, false); err != nil {
		t.Fatal(err)
	}
	if _, order := commandsByKey(t, dst); strings.Join(order, ",") != "prefix+alt+t,prefix+d" {
		t.Errorf("order = %v, want the user's binding then the repo's", order)
	}
}

// A managed element without the identity field cannot own a slot; falling back
// to the union would bring #2455 back silently, so it is an error naming where.
func TestDeploy_MergeRefusesAManagedElementWithoutItsIdentity(t *testing.T) {
	for name, src := range map[string]string{
		"missing key": "[[keys.command]]\ntype = \"popup\"\ncommand = \"lazygit\"\n",
		"not a table": "[keys]\ncommand = [\"lazygit\"]\n",
		"shared key":  lazygitSource + "\n" + lazygitSource,
	} {
		t.Run(name, func(t *testing.T) {
			root := repoWithHerdrConfig(t, src)
			_, err := Deploy(herdrByKey(), root, t.TempDir(), noResolve, nil, false)
			if err == nil || !strings.Contains(err.Error(), "keys.command") {
				t.Fatalf("got %v, want an error naming keys.command", err)
			}
		})
	}
}

func TestParseManifest_ListKeys(t *testing.T) {
	const entry = `{"name":"herdr","src":"ai/herdr/config.toml","dst":"{HOME}/c.toml","strategy":"merge","listKeys":`
	for name, tc := range map[string]struct {
		doc, want string
	}{
		"accepted at version 5":   {`{"version":5,"configs":[` + entry + `{"keys.command":"key"}}]}`, ""},
		"refused at version 4":    {`{"version":4,"configs":[` + entry + `{"keys.command":"key"}}]}`, "listKeys needs manifest version 5"},
		"refused without merge":   {`{"version":5,"configs":[{"name":"x","src":"a.toml","dst":"b.toml","listKeys":{"k":"id"}}]}`, "listKeys needs strategy merge"},
		"refused with empty path": {`{"version":5,"configs":[` + entry + `{"":"key"}}]}`, "empty"},
		"refused with empty key":  {`{"version":5,"configs":[` + entry + `{"keys.command":""}}]}`, "empty"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tc.doc))
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("want accepted, got %v", err)
			case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
				t.Fatalf("want an error containing %q, got %v", tc.want, err)
			}
		})
	}
}
