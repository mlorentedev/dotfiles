package secrets

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// Reading the vault's SHAPE without reading its contents.
//
// Every other read path in this package resolves ONE field a caller already
// named. Comparing the declared layout against the store needs the opposite: the
// whole inventory, and none of its values. The two requirements pull against each
// other — `bw serve` has no projection, so `/list/object/items` answers with every
// item's complete plaintext, values included.
//
// So the projection happens HERE, at the producer, and it is enforced by the type
// system rather than by discipline: `encoding/json` discards what the target
// struct does not name, so a password, a TOTP seed or a custom field's content is
// dropped during decoding and never exists as a Go value at all. There is no line
// to forget to redact and no filter that can be reordered past.
//
// Two members are read as content and immediately reduced to booleans, because the
// questions they answer cannot be asked otherwise: `notes` and `login.username`.
// `login.password` is NOT among them — see itemWire.
//
// That is the same rule bwserve.go's call() already states for error messages —
// "status code and byte count ONLY, never the body itself" — applied to the one
// endpoint whose body IS the vault.

// ItemSummary is one vault item's shape: what it is called, where it lives, and
// which fields it carries. Deliberately not "one item": there is no accessor for
// a value here, because this type is what crosses the seam into reporting code
// that formats for a terminal and a log.
type ItemSummary struct {
	Name string
	// Folder is the item's folder NAME, already resolved from its id, or "" when
	// the item is unfoldered. A name is what a declaration compares against and
	// what a human reads; an id is neither.
	Folder string
	// Fields are the custom field names, sorted. Names only — see the file header.
	Fields []string
	// HasNotes records whether the item's note is non-empty, WITHOUT reading it.
	// It matters because the registry uses `field: notes` for multi-line secrets,
	// so "the declared field is notes" is checkable only if this is known.
	HasNotes bool
	// HasLogin and HasUsername mirror fieldFromItem's other special cases: it
	// answers `username` and `password` from the typed login block, not from the
	// custom-field list, so a checker that only knew about custom fields reports
	// every login-sourced declaration as missing. Measured: DOCKERHUB_USERNAME
	// resolves fine and was reported absent.
	//
	// There is no HasPassword, deliberately. itemWire below never names
	// `password`, so it is discarded during decoding and cannot be read here at
	// all; a declared `field: password` is answered from HasLogin instead. That
	// is a weaker answer — an item with a login block but an empty password reads
	// as present — and it is the right trade: this type's whole purpose is that no
	// secret value ever crosses it, and a password is the one field where
	// "non-empty" cannot be computed without decoding the secret itself.
	HasLogin    bool
	HasUsername bool
	// Revised is the item's last-modified time. Bitwarden bumps it on ANY edit —
	// a rename, a folder move, a note tweak — so it is an UPPER BOUND on the age
	// of the credential, never the rotation date. Callers that report rotation
	// age must say which one they mean.
	Revised time.Time
}

// BWLister reads the vault's inventory as shapes. A seam, so a caller can be
// tested against a fixture with no daemon, no vault and no unlock — and so the
// projection above has exactly one implementation to audit.
type BWLister interface {
	ListItems() ([]ItemSummary, error)
	ListFolders() ([]string, error)
}

// itemWire is the decode target, and its FIELD SET is the security boundary.
//
// It names `id`, `name`, `folderId`, `revisionDate`, `notes`, the field objects'
// `name`, and `login.username`. It does not name `login.password`, `card`,
// `identity`, `sshKey`, `passwordHistory`, or a field's `value`. Anything absent
// here is discarded by encoding/json before this function can mishandle it —
// most importantly the password, which therefore cannot be read even by mistake.
//
// `notes` and `login.username` are the uncomfortable members: they are content,
// not shape. Each is read because the corresponding boolean cannot be computed
// without it, and each is consumed into a bool a few lines later — never stored,
// never returned, never logged. If that ever stops being true, the type has
// stopped being a boundary.
type itemWire struct {
	Name         string `json:"name"`
	FolderID     string `json:"folderId"`
	RevisionDate string `json:"revisionDate"`
	Notes        string `json:"notes"`
	Fields       []struct {
		Name string `json:"name"`
	} `json:"fields"`
	Login *struct {
		Username string `json:"username"`
	} `json:"login"`
}

