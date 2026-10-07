package initrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// Forge classifies a repository's origin remote. `dotf init` assumed GitHub and
// `main`; on a self-hosted Gitea remote with a `master` default branch it called
// origin absent and scaffolded CI that never ran on push (#1962).
type Forge string

const (
	ForgeAbsent Forge = "absent" // no origin remote
	ForgeGitHub Forge = "github" // a github.com remote
	// ForgeGitea is a non-GitHub remote whose checkout already carries a
	// .gitea/ or .forgejo/ directory. The remote URL alone cannot tell Gitea
	// from any other self-hosted forge without a network probe, which init
	// does not make (ADR-022 C7: host coupling degrades, it does not block).
	ForgeGitea Forge = "gitea"
	ForgeOther Forge = "other" // any other remote
)

// originURL returns the origin remote's URL, or "" when there is none.
func originURL(root string) string {
	out, err := exec.Command("git", "-C", root, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ClassifyOrigin reports root's forge and the origin URL it was read from.
func ClassifyOrigin(root string) (Forge, string) {
	url := originURL(root)
	switch {
	case url == "":
		return ForgeAbsent, ""
	case isGitHubRemote(url):
		return ForgeGitHub, url
	case giteaActionsDir(root) != "":
		return ForgeGitea, url
	default:
		return ForgeOther, url
	}
}

func isGitHubRemote(url string) bool {
	_, err := ParseOriginRepo(url)
	return err == nil
}

// giteaActionsDir is the workflow directory a Gitea-family checkout already
// declares (.forgejo/ wins, as Forgejo reads it first), or "" when it has none.
func giteaActionsDir(root string) string {
	for _, d := range []string{".forgejo", ".gitea"} {
		if st, err := os.Stat(filepath.Join(root, d)); err == nil && st.IsDir() {
			return filepath.Join(d, "workflows")
		}
	}
	return ""
}

// WorkflowDir is where a CI workflow belongs for root's forge. Every forge
// other than a recognised Gitea one gets .github/workflows, which Gitea also
// reads as a fallback, so the "other" case still runs there.
func WorkflowDir(root string) string {
	if f, _ := ClassifyOrigin(root); f == ForgeGitea {
		return giteaActionsDir(root)
	}
	return filepath.Join(".github", "workflows")
}

// DefaultBranch is the branch CI should trigger on: origin's HEAD when the
// clone recorded one, else the checkout's current branch (unborn included),
// else, on a detached HEAD, the local branches (the only one, or main, or
// master), else "main" for a directory that is not a repository yet.
func DefaultBranch(root string) string {
	if out, err := exec.Command("git", "-C", root, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD").Output(); err == nil {
		if b := strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/"); b != "" {
			return b
		}
	}
	if out, err := exec.Command("git", "-C", root, "symbolic-ref", "--quiet", "--short", "HEAD").Output(); err == nil {
		if b := strings.TrimSpace(string(out)); b != "" {
			return b
		}
	}
	if out, err := exec.Command("git", "-C", root, "for-each-ref", "--format=%(refname:short)", "refs/heads").Output(); err == nil {
		branches := strings.Fields(string(out))
		if len(branches) == 1 {
			return branches[0]
		}
		for _, b := range []string{"main", "master"} {
			if slices.Contains(branches, b) {
				return b
			}
		}
	}
	return "main"
}

// yamlBranch renders a branch name for a flow sequence, always double-quoted
// (YAML accepts JSON string syntax). A bare scalar is not always a string: a
// branch named 1.0, 2024 or on would be read as a number or a boolean.
func yamlBranch(b string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(b) + `"`
}

// originHost is the host of a remote URL, for reports. The full URL is never
// printed: a self-hosted remote often embeds a token (https://user:token@host/...),
// and a report line lands in terminals and agent transcripts.
func originHost(url string) string {
	u := url
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
	} else if at := strings.Index(u, "@"); at >= 0 {
		u = u[at+1:] // scp-like user@host:path
	}
	if at := strings.LastIndex(u, "@"); at >= 0 && at < strings.IndexAny(u+"/", "/") {
		u = u[at+1:]
	}
	host, _, _ := strings.Cut(u, "/")
	host, _, _ = strings.Cut(host, ":")
	if host == "" {
		return "a local path"
	}
	return host
}
