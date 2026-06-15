package initrepo

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// vaultEntryFiles maps an embedded template to its path under the vault entry
// dir. Note: NO 11-tasks.md — task state lives in the bitácora GitHub Project
// (ADR-018), not the vault.
var vaultEntryFiles = []struct{ template, dest string }{
	{"vault-context.md", "00-context.md"},
	{"vault-roadmap.md", "10-roadmap.md"},
	{"vault-memory.md", filepath.Join("memory", "MEMORY.md")},
}

// ResolveVault returns the vault root if it exists, honoring $VAULT_PATH and
// falling back to ~/Projects/knowledge. It returns "" when no vault is present —
// the signal the orchestrator uses to skip the (host-coupled) vault entry.
func ResolveVault() string {
	v := os.Getenv("VAULT_PATH")
	if v == "" {
		if home, err := os.UserHomeDir(); err == nil {
			v = filepath.Join(home, "Projects", "knowledge")
		}
	}
	if v == "" {
		return ""
	}
	if info, err := os.Stat(v); err == nil && info.IsDir() {
		return v
	}
	return ""
}

// VaultEntryOptions configures WriteVaultEntry. VaultPath must already exist (the
// caller resolves it via ResolveVault); an empty VaultPath is the skip signal.
type VaultEntryOptions struct {
	VaultPath         string // existing vault root, or "" to skip
	RepoRoot          string // absolute path of the new repo
	Stack             string
	Date              string // YYYY-MM-DD
	ClaudeProjectsDir string // ~/.claude/projects, for the memory symlink (best-effort)
}

// VaultResult reports what WriteVaultEntry did.
type VaultResult struct {
	Action   string   // "written" | "skipped"
	Reason   string   // populated for "skipped"
	EntryDir string   // the 10_projects/<repo> dir
	Created  []string // entry-relative files written this run
	Symlink  string   // the memory symlink created, if any
}

// WriteVaultEntry creates the vault project entry under
// VaultPath/10_projects/<repo>/ from embedded templates (skip-if-present) and,
// on non-Windows, eagerly links the Claude auto-memory dir to the entry's
// memory/ (init-project.sh did this so the symlink doesn't wait for the first
// session). It is a no-op skip when VaultPath is empty.
func WriteVaultEntry(opts VaultEntryOptions) (VaultResult, error) {
	if opts.VaultPath == "" {
		return VaultResult{Action: "skipped", Reason: "no vault present (or --skip-vault)"}, nil
	}

	repo := filepath.Base(opts.RepoRoot)
	entryDir := filepath.Join(opts.VaultPath, "10_projects", repo)
	res := VaultResult{Action: "written", EntryDir: entryDir}

	if err := os.MkdirAll(filepath.Join(entryDir, "memory"), 0o755); err != nil {
		return res, err
	}

	repl := strings.NewReplacer("{{repo}}", repo, "{{stack}}", opts.Stack, "{{date}}", opts.Date)
	for _, f := range vaultEntryFiles {
		dest := filepath.Join(entryDir, f.dest)
		if fileExists(dest) {
			continue
		}
		raw, err := ReadTemplate(f.template)
		if err != nil {
			return res, err
		}
		if err := os.WriteFile(dest, []byte(repl.Replace(string(raw))), 0o644); err != nil {
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
