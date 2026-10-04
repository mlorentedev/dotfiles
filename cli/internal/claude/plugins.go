package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PluginsRel is the plugin list, relative to the repo root.
const PluginsRel = "ai/claude/plugins.json"

// LoadPlugins reads the plugin list. Strict for the same reason the deploy
// manifest is: a file this binary cannot fully read is one it must not act on.
func LoadPlugins(path string) ([]string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		return nil, err
	}
	var doc struct {
		Comment []string `json:"$comment"`
		Plugins []string `json:"plugins"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", PluginsRel, err)
	}
	seen := map[string]bool{}
	for _, id := range doc.Plugins {
		name, market, ok := strings.Cut(id, "@")
		if !ok || name == "" || market == "" || strings.Contains(market, "@") {
			return nil, fmt.Errorf("%s: %q is not <plugin>@<marketplace>", PluginsRel, id)
		}
		if seen[id] {
			return nil, fmt.Errorf("%s: duplicate plugin %q", PluginsRel, id)
		}
		seen[id] = true
	}
	return doc.Plugins, nil
}

// Runner is the claude CLI as the sync uses it: the seam a test fills with a
// fake instead of a real install.
type Runner interface {
	List() (string, error)   // `claude plugin list` output
	Install(id string) error // `claude plugin install <id>`
}

// Syncer installs the declared plugins a box lacks, each claude call inside
// the snapshot guard on ClaudeJSON.
type Syncer struct {
	Run        Runner
	ClaudeJSON string
	Floor      int
}

// PluginReport is what one Sync found and did. Added holds only installs that
// succeeded, or would be attempted on a dry run; a failed install is in Failed
// and nowhere else (AC7; the PowerShell twin counts attempts, #1491).
type PluginReport struct {
	Present, Added, Failed []string
	Restored               int // .claude.json restores the guard made
}

// Sync installs every id in ids that `claude plugin list` does not mention.
// A plugin counts as present when its id appears in the list output, which is
// the test both twins use. A list that fails is returned, not read as empty:
// treating a broken CLI as "nothing installed" reinstalls everything.
func (s Syncer) Sync(ids []string, dryRun bool) (PluginReport, error) {
	var rep PluginReport
	var listed string
	restored, err := Guard(s.ClaudeJSON, s.Floor, func() error {
		var err error
		listed, err = s.Run.List()
		return err
	})
	rep.Restored += b2i(restored)
	if err != nil {
		return rep, fmt.Errorf("claude plugin list: %w", err)
	}
	for _, id := range ids {
		switch {
		case strings.Contains(listed, id):
			rep.Present = append(rep.Present, id)
		case dryRun:
			rep.Added = append(rep.Added, id)
		default:
			restored, err := Guard(s.ClaudeJSON, s.Floor, func() error { return s.Run.Install(id) })
			rep.Restored += b2i(restored)
			if err != nil {
				rep.Failed = append(rep.Failed, id)
				continue
			}
			rep.Added = append(rep.Added, id)
		}
	}
	return rep, nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
