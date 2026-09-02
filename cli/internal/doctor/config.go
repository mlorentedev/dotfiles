package doctor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

// Config holds the resolved locations and parsed manifests a doctor run needs:
// the deployed dotfiles dir, the version pins from versions.conf, and the paths
// to env-contract.json and versions.conf (kept for the provenance line that makes
// a stale-copy read self-diagnosing). It mirrors how healthcheck.sh sourced
// versions.conf and doctor.sh resolved the contract.
type Config struct {
	DotfilesDir string
	Versions    map[string]string
	// RepoDir is the resolved checkout, or "" when the run is not inside one.
	// loadConfig has always computed this to resolve files repo-first; it was
	// discarded afterwards. The dotf-provenance check (#1158) needs the checkout
	// ITSELF, not a file inside it — it asks git what HEAD is — and "" is the
	// legitimate no-checkout state that check must SKIP on rather than guess at.
	RepoDir      string
	ContractPath string
	VersionsPath string
}

// loadConfig resolves DOTFILES_DIR (defaulting to $HOME/.dotfiles), then locates
// versions.conf and env-contract.json **repo-first** — the same precedence
// `dotf env generate` uses (env.ResolveRepoFirst) — so doctor and env never read
// different copies of the same file and hand out contradictory stale/fresh
// verdicts on a machine whose deploy dir lags the repo (#697). The checkout is
// resolved like env does (DOTFILES_REPO_DIR when it points at a real dir, else
// the .git walk-up from startDir). A missing versions.conf is tolerated (version
// checks skip); a missing contract is reported by the caller.
func loadConfig(sys *System, startDir string) (*Config, error) {
	dotfilesDir := sys.env("DOTFILES_DIR", filepath.Join(sys.home(), ".dotfiles"))

	repoDir := sys.env("DOTFILES_REPO_DIR", "")
	if !isDir(repoDir) {
		repoDir, _ = findRepoRoot(startDir)
	}

	cfg := &Config{
		DotfilesDir:  dotfilesDir,
		Versions:     map[string]string{},
		RepoDir:      repoDir,
		ContractPath: envpkg.ResolveRepoFirst("env-contract.json", repoDir, dotfilesDir, startDir),
		VersionsPath: envpkg.ResolveRepoFirst("versions.conf", repoDir, dotfilesDir, startDir),
	}

	if cfg.VersionsPath != "" {
		v, err := parseVersionsConf(cfg.VersionsPath)
		if err != nil {
			return nil, err
		}
		cfg.Versions = v
	}

	return cfg, nil
}

// parseVersionsConf reads the KEY=VALUE manifest (no export, no quotes per its
// header). Comments (#) and blank lines are skipped; surrounding whitespace is
// trimmed. This is the native replacement for sourcing the file in shell.
func parseVersionsConf(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open versions.conf: %w", err)
	}
	defer func() { _ = f.Close() }()

	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read versions.conf: %w", err)
	}
	return out, nil
}

// findRepoRoot walks up from start until it finds a .git entry (dir in a normal
// clone, file in a worktree). Used only as a fallback contract/versions source.
func findRepoRoot(start string) (string, error) {
	if start == "" {
		return "", fmt.Errorf("no start dir")
	}
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .git from %s", start)
		}
		dir = parent
	}
}
