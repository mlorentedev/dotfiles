package deploy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// RecordsRel is where, under dotf's state directory, a merge entry records the
// source it last merged (#2570): <StateDir>/deploy/merged/<name>.json.
const RecordsRel = "deploy/merged"

// RecordsDir is the records directory under state, dotf's state directory.
func RecordsDir(state string) string { return filepath.Join(state, filepath.FromSlash(RecordsRel)) }

// MergeRecords is the records directory a run over targets uses. Only a merge
// entry needs one, so a state directory that cannot be placed fails the run
// only when one of targets merges; before records existed, nothing did.
func MergeRecords(targets []Config, stateDir func() (string, error)) (string, error) {
	for _, c := range targets {
		if c.strategy() != StrategyMerge {
			continue
		}
		state, err := stateDir()
		if err != nil {
			return "", fmt.Errorf("merge entries record what they manage under dotf's state directory: %w", err)
		}
		return RecordsDir(state), nil
	}
	return "", nil
}

const recordVersion = 1

// mergeRecord is what one merge entry managed at its last deploy: the source
// as it was merged, after paths rendering, kept as text so it decodes with the
// codec that merged it and a TOML integer or datetime compares exactly.
//
// The record is per machine, as the destination is, and holds repo content and
// resolved local paths only: a merge entry cannot render secrets.
type mergeRecord struct {
	Version int    `json:"version"`
	Dst     string `json:"dst"`
	Source  string `json:"source"`
}

// recordMerge writes the record a landed or in-sync plan carries. The deploy
// already happened, so the error says so: the operator must not redeploy
// thinking nothing landed.
func recordMerge(dir string, c Config, p Plan) error {
	if p.record == nil {
		return nil
	}
	if err := writeRecord(dir, c, p.Dst, p.record); err != nil {
		return fmt.Errorf("config %q: deployed %s, but could not record what it manages (%w); the next deploy removes nothing it retired since", c.Name, p.Dst, err)
	}
	return nil
}

func recordPath(dir, name string) string { return filepath.Join(dir, name+".json") }

// readPrior is the document c managed at dst on its last deploy, or nil when
// there is nothing to compare with: no records directory, no record, a record
// written for another destination, or one in a version this binary does not
// read. nil removes nothing. A record that exists and does not parse is an
// error: reading it as nil would be safe, but would hide that the history is
// gone.
func readPrior(dir string, c Config, dst string) (map[string]any, error) {
	if dir == "" {
		return nil, nil
	}
	path := recordPath(dir, c.Name)
	raw, err := os.ReadFile(path) //nolint:gosec // dotf's own state directory
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("merge record %s: %w", path, err)
	}
	corrupt := func(err error) error {
		return fmt.Errorf("merge record %s does not parse (%w); delete it to start over: the next deploy then removes nothing and records again", path, err)
	}
	var r mergeRecord
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, corrupt(err)
	}
	if r.Version != recordVersion || r.Dst != dst {
		return nil, nil
	}
	prior, err := mergeFormatFor(dst).decodeSource([]byte(r.Source))
	if err != nil {
		return nil, corrupt(err)
	}
	return prior, nil
}

// writeRecord records src as what c manages at dst. An unchanged record is not
// rewritten. The write is atomic: a crash leaves the old record or the new
// one, never a half that the next deploy refuses.
func writeRecord(dir string, c Config, dst string, src []byte) error {
	if dir == "" {
		return nil
	}
	data, err := json.MarshalIndent(mergeRecord{Version: recordVersion, Dst: dst, Source: string(src)}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := recordPath(dir, c.Name)
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) { //nolint:gosec // dotf's own state directory
		return nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".record-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op once renamed away
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// retire removes from existing what prior managed and managed no longer
// declares, wherever existing still holds the value prior recorded: a value
// the user changed since is theirs. A table emptied by the removal goes too.
// prefix is the dotted path of the tables being walked, as in deepMerge.
func retire(existing, prior, managed map[string]any, listKeys map[string]string, prefix string) bool {
	changed := false
	for key, priorValue := range prior {
		existingValue, present := existing[key]
		if !present {
			continue
		}
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		managedValue, declared := managed[key]
		if !declared && jsonEqual(existingValue, priorValue) {
			delete(existing, key)
			changed = true
			continue
		}
		switch pv := priorValue.(type) {
		case map[string]any:
			ev, ok := existingValue.(map[string]any)
			if !ok {
				continue
			}
			mv, _ := managedValue.(map[string]any)
			if !declared || mv != nil {
				if retire(ev, pv, mv, listKeys, path) {
					changed = true
					if len(ev) == 0 && !declared {
						delete(existing, key)
					}
				}
			}
		case []any:
			ev, ok := existingValue.([]any)
			if !ok {
				continue
			}
			mv, isList := managedValue.([]any)
			if declared && !isList {
				continue
			}
			field, identified := listKeys[path]
			kept, removed := retireElements(ev, pv, mv, field, identified)
			if !removed {
				continue
			}
			changed = true
			if len(kept) == 0 && !declared {
				delete(existing, key)
			} else {
				existing[key] = kept
			}
		}
	}
	return changed
}

// retireElements drops from existing each element prior declared and managed
// no longer does, while it is still equal to what prior recorded. Under
// listKeys an element is still declared when its identity is: an edited
// binding is replaced by mergeByIdentity, not removed.
func retireElements(existing, prior, managed []any, field string, identified bool) ([]any, bool) {
	var retired []any
	for _, p := range prior {
		if identified {
			if table, ok := p.(map[string]any); ok {
				if id, ok := table[field]; ok && identityIndex(id, managed, field) >= 0 {
					continue
				}
			}
		} else if containsEqual(managed, p) {
			continue
		}
		retired = append(retired, p)
	}
	out := make([]any, 0, len(existing))
	removed := false
	for _, e := range existing {
		if containsEqual(retired, e) {
			removed = true
			continue
		}
		out = append(out, e)
	}
	return out, removed
}

func containsEqual(list []any, v any) bool {
	for _, e := range list {
		if jsonEqual(e, v) {
			return true
		}
	}
	return false
}
