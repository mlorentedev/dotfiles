package deploy

import (
	"fmt"
	"os"
)

// EditJSON applies edit to an installed JSON destination of c and, when edit
// reports a change, writes the result through the same staged write Deploy
// uses: a temp file beside dst, the one-time backup, an atomic rename and the
// declared mode. It is for a repair that removes what a merge never will
// (#2350): deepMerge unions lists, so an entry the source dropped stays on
// every machine that had it. A destination that is absent is an error, not an
// empty document, so a repair never creates a file deploy has not.
func EditJSON(c Config, dst string, edit func(doc map[string]any) bool) (bool, error) {
	mode, err := c.FileMode()
	if err != nil {
		return false, fmt.Errorf("config %q: %w", c.Name, err)
	}
	raw, err := os.ReadFile(dst) //nolint:gosec // a manifest-declared destination
	if err != nil {
		return false, fmt.Errorf("config %q: %w", c.Name, err)
	}
	doc, err := decodeJSONMap(stripLineComments(raw))
	if err != nil {
		return false, fmt.Errorf("config %q: %s is not a JSON object: %w", c.Name, dst, err)
	}
	if !edit(doc) {
		return false, nil
	}
	data, err := encodeJSON(doc)
	if err != nil {
		return false, fmt.Errorf("config %q: %w", c.Name, err)
	}
	install := installMode(dst, mode)
	staged, err := stage(c, dst, data, install)
	if err != nil {
		return false, err
	}
	defer func() { _ = os.Remove(staged) }() // no-op once renamed away
	if _, err := backupOnce(dst, install); err != nil {
		return false, err
	}
	return true, commit(c, staged, dst, install)
}
