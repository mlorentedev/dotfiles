package vault

import (
	"os"
	"os/exec"
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
	{"vault-context.md", "context.md"},
	{"vault-roadmap.md", "roadmap.md"},
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

	// A repository checked out under another name than its vault slug is
	// already tracked; a second entry named after the directory would split
	// its memory and context in two (#2115).
	url := withoutCredentials(originURL(opts.RepoRoot))
	if slug := trackedSlug(opts.VaultPath, url); slug != "" && slug != repo {
		res.Action = "skipped"
		res.Reason = "repository already tracked as 10_projects/" + slug + " (repo_url " + normalizeRepoURL(url) + ")"
		res.EntryDir = filepath.Join(opts.VaultPath, "10_projects", slug)
		return res, nil
	}

	if err := os.MkdirAll(filepath.Join(entryDir, "memory"), 0o755); err != nil {
		return res, err
	}

	repl := strings.NewReplacer("{{repo}}", repo, "{{stack}}", opts.Stack, "{{date}}", opts.Date,
		"{{repo_url}}", url)
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

// originURL is repoRoot's origin remote URL, or "" when it has none.
func originURL(repoRoot string) string {
	out, err := exec.Command("git", "-C", repoRoot, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// withoutCredentials drops a password or token from a URL's user info
// (https://user:token@host/... becomes https://host/...). The origin is written
// into the vault's context.md and printed in reports, and a self-hosted remote
// often embeds a token. An ssh user with no password (git@host) is kept: it is
// part of how the remote is reached, not a secret.
func withoutCredentials(u string) string {
	scheme, rest, ok := strings.Cut(u, "://")
	if !ok {
		return u
	}
	end := strings.IndexAny(rest+"/", "/")
	at := strings.LastIndex(rest[:end], "@")
	if at < 0 {
		return u
	}
	if userinfo := rest[:at]; strings.Contains(userinfo, ":") || scheme == "https" || scheme == "http" {
		return scheme + "://" + rest[at+1:]
	}
	return u
}

// trackedSlug is the 10_projects/<slug> whose context.md repo_url names the
// same repository as url, or "" when none does, url is empty, or two entries
// claim it (an ambiguity is not resolved by guessing).
func trackedSlug(vaultPath, url string) string {
	want := normalizeRepoURL(url)
	if want == "" {
		return ""
	}
	contexts, _ := filepath.Glob(filepath.Join(vaultPath, "10_projects", "*", "context.md"))
	match := ""
	for _, ctx := range contexts {
		if normalizeRepoURL(frontmatterValue(ctx, "repo_url")) != want {
			continue
		}
		if match != "" {
			return ""
		}
		match = filepath.Base(filepath.Dir(ctx))
	}
	return match
}

// normalizeRepoURL reduces the SSH, scp-like and HTTPS spellings of one
// repository to host/path: git@github.com:o/r.git, ssh://git@github.com:22/o/r
// and https://github.com/o/r/ all become github.com/o/r. The host is
// lower-cased; the path keeps its case, as forges differ on that.
func normalizeRepoURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
	} else if at := strings.Index(u, "@"); at >= 0 && strings.Contains(u[at:], ":") {
		u = strings.Replace(u[at+1:], ":", "/", 1) // scp-like user@host:path
	}
	if at := strings.Index(u, "@"); at >= 0 && at < strings.Index(u+"/", "/") {
		u = u[at+1:] // user info
	}
	host, path, _ := strings.Cut(u, "/")
	if h, _, ok := strings.Cut(host, ":"); ok {
		host = h // port
	}
	path = strings.TrimSuffix(strings.TrimSuffix(path, "/"), ".git")
	if host == "" || path == "" {
		return ""
	}
	return strings.ToLower(host) + "/" + path
}

// frontmatterValue reads key from a note's leading frontmatter block, unquoted
// and without a trailing YAML comment, or "" when absent. A block with no
// closing delimiter is not frontmatter.
func frontmatterValue(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	value := ""
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return value
		}
		if v, ok := strings.CutPrefix(line, key+":"); ok && value == "" {
			v, _, _ = strings.Cut(v, " #")
			value = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}
