package secrets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

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
			return "", fmt.Errorf("field %q requested but item has no login block", field)
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
	return "", fmt.Errorf("field %q not found on item", field)
}
