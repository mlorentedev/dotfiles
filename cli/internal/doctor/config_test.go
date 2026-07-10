package doctor

import (
	"path/filepath"
	"testing"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

// TestLoadConfigResolvesRepoFirst is the #697 anti-drift guard: when both the
// checkout copy and the deployed copy of env-contract.json / versions.conf exist,
// doctor must resolve the checkout copy (the fresher SSOT) — the same one
// `dotf env generate` reads — so the two never hand out contradictory
// stale/fresh verdicts. It also confirms doctor reads the repo's pins, not a
// stale deployed pin that would produce nonsensical version-drift directions.
func TestLoadConfigResolvesRepoFirst(t *testing.T) {
	repo := t.TempDir()
	deployed := t.TempDir()

	repoContract := filepath.Join(repo, "env-contract.json")
	writeFile(t, repoContract, `{"env_vars":[]}`)
	writeFile(t, filepath.Join(deployed, "env-contract.json"), `{"env_vars":[]}`)

	repoVersions := filepath.Join(repo, "versions.conf")
	writeFile(t, repoVersions, "DOTF_VERSION=9.9.9\n")
	writeFile(t, filepath.Join(deployed, "versions.conf"), "DOTF_VERSION=0.0.1\n")

	sys := newSys(map[string]string{"DOTFILES_DIR": deployed, "DOTFILES_REPO_DIR": repo}, nil, nil)
	cfg, err := loadConfig(sys, repo)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.ContractPath != repoContract {
		t.Errorf("ContractPath = %q, want repo copy %q", cfg.ContractPath, repoContract)
	}
	if cfg.VersionsPath != repoVersions {
		t.Errorf("VersionsPath = %q, want repo copy %q", cfg.VersionsPath, repoVersions)
	}
	if cfg.Versions["DOTF_VERSION"] != "9.9.9" {
		t.Errorf("read a stale deployed pin: DOTF_VERSION=%q, want 9.9.9 from the repo", cfg.Versions["DOTF_VERSION"])
	}

	// Cross-check: env's own resolver (globals) agrees with doctor's (seam), so the
	// two resolution paths can never diverge again (#697).
	t.Setenv("DOTFILES_REPO_DIR", repo)
	t.Setenv("DOTFILES_DIR", deployed)
	if got := envpkg.ResolveContractPath(); got != repoContract {
		t.Errorf("env.ResolveContractPath() = %q, want repo copy %q (must agree with doctor)", got, repoContract)
	}
}
