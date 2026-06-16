package vault

import (
	"os"
	"path/filepath"
	"strings"
)

// workEntryFiles maps an embedded template to its path under the component entry
// dir. The parent family context is handled separately (create-if-absent).
var workEntryFiles = []struct{ template, dest string }{
	{"work-sdk-context.md", "00-context.md"},
	{"work-sdk-memory.md", filepath.Join("memory", "MEMORY.md")},
}

// WorkEntryOptions configures WriteWorkEntry. VaultPath must already exist (the
// caller resolves it via ResolveVaultStrict). Family/Component become directory
// names and are validated against traversal.
type WorkEntryOptions struct {
	VaultPath string // existing vault root
	Family    string
	Component string
	Date      string // YYYY-MM-DD
	Force     bool   // regenerate component files even if present
}

// WorkResult reports what WriteWorkEntry did.
type WorkResult struct {
	EntryDir string   // 50_work/45-development/<family>/<component>
	Created  []string // entry-relative files written this run
	Skipped  []string // entry-relative files left untouched (present, no --force)
	Family   string   // "created" | "exists" — the parent family context
}

// WriteWorkEntry scaffolds the work-SDK vault entry under
// VaultPath/50_work/45-development/<family>/<component>/ from embedded templates:
// 00-context.md + memory/MEMORY.md (skip-if-present, regenerated under Force), and
// the parent <family>/00-context.md created only when absent (never clobbered —
// it accumulates the family's repo table across components). It restores the
// capability removed from init-project.sh in CLI-014 (#388).
func WriteWorkEntry(opts WorkEntryOptions) (WorkResult, error) {
	if err := validateSlug("family", opts.Family); err != nil {
		return WorkResult{}, err
	}
	if err := validateSlug("component", opts.Component); err != nil {
		return WorkResult{}, err
	}

	devRoot := filepath.Join(opts.VaultPath, "50_work", "45-development")
	familyDir := filepath.Join(devRoot, opts.Family)
	entryDir := filepath.Join(familyDir, opts.Component)
	res := WorkResult{EntryDir: entryDir}

	if err := os.MkdirAll(filepath.Join(entryDir, "memory"), 0o755); err != nil {
		return res, err
	}

	repl := strings.NewReplacer(
		"{{family}}", opts.Family,
		"{{component}}", opts.Component,
		"{{date}}", opts.Date,
	)

	for _, f := range workEntryFiles {
		dest := filepath.Join(entryDir, f.dest)
		if fileExists(dest) && !opts.Force {
			res.Skipped = append(res.Skipped, f.dest)
			continue
		}
		if err := renderTemplate(f.template, dest, repl); err != nil {
			return res, err
		}
		res.Created = append(res.Created, f.dest)
	}

	// Parent family context: create-if-absent, never overwritten (even with
	// --force) — it carries the family's accumulated repo table.
	familyFile := filepath.Join(familyDir, "00-context.md")
	if fileExists(familyFile) {
		res.Family = "exists"
	} else {
		if err := renderTemplate("work-sdk-family.md", familyFile, repl); err != nil {
			return res, err
		}
		res.Family = "created"
	}

	return res, nil
}

// renderTemplate reads an embedded template, applies the replacer, and writes it
// to dest (0644). The parent dir must already exist.
func renderTemplate(name, dest string, repl *strings.Replacer) error {
	raw, err := readTemplate(name)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, []byte(repl.Replace(string(raw))), 0o644)
}