// folderWire is the same idea for folders, which carry no secret material at all
// (bwbackend.go already records `/list/object/folders` as clean on that basis).
type folderWire struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// listEnvelope is the inner wrapper bw serve puts around a list: the outer
// envelope's `data` is itself `{"object":"list","data":[...]}`.
type listEnvelope struct {
	Data json.RawMessage `json:"data"`
}

// ListFolders returns every folder name, sorted. Bitwarden has no real hierarchy:
// a nested folder is one whose NAME contains "/", so "Dotfiles/apps" is a single
// folder whose name happens to read like a path. Returned verbatim for that
// reason — normalising it here would hide exactly the mismatch a layout check
// exists to report.
func (c BWServeClient) ListFolders() ([]string, error) {
	raw, err := c.call("GET", "/list/object/folders", nil)
	if err != nil {
		return nil, err
	}
	folders, err := decodeFolders(raw)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(folders))
	for _, f := range folders {
		// bw reports the pseudo-folder "No Folder" with a null id. It is not a
		// folder anything can be declared into, so it is not one here either.
		if f.ID == "" {
			continue
		}
		out = append(out, f.Name)
	}
	sort.Strings(out)
	return out, nil
}

// ListItems returns every item's shape, sorted by name, with folder ids already
// resolved to names. Two round trips rather than one: the item list carries only
// folder IDs, and a report naming a UUID is not a report.
func (c BWServeClient) ListItems() ([]ItemSummary, error) {
	rawFolders, err := c.call("GET", "/list/object/folders", nil)
	if err != nil {
		return nil, err
	}
	folders, err := decodeFolders(rawFolders)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]string, len(folders))
	for _, f := range folders {
		byID[f.ID] = f.Name
	}

	rawItems, err := c.call("GET", "/list/object/items", nil)
	if err != nil {
		return nil, err
	}
	return decodeItems(rawItems, byID)
}

// decodeItems is separated from the HTTP call so the projection — the part that
// matters — is unit-testable against a recorded payload, including one carrying
// values that must not survive.
func decodeItems(raw json.RawMessage, folderByID map[string]string) ([]ItemSummary, error) {
	var inner listEnvelope
	if err := json.Unmarshal(raw, &inner); err != nil {
		// Byte count, never the body: this one carries the whole vault.
		return nil, fmt.Errorf("bw serve item list: unexpected shape (%d bytes)", len(raw))
	}
	var wire []itemWire
	if err := json.Unmarshal(inner.Data, &wire); err != nil {
		return nil, fmt.Errorf("bw serve item list: unexpected item shape (%d bytes)", len(inner.Data))
	}

	out := make([]ItemSummary, 0, len(wire))
	for _, w := range wire {
		names := make([]string, 0, len(w.Fields))
		for _, f := range w.Fields {
			names = append(names, f.Name)
		}
		sort.Strings(names)

		// Parsed leniently: an unparseable timestamp yields the zero time, which
		// a caller reports as "unknown", rather than aborting an inventory over
		// one malformed row.
		revised, _ := time.Parse(time.RFC3339, w.RevisionDate)

		out = append(out, ItemSummary{
			Name:        w.Name,
			Folder:      folderByID[w.FolderID],
			Fields:      names,
			HasNotes:    w.Notes != "",
			HasLogin:    w.Login != nil,
			HasUsername: w.Login != nil && w.Login.Username != "",
			Revised:     revised,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func decodeFolders(raw json.RawMessage) ([]folderWire, error) {
	var inner listEnvelope
	if err := json.Unmarshal(raw, &inner); err != nil {
		return nil, fmt.Errorf("bw serve folder list: unexpected shape (%d bytes)", len(raw))
	}
	var wire []folderWire
	if err := json.Unmarshal(inner.Data, &wire); err != nil {
		return nil, fmt.Errorf("bw serve folder list: unexpected folder shape (%d bytes)", len(inner.Data))
	}
	return wire, nil
}
