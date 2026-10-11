package harness

import (
	"path/filepath"
	"strings"
)

// The deploy dir also carries the checkout files the shell and the setup
// scripts read from ~/.dotfiles rather than from the checkout: the rc files,
// versions.conf (sourced by .zshrc), scripts/, the encrypted secrets. Setup
// copied them by hand, so a machine that never runs setup (macOS, ADR-045)
// kept stale copies that nothing could refresh, and doctor's drift check
// compared a list of its own (#2224).
//
// These two lists are the one owner: Mirror copies them and doctor's
// repo↔deploy-dir drift check compares exactly them. setup-linux.sh still
// copies them early, before dotf is installed; a test pins its copy block to
// this set until that block is deleted.
var (
	// DeployDirFiles are single files at the checkout root.
	DeployDirFiles = []string{
		"versions.conf", "packages.json", "mcp-servers.json",
		".zshrc", ".bashrc", ".profile", ".inputrc", ".editorconfig", "tmux.conf",
		"env-contract.json",
	}
	// DeployDirTrees are directories copied whole.
	DeployDirTrees = []string{".zsh", "ssh", "scripts", "sensitive", "secrets"}
)

// IsDeployDirPath reports whether a slash-separated checkout path is one the
// deploy dir carries.
func IsDeployDirPath(rel string) bool {
	for _, f := range DeployDirFiles {
		if rel == f {
			return true
		}
	}
	for _, d := range DeployDirTrees {
		if strings.HasPrefix(rel, d+"/") {
			return true
		}
	}
	return false
}

// mirrorDeployDir copies the deploy-dir set. A path the checkout lacks is not
// an error: the drift check compares only files present on both sides, and a
// test asserts every entry exists in the real checkout, so a rename cannot
// drop one silently.
func mirrorDeployDir(repoRoot, deployDir string, ignored map[string]bool, dryRun bool, res *MirrorResult) error {
	for _, rel := range DeployDirFiles {
		src := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if !isRegular(src) {
			continue
		}
		if err := mirrorFile(src, filepath.Join(deployDir, filepath.FromSlash(rel)), dryRun, res); err != nil {
			return err
		}
	}
	for _, sub := range DeployDirTrees {
		if !isDir(filepath.Join(repoRoot, sub)) {
			continue
		}
		if err := mirrorTree(repoRoot, deployDir, sub, ignored, dryRun, res); err != nil {
			return err
		}
	}
	return nil
}
