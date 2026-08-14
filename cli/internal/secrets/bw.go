package secrets

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrBWItemNotFound marks a `bw get item` failure where Bitwarden reports the item
// does not exist (its "Not found." message), as distinct from a locked or
// unauthenticated vault (a different message, surfaced as-is). `dotf secrets set`
// keys its create-absent path on this sentinel: ONLY a genuinely-missing item may
// trigger a create, so a locked vault can never be mistaken for absent and spawn a
// duplicate item. The discrimination is by CLI message (fragile by nature); `bw serve`
// would expose a proper status code behind the same seam later.
var ErrBWItemNotFound = errors.New("bw item not found")

// ErrBWFieldNotFound marks an item that was fetched fine but does not carry the
// requested field. For `dotf secrets set` this is a writable case (append the field),
// distinct from a missing item (create) or a locked vault (fail) — the three branches
// the write path must keep apart.
var ErrBWFieldNotFound = errors.New("bw field not found on item")

// isNotFound reports whether a bw CLI error message indicates a missing item (rather
// than a locked/unauthenticated vault, which yields a different message).
func isNotFound(msg string) bool {
	return strings.Contains(strings.ToLower(msg), "not found")
}

// BWReader fetches a single field value from a Bitwarden item. The seam that keeps
// the bw backend unit-testable with no Bitwarden vault and no unlock — tests inject
// a map-backed fake; production is BWGet (a `bw get` shell-out).
type BWReader interface {
	Field(item, field string) (string, error)
}

// BWGet is the production BWReader: it shells out to the Bitwarden CLI to read a
// field from an item. It is the bw analog of AgeDecrypt (age --decrypt) — thin I/O,
// verified by a live smoke with the operator's unlocked session, not in CI.
//
// One `bw get item <item>` call returns the item JSON; fieldFromItem then picks the
// field from the typed login (password/username), the note (notes), or a custom
// field by name. `--nointeraction` guarantees it never blocks on a prompt: a locked
// vault or missing BW_SESSION fails fast with the CLI's stderr surfaced. The item is
// keyed by its unique name or id (the Bitwarden folder is not part of the lookup).
//
// bw serve — a local REST daemon holding the unlocked vault, faster for batch reads
// (one unlock, many millisecond GETs) — is a drop-in replacement behind this same
// BWReader seam, deferred as a perf upgrade (ADR-028). It must stay localhost-only:
// the serve API is unauthenticated by design and exposes the whole unlocked vault.
type BWGet struct {
	Bin string // bw binary name/path; "" → "bw" (a field for tests/overrides)
}

// Field reads field from the Bitwarden item via one `bw get item` shell-out.
func (g BWGet) Field(item, field string) (string, error) {
	bin := g.Bin
	if bin == "" {
		bin = "bw"
	}
	cmd := exec.Command(bin, "get", "item", item, "--nointeraction") //nolint:gosec // item is operator-controlled registry data
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if isNotFound(msg) {
			return "", fmt.Errorf("%w: bw get item %q: %s", ErrBWItemNotFound, item, msg)
		}
		return "", fmt.Errorf("bw get item %q: %s", item, msg)
	}
	return fieldFromItem(stdout.Bytes(), field)
}

// bwItem is the subset of `bw get item` JSON that BWGet reads.
type bwItem struct {
	Login *struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"login"`
	Notes  string `json:"notes"`
	Fields []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"fields"`
}

// fieldFromItem extracts field from a `bw get item` JSON payload. The typed login
// fields and the note are first-class names; any other name is matched against the
// item's custom fields[] — an unknown name is an error (catches a registry typo).
// Requesting a login field on an item with no login block (e.g. field: password on
// a secure note) is an error too, not a silent empty value (#612 A1). An empty but
// present value is left to EnvFor's empty-value guard.
func fieldFromItem(data []byte, field string) (string, error) {
	var it bwItem
	if err := json.Unmarshal(data, &it); err != nil {
		return "", fmt.Errorf("parse bw item JSON: %w", err)
	}
	switch field {
	case "password", "username":
		if it.Login == nil {
			return "", fmt.Errorf("%w: field %q requested but item has no login block", ErrBWFieldNotFound, field)
		}
		if field == "password" {
			return it.Login.Password, nil
		}
		return it.Login.Username, nil
	case "notes":
		return it.Notes, nil
	}
	for _, f := range it.Fields {
		if f.Name == field {
			return f.Value, nil
		}
	}
	return "", fmt.Errorf("%w: field %q", ErrBWFieldNotFound, field)
}

