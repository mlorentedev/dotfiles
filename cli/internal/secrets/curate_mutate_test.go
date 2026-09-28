package secrets

import (
	"encoding/json"
	"errors"
	"testing"
)

func mustMutate(t *testing.T, item string, m Mutation) map[string]any {
	t.Helper()
	out, err := applyMutation([]byte(item), m)
	if err != nil {
		t.Fatalf("applyMutation(%v): %v", m.Kind, err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	return got
}

// Every mutation must refuse an item that carries a passkey, before it changes
// anything: whether a whole-item rewrite through bw preserves
// login.fido2Credentials has never been measured, so curate never risks one.
func TestApplyMutationRefusesAPasskeyItem(t *testing.T) {
	item := `{"id":"x","login":{"fido2Credentials":[{"credentialId":"c"}]},"fields":[{"name":"k","type":0,"value":"v"}]}`
	for _, m := range []Mutation{
		{Kind: MutMove, Value: "f"}, {Kind: MutHide, Field: "k"}, {Kind: MutReprompt},
		{Kind: MutAddURI, Value: "https://a"}, {Kind: MutSetText, Field: "n", Value: "v"},
		{Kind: MutRemoveField, Field: "k"},
	} {
		if _, err := applyMutation([]byte(item), m); !errors.Is(err, ErrPasskeyItem) {
			t.Errorf("%v on a passkey item: want ErrPasskeyItem, got %v", m.Kind, err)
		}
	}
}

func TestApplyMutationEditsOneThingAndKeepsTheRest(t *testing.T) {
	item := `{"id":"x","name":"n","unknownKey":7,"reprompt":0,"folderId":null,
	  "login":{"username":"u","uris":[{"uri":"https://a","match":null}]},
	  "fields":[{"name":"token","type":0,"value":"v"},{"name":"acct","type":0,"value":"w"}]}`

	got := mustMutate(t, item, Mutation{Kind: MutHide, Field: "token"})
	fields := got["fields"].([]any)
	if fields[0].(map[string]any)["type"].(float64) != 1 || fields[1].(map[string]any)["type"].(float64) != 0 {
		t.Errorf("hide must change only the named field: %v", fields)
	}
	if got["unknownKey"].(float64) != 7 {
		t.Error("an unknown key must survive the rewrite")
	}

	got = mustMutate(t, item, Mutation{Kind: MutReprompt})
	if got["reprompt"].(float64) != 1 {
		t.Errorf("reprompt: %v", got["reprompt"])
	}

	got = mustMutate(t, item, Mutation{Kind: MutAddURI, Value: "https://b"})
	uris := got["login"].(map[string]any)["uris"].([]any)
	if len(uris) != 2 || uris[1].(map[string]any)["uri"] != "https://b" {
		t.Errorf("add-uri must append: %v", uris)
	}

	got = mustMutate(t, item, Mutation{Kind: MutSetText, Field: "alias", Value: "a@x"})
	fields = got["fields"].([]any)
	last := fields[len(fields)-1].(map[string]any)
	if last["name"] != "alias" || last["value"] != "a@x" || last["type"].(float64) != 0 {
		t.Errorf("set-text must append a TEXT field: %v", last)
	}

	got = mustMutate(t, item, Mutation{Kind: MutRemoveField, Field: "acct"})
	if len(got["fields"].([]any)) != 1 {
		t.Errorf("remove-field: %v", got["fields"])
	}

	got = mustMutate(t, item, Mutation{Kind: MutMove, Value: "folder-1"})
	if got["folderId"] != "folder-1" {
		t.Errorf("move: %v", got["folderId"])
	}
}

// A mutation that finds nothing to act on is an error, never a silent no-op: the
// plan saw the thing, so its absence means the store changed underneath.
func TestApplyMutationRefusesAMissingTarget(t *testing.T) {
	item := `{"id":"x","login":{"uris":[]},"fields":[{"name":"flag","type":2,"value":"true"}]}`
	for _, m := range []Mutation{
		{Kind: MutHide, Field: "absent"},
		{Kind: MutHide, Field: "flag"}, // a boolean has no hidden form
		{Kind: MutRemoveField, Field: "absent"},
		{Kind: MutSetText, Field: "flag", Value: "v"}, // never overwrites
	} {
		if _, err := applyMutation([]byte(item), m); err == nil {
			t.Errorf("%v %q: want an error", m.Kind, m.Field)
		}
	}
	if _, err := applyMutation([]byte(`{"id":"x","fields":[]}`), Mutation{Kind: MutAddURI, Value: "https://a"}); err == nil {
		t.Error("add-uri on an item with no login: want an error")
	}
}
