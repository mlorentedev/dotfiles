package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrPasskeyItem is returned for any curate write to an item that carries a
// passkey. Every edit is a whole-item rewrite, and whether that round trip keeps
// login.fido2Credentials intact has never been measured here. A lost passkey
// cannot be restored from the escrow into the account it signed in to, so the
// write is refused rather than tested live (SEC-006).
var ErrPasskeyItem = errors.New("the item carries a passkey; curate never writes to one")

// MutationKind names one curate edit.
type MutationKind string

// The curate edits. Each changes one thing and preserves every other key of the
// item, known or not.
const (
	MutMove        MutationKind = "move"         // Value: folder id, "" unfiles
	MutHide        MutationKind = "hide"         // Field: a text field turns hidden
	MutReprompt    MutationKind = "reprompt"     // turn master-password reprompt on
	MutAddURI      MutationKind = "add-uri"      // Value: the URI to append
	MutSetText     MutationKind = "set-text"     // Field, Value: a NEW text field
	MutRemoveField MutationKind = "remove-field" // Field: a custom field to remove
)

// Mutation is one typed edit. Typed rather than an arbitrary func so that every
// curate write passes through applyMutation, and the passkey guard has exactly
// one place to live.
type Mutation struct {
	Kind  MutationKind
	Field string
	Value string
}

// BWCurator applies one Mutation to the item with the given id. Its own interface,
// like BWMover and BWFieldRemover: only `curate` holds one.
type BWCurator interface {
	Curate(id string, m Mutation) error
}

// Curate applies m to item through the CLI's read-modify-write.
func (p BWPut) Curate(item string, m Mutation) error {
	return p.editItem(item, func(cur []byte) ([]byte, error) { return applyMutation(cur, m) })
}

// Curate applies m to item through the daemon's read-modify-write.
func (w BWServeWriter) Curate(item string, m Mutation) error {
	return w.editItem(item, func(cur []byte) ([]byte, error) { return applyMutation(cur, m) })
}

// applyMutation returns itemJSON with m applied. It refuses a passkey item first,
// before any branch, so no mutation added later can skip the check.
func applyMutation(itemJSON []byte, m Mutation) ([]byte, error) {
	var item map[string]any
	if err := json.Unmarshal(itemJSON, &item); err != nil {
		return nil, fmt.Errorf("parse bw item JSON: %w", err)
	}
	login, _ := item["login"].(map[string]any)
	if pk, _ := login["fido2Credentials"].([]any); len(pk) > 0 {
		return nil, ErrPasskeyItem
	}

	switch m.Kind {
	case MutMove:
		return setItemFolder(itemJSON, m.Value)
	case MutRemoveField:
		if m.Field == "password" || m.Field == "username" || m.Field == "notes" {
			return nil, fmt.Errorf("curate removes custom fields only, not %q", m.Field)
		}
		return removeItemField(itemJSON, m.Field)
	case MutReprompt:
		item["reprompt"] = 1
	case MutHide:
		f, err := customField(item, m.Field)
		if err != nil {
			return nil, err
		}
		if t, _ := f["type"].(float64); t != 0 && t != 1 {
			return nil, fmt.Errorf("field %q is not a text field (type %v)", m.Field, f["type"])
		}
		f["type"] = 1
	case MutAddURI:
		if login == nil {
			return nil, fmt.Errorf("the item has no login to carry a URI")
		}
		uris, _ := login["uris"].([]any)
		login["uris"] = append(uris, map[string]any{"uri": m.Value, "match": nil})
	case MutSetText:
		if _, err := customField(item, m.Field); err == nil {
			return nil, fmt.Errorf("field %q already exists; curate never overwrites one", m.Field)
		}
		fields, _ := item["fields"].([]any)
		item["fields"] = append(fields, map[string]any{"name": m.Field, "value": m.Value, "type": 0})
	default:
		return nil, fmt.Errorf("unknown mutation %q", m.Kind)
	}
	return json.Marshal(item)
}

// customField returns the custom field named name, as a live map into item.
func customField(item map[string]any, name string) (map[string]any, error) {
	fields, _ := item["fields"].([]any)
	for _, f := range fields {
		if fm, ok := f.(map[string]any); ok {
			if n, _ := fm["name"].(string); n == name {
				return fm, nil
			}
		}
	}
	return nil, fmt.Errorf("item carries no field %q", name)
}