// BWWriter writes a field value into a Bitwarden item. The write analog of BWReader:
// the seam that keeps the write path unit-testable with no Bitwarden vault — tests
// inject a fake; production is BWPut (a `bw edit` shell-out).
type BWWriter interface {
	SetField(item, field, value string) error
}

// setItemField returns itemJSON with field set to value, preserving EVERY other key
// of the item (read-modify-write) so a sibling field — or the item's id/type/name —
// is never clobbered. It works on a generic map (not the narrow bwItem struct) so
// unknown keys survive. password/username set the typed login (created if absent);
// notes sets the note; any other name updates the matching custom field or appends a
// new hidden one (type 1).
func setItemField(itemJSON []byte, field, value string) ([]byte, error) {
	var m map[string]any
	if err := json.Unmarshal(itemJSON, &m); err != nil {
		return nil, fmt.Errorf("parse bw item JSON: %w", err)
	}
	switch field {
	case "password", "username":
		login, _ := m["login"].(map[string]any)
		if login == nil {
			login = map[string]any{}
			m["login"] = login
		}
		login[field] = value
	case "notes":
		m["notes"] = value
	default:
		setCustomField(m, field, value)
	}
	return json.Marshal(m)
}

// setCustomField updates the custom field named field in m's fields[], or appends a
// new hidden field (type 1) when none matches — never touching the other fields.
func setCustomField(m map[string]any, field, value string) {
	fields, _ := m["fields"].([]any)
	for _, f := range fields {
		if fm, ok := f.(map[string]any); ok {
			if name, _ := fm["name"].(string); name == field {
				fm["value"] = value
				return
			}
		}
	}
	m["fields"] = append(fields, map[string]any{"name": field, "value": value, "type": 1})
}

// itemID extracts the Bitwarden item id from its JSON (the handle `bw edit item`
// needs). An item with no id is a malformed payload.
func itemID(itemJSON []byte) (string, error) {
	var m struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(itemJSON, &m); err != nil {
		return "", fmt.Errorf("parse bw item JSON: %w", err)
	}
	if m.ID == "" {
		return "", fmt.Errorf("bw item has no id")
	}
	return m.ID, nil
}

// BWPut is the production BWWriter: it updates an existing item via read-modify-write
// (`bw get item` → setItemField → base64 → `bw edit item <id>`). Thin I/O verified by
// a live smoke with the operator's unlocked session, not in CI (the analog of BWGet).
// The new value reaches `bw` through stdin (base64), never a temp file. Creating an
// absent item is deferred to `dotf secrets set` — `SetField` errors clearly when the
// item is missing rather than risk spawning a duplicate on a locked/transient failure.
type BWPut struct {
	Bin string // bw binary name/path; "" → "bw"
}

// SetField sets field on item to value, preserving the item's other fields.
func (p BWPut) SetField(item, field, value string) error {
	cur, err := p.run(nil, "get", "item", item)
	if err != nil {
		if isNotFound(err.Error()) {
			return fmt.Errorf("%w: %q (use `dotf secrets set` to create it)", ErrBWItemNotFound, item)
		}
		return fmt.Errorf("bw get item %q (it must already exist to edit): %w", item, err)
	}
	id, err := itemID(cur)
	if err != nil {
		return err
	}
	updated, err := setItemField(cur, field, value)
	if err != nil {
		return err
	}
	enc := base64.StdEncoding.EncodeToString(updated)
	if _, err := p.run(strings.NewReader(enc), "edit", "item", id); err != nil {
		return fmt.Errorf("bw edit item %q: %w", item, err)
	}
	return nil
}

// BWCreator creates a brand-new Bitwarden item. It is split from BWWriter because
// creating is the privileged, duplicate-risky path `dotf secrets set` gates behind an
// explicit confirm / --yes; an update-only caller (e.g. migrate's re-verify) needs only
// BWWriter and must not be able to create by accident.
type BWCreator interface {
	// CreateItem creates item carrying field=value. folderID is an already-resolved
	// Bitwarden folder id (via BWFolderResolver) — "" places the item unfoldered,
	// today's default for any secret with no bw.folder declared (OPS-028).
	CreateItem(item, field, value, folderID string) error
}

