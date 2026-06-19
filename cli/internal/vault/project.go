package vault

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// projectEntryFiles maps an embedded template to its path under the personal-
// project vault entry dir. Note: NO 11-tasks.md — task state lives in the
// bitácora GitHub Project (ADR-018), not the vault.
//
// These templates are embed-only (no drift guard yet): they diverge from the
// vault SSOT's project-context.md / agent-memory.md rather than mirror them.
// Reconciling that — and reaching drift parity with the work-SDK set — is
// tracked in #400 (kept out of #395 to preserve byte-identical dotf init output).
var projectEntryFiles = []struct{ template, dest string }{
	{"vault-context.md", "00-context.md"},
	{"vault-roadmap.md", "10-roadmap.md"},
	{"vault-memory.md", filepath.Join("memory", "MEMORY.md")},
}

// ProjectEntryOptions configures WriteProjectEntry. VaultPath must already exist;
// the caller resolves it (dotf init via ResolveVault, dotf vault project via
// ResolveVaultStrict). An empty VaultPath is the skip signal dotf init relies on
// for an absent vault / --skip-vault.
type ProjectEntryOptions struct {
	VaultPath         string // existing vault root, or "" to skip (dotf init path)
	RepoRoot          string // absolute path of the repo
	Stack             string
	Date              string // YYYY-MM-DD
	ClaudeProjectsDir string // ~/.claude/projects, for the memory symlink (best-effort)
	Force             bool   // regenerate entry files even if present
}

// ProjectResult reports what WriteProjectEntry did.
type ProjectResult struct {
	Action   string   // "written" | "skipped"
	Reason   string   // populated for "skipped"
	EntryDir string   // the 10_projects/<repo> dir
	Created  []string // entry-relative files written this run
	Skipped  []string // entry-relative files left untouched (present, no --force)
	Symlink  string   // the memory symlink created, if any
}

// WriteProjectEntry creates the personal-project vault entry under
// VaultPath/10_projects/<repo>/ from embedded templates (skip-if-present, or
// regenerated under Force) and, on non-Windows, eagerly links the Claude
// auto-memory dir to the entry's memory/ (init-project.sh did this so the
// symlink doesn't wait for the first session). It is a no-op skip when VaultPath
// is empty — the signal dotf init uses for an absent vault / --skip-vault.
//
// Extracted from initrepo.WriteVaultEntry in CLI-015 PR2 (#395): one renderer,
// two entry points (the dotf init orchestrator + dotf vault project).
func WriteProjectEntry(opts ProjectEntryOptions) (ProjectResult, error) {
	if opts.VaultPath == "" {
		return ProjectResult{Action: "skipped", Reason: "no vault present (or --skip-vault)"}, nil
	}

	repo := filepath.Base(opts.RepoRoot)
	entryDir := filepath.Join(opts.VaultPath, "10_projects", repo)
	res := ProjectResult{Action: "written", EntryDir: entryDir}

	if err := os.MkdirAll(filepath.Join(entryDir, "memory"), 0o755); err != nil {
		return res, err
	}

	repl := strings.NewReplacer("{{repo}}", repo, "{{stack}}", opts.Stack, "{{date}}", opts.Date)
	for _, f := range projectEntryFiles {
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

	res.Symlink = linkMemory(opts.ClaudeProjectsDir, opts.RepoRoot, filepath.Join(entryDir, "memory"))
	return res, nil
}

// linkMemory best-effort links <claudeProjectsDir>/<encoded-repo-path>/memory to
// the vault entry's memory dir, mirroring Claude Code's path encoding (abs path
// with separators -> '-'). Windows uses junctions managed by setup, so this is
// non-Windows only. Any failure is swallowed (the session hook re-creates it).
func linkMemory(claudeProjectsDir, repoRoot, target string) string {
	if claudeProjectsDir == "" || runtime.GOOS == "windows" {
		return ""
	}
	encoded := strings.ReplaceAll(repoRoot, string(filepath.Separator), "-")
	link := filepath.Join(claudeProjectsDir, encoded, "memory")
	if fileExists(link) {
		return ""
	}
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return ""
	}
	if err := os.Symlink(target, link); err != nil {
		return ""
	}
	return link
}
