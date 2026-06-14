package doctor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the resolved locations and parsed manifests a doctor run needs:
// the deployed dotfiles dir, the version pins from versions.conf, and the path
// to env-contract.json. It mirrors how healthcheck.sh sourced versions.conf and
// doctor.sh resolved the contract.
type Config struct {
	DotfilesDir  string
	Versions     map[string]string
	ContractPath string
}

// loadConfig resolves DOTFILES_DIR (defaulting to $HOME/.dotfiles), then locates
// versions.conf and env-contract.json under it, falling back to the git repo
// root walked up from startDir (so `dotf doctor` works from inside a checkout,
// not just against a deployed ~/.dotfiles). A missing versions.conf is tolerated
// (version checks skip); a missing contract is reported by the caller.
func loadConfig(sys *System, startDir string) (*Config, error) {
	dotfilesDir := sys.env("DOTFILES_DIR", filepath.Join(sys.home(), ".dotfiles"))

	repoRoot, _ := findRepoRoot(startDir)

	cfg := &Config{
		DotfilesDir: dotfilesDir,
		Versions:    map[string]string{},
		ContractPath: firstExisting(
			filepath.Join(dotfilesDir, "env-contract.json"),
			joinIf(repoRoot, "env-contract.json"),
		),
	}

	if vp := firstExisting(
		filepath.Join(dotfilesDir, "versions.conf"),
		joinIf(repoRoot, "versions.conf"),
	); vp != "" {
		v, err := parseVersionsConf(vp)
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

// firstExisting returns the first path that exists, or "".
func firstExisting(paths ...string) string {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// joinIf joins dir/name, or returns "" when dir is empty.
func joinIf(dir, name string) string {
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, name)
}