// BWFolderResolver resolves a Bitwarden folder NAME (registry bw.folder, e.g.
// "Dotfiles/apps") to its id, creating the folder if the vault doesn't have it yet —
// OPS-028, closing the gap where ADR-028 ratified a folder taxonomy the write path
// couldn't place anything into. Split from BWCreator (its own interface, its own test
// double) because folder resolution and item creation are independently testable:
// CreateItem's JSON-body test never needs a fake folder list, and ResolveFolder's
// name→id test never needs a fake item store.
type BWFolderResolver interface {
	// ResolveFolder returns "" for an empty name (no folder declared — a no-op, not
	// an error) so callers never special-case the common unfoldered secret.
	ResolveFolder(name string) (string, error)
}

// bwFolder is the subset of `bw list folders` / `bw create folder` JSON this package
// reads.
type bwFolder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ResolveFolder lists the vault's folders, matches by exact name, and creates the
// folder when absent — idempotent (a second call finds the just-created folder and
// never creates a duplicate). Live-verified against an unlocked vault, not in CI, like
// the rest of the write seam.
func (p BWPut) ResolveFolder(name string) (string, error) {
	if name == "" {
		return "", nil
	}
	out, err := p.run(nil, "list", "folders")
	if err != nil {
		return "", fmt.Errorf("bw list folders: %w", err)
	}
	var folders []bwFolder
	if err := json.Unmarshal(out, &folders); err != nil {
		return "", fmt.Errorf("parse bw list folders JSON: %w", err)
	}
	for _, f := range folders {
		if f.Name == name {
			return f.ID, nil
		}
	}
	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return "", err
	}
	enc := base64.StdEncoding.EncodeToString(body)
	created, err := p.run(strings.NewReader(enc), "create", "folder")
	if err != nil {
		return "", fmt.Errorf("bw create folder %q: %w", name, err)
	}
	var f bwFolder
	if err := json.Unmarshal(created, &f); err != nil {
		return "", fmt.Errorf("parse bw create folder JSON: %w", err)
	}
	if f.ID == "" {
		return "", fmt.Errorf("bw create folder %q: response carried no id", name)
	}
	return f.ID, nil
}

// CreateItem creates a new login item named item carrying field=value, filed under
// folderID when non-empty. It applies the same read-modify-write (setItemField) to a
// minimal login template, base64-encodes it, and pipes it to `bw create item` — so a
// created item and an edited item place a field identically. Live-verified (canary,
// #612 C8), not in CI, like BWPut.SetField.
func (p BWPut) CreateItem(item, field, value, folderID string) error {
	body, err := newItemBody(item, field, value, folderID)
	if err != nil {
		return err
	}
	enc := base64.StdEncoding.EncodeToString(body)
	if _, err := p.run(strings.NewReader(enc), "create", "item"); err != nil {
		return fmt.Errorf("bw create item %q: %w", item, err)
	}
	return nil
}

// newItemBody builds the JSON for a new login item named item carrying field=value, by
// applying setItemField to a minimal login template. folderID, when non-empty, is
// attached as-is — this function trusts an already-resolved id, never a name (that
// resolution is BWFolderResolver's job). The pure, unit-tested core of CreateItem
// (CreateItem itself is the live-tested shell-out).
func newItemBody(item, field, value, folderID string) ([]byte, error) {
	m := map[string]any{
		"type":  1, // login
		"name":  item,
		"login": map[string]any{},
	}
	if folderID != "" {
		m["folderId"] = folderID
	}
	base, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return setItemField(base, field, value)
}

// run executes `bw <args...> --nointeraction` with optional stdin, returning stdout
// or an error carrying bw's stderr.
func (p BWPut) run(stdin *strings.Reader, args ...string) ([]byte, error) {
	bin := p.Bin
	if bin == "" {
		bin = "bw"
	}
	cmd := exec.Command(bin, append(args, "--nointeraction")...) //nolint:gosec // args are operator-controlled registry data
	if stdin != nil {
		cmd.Stdin = stdin
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return stdout.Bytes(), nil
}
