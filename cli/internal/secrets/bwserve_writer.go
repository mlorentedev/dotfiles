package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// BWWriteClient is the whole write surface a Bitwarden-backed mutation needs: update
// an existing field (BWWriter), create an absent item (BWCreator), and resolve — or
// create — the folder the registry's taxonomy declares (BWFolderResolver). The three
// stay separate interfaces so an update-only caller cannot create by accident; this
// alias exists because `set`/`rotate`/`migrate` need all three at once and threading
// three parameters through them buys nothing.
//
// Two implementations: BWPut (the `bw` CLI shellout) and BWServeWriter (the daemon).
type BWWriteClient interface {
	BWWriter
	BWCreator
	BWFolderResolver
}

// BWServeWriter is the serve-backed BWWriteClient: the write analog of BWServeReader,
// and the fix for BUG-084 (#993). CLI-024-secrets-bw-serve moved the read path to the
// daemon and deferred the write path as "a natural fast-follow"; nothing enforced that,
// so `dotf secrets set` kept shelling out to a `bw` binary that authenticates from
// BW_SESSION — the ambient variable ADR-028 exists to abolish. The result was a write
// path that could not write on a correctly-unlocked machine.
//
// Every method reuses the same pure core as BWPut (setItemField, newItemBody, itemID),
// so an item created or edited through the daemon is byte-identical to one created or
// edited through the CLI. Only the transport differs — which is the entire point of
// the seam, and what makes the two backends safe to choose between at runtime.
//
// Not available here: export. `bw serve` exposes no export route (verified against the
// @bitwarden/cli 2026.5.0 bundle — the sole "/export" string in it is an *organization*
// export against the cloud API), so BWExport stays a CLI shellout permanently. See
// Backup's error path and BUG-084's proposal Out of scope.
type BWServeWriter struct {
	Client BWServeClient
}

// SetField sets field on item to value, preserving every other key of the item.
//
// Read-modify-write, exactly like BWPut.SetField: resolve the item, mutate one field,
// PUT the whole thing back. It reuses BWServeReader.getItemJSON for the resolution so
// the daemon's name/id disambiguation cannot drift from the read path's, and
// setItemField for the mutation so field placement cannot drift from the CLI's.
//
// The item JSON goes back as a raw JSON body (json.RawMessage), NOT base64: the
// daemon's REST API takes JSON directly, where the `bw` CLI needs `bw encode`d stdin.
// That asymmetry is the only real difference between the two implementations.
//
// It does NOT sync first — deliberately, for parity with BWPut, so the two backends
// behave identically under the same vault state (AC5). The daemon answers reads from
// its own cache, so a sibling field edited in the web vault since the last sync can be
// clobbered by this read-modify-write. That risk is inherited, not introduced (the CLI
// has the same cache), but an unlocked daemon is easier to leave running for days,
// which widens the window. Callers that care sync explicitly first — see
// BWServeClient.Sync, and CLI-037's rotate.
func (w BWServeWriter) SetField(item, field, value string) error {
	// Same shape, same client: the read half of this read-modify-write is literally the
	// read backend, so a conversion keeps them provably identical (and stops compiling
	// if the two ever diverge) rather than re-deriving one from the other.
	cur, err := BWServeReader(w).getItemJSON(item)
	if err != nil {
		// getItemJSON already classifies absence as ErrBWItemNotFound; re-wrap it
		// with the same remediation BWPut.SetField gives, so `set`'s create-absent
		// gate keys off one sentinel regardless of which backend answered.
		if errors.Is(err, ErrBWItemNotFound) {
			return fmt.Errorf("%w: %q (use `dotf secrets set` to create it)", ErrBWItemNotFound, item)
		}
		return fmt.Errorf("bw serve get item %q (it must already exist to edit): %w", item, err)
	}
	id, err := itemID(cur)
	if err != nil {
		return err
	}
	updated, err := setItemField(cur, field, value)
	if err != nil {
		return err
	}
	if _, err := w.Client.call(http.MethodPut, "/object/item/"+url.PathEscape(id), json.RawMessage(updated)); err != nil {
		return fmt.Errorf("bw serve edit item %q: %w", item, err)
	}
	return nil
}

// CreateItem creates a new login item named item carrying field=value, filed under
// folderID when non-empty. Same minimal-login-template core as BWPut.CreateItem
// (newItemBody), so a created item is identical whichever backend created it.
func (w BWServeWriter) CreateItem(item, field, value, folderID string) error {
	body, err := newItemBody(item, field, value, folderID)
	if err != nil {
		return err
	}
	if _, err := w.Client.call(http.MethodPost, "/object/item", json.RawMessage(body)); err != nil {
		return fmt.Errorf("bw serve create item %q: %w", item, err)
	}
	return nil
}

// ResolveFolder returns the id of the folder named name, creating it when absent.
// Empty name → empty id (no folder declared is a no-op, not an error), matching
// BWPut.ResolveFolder's contract exactly.
//
// It carries the same accepted limitation as the CLI implementation: idempotent for
// sequential calls, NOT safe against concurrent ones, since Bitwarden permits duplicate
// folder names and two racing processes can both observe the folder absent (OPS-028
// adversarial review). `dotf secrets` has no concurrent-writer story anywhere in its
// design, so this is a limitation, not a regression.
func (w BWServeWriter) ResolveFolder(name string) (string, error) {
	if name == "" {
		return "", nil
	}
	data, err := w.Client.call(http.MethodGet, "/list/object/folders", nil)
	if err != nil {
		return "", fmt.Errorf("bw serve list folders: %w", err)
	}
	var list bwServeListData
	if err := json.Unmarshal(data, &list); err != nil {
		return "", fmt.Errorf("bw serve list folders: unparseable data: %w", err)
	}
	for _, f := range list.Data {
		if f.Name == name {
			return f.ID, nil
		}
	}
	created, err := w.Client.call(http.MethodPost, "/object/folder", map[string]string{"name": name})
	if err != nil {
		return "", fmt.Errorf("bw serve create folder %q: %w", name, err)
	}
	var f bwServeListItem
	if err := json.Unmarshal(created, &f); err != nil {
		return "", fmt.Errorf("bw serve create folder %q: unparseable data: %w", name, err)
	}
	if f.ID == "" {
		return "", fmt.Errorf("bw serve create folder %q: response carried no id", name)
	}
	return f.ID, nil
}
