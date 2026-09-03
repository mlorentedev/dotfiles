package doctor

import (
	"path/filepath"
	"regexp"
	"strings"
)

// homeDeployEntry maps one file inside the deploy dir (~/.dotfiles) to where
// setup copies it under $HOME, and says whether the two are expected to stay
// byte-identical.
//
// The map cannot reuse isManagedDeployPath: that predicate serves the repo →
// deploy-dir leg, where a file keeps its relative path. On this leg the path
// changes (ssh/config → .ssh/config, tmux.conf → .tmux.conf), so the mapping has
// to be explicit — and explicit means it can drift from setup, which is what
// TestHomeDeployMapCoversSetup exists to prevent.
type homeDeployEntry struct {
	// src is relative to cfg.DotfilesDir; dst is relative to $HOME.
	src, dst string
	// contentChecked is false for files something other than setup legitimately
	// writes after deployment. Those are checked for existence only.
	contentChecked bool
	// exemptReason states the mechanism that makes contentChecked false. It is
	// required (TestHomeDeployExemptionsAreReasoned) because the defect this
	// spec found in setup's own NOTE was an exemption whose justification had
	// outlived its mechanism — the comment cited a `sed -i` that the OPS-040
	// purge removed.
	exemptReason string
}

// homeDeployMap mirrors setup-linux.sh's `deploy_file … "$HOME/…"` call sites.
// Adding one there without adding it here fails TestHomeDeployMapCoversSetup.
var homeDeployMap = []homeDeployEntry{
	{src: ".zshrc", dst: ".zshrc", exemptReason: "tool installers (opencode, bun, NVM, ggshield) append PATH/init lines post-deploy"},
	{src: ".bashrc", dst: ".bashrc", exemptReason: "same installer appends; setup only asserts existence here too"},
	{src: ".profile", dst: ".profile", exemptReason: "same installer appends"},
	{src: ".gitconfig", dst: ".gitconfig", exemptReason: "every `git config --global` rewrites it; measured drifting on a converged box 2026-09-02"},
	{src: "ssh/config", dst: ".ssh/config", contentChecked: true},
	{src: ".zsh/aliases.zsh", dst: ".zsh/aliases.zsh", contentChecked: true},
	{src: ".zsh/functions.zsh", dst: ".zsh/functions.zsh", contentChecked: true},
	{src: ".zsh/functions.sh", dst: ".zsh/functions.sh", contentChecked: true},
	{src: ".zsh/nvm.zsh", dst: ".zsh/nvm.zsh", contentChecked: true},
	{src: "tmux.conf", dst: ".tmux.conf", contentChecked: true},
	{src: ".inputrc", dst: ".inputrc", contentChecked: true},
}

// checkHomeDeployDrift covers the deploy-dir → $HOME leg of
// repo → ~/.dotfiles → $HOME. Before OPS-043 no doctor section compared content
// here: checkSymlinks and checkProfileFiles test existence, and PASS a real file
// that has drifted. setup-linux.sh's check_deployed was the only assertion, and
// setup-windows.ps1 never had one — so on Windows this leg was unguarded
// outright.
//
// It ports check_deployed's two severities exactly, so deleting the shell calls
// loses nothing: drifted content FAILs, and so does a symlink where a regular
// file is expected (ADR-012 moved deployment to copy; cmp would follow the link
// and checkSymlinks PASSes it, so nothing else catches that).
func checkHomeDeployDrift(sys *System, cfg *Config, rep *Report) {
	rep.Section("Deploy-dir↔$HOME drift")

	// Every entry is POSIX-only, since the map is derived from setup-linux.sh.
	// The Windows deploy targets need their own map and their own join guard —
	// OPS-046 (#1447), deliberately out of this spec's scope.
	if sys.GOOS == "windows" {
		rep.Skip("POSIX-only deploy targets (Windows map tracked by #1447)")
		return
	}

	home := sys.home()
	deploy := cfg.DotfilesDir
	if !isDir(deploy) {
		rep.Skip("deploy-dir absent: " + deploy + " (run setup)")
		return
	}

	for _, e := range homeDeployMap {
		src := filepath.Join(deploy, filepath.FromSlash(e.src))
		dst := filepath.Join(home, filepath.FromSlash(e.dst))

		// setup guards its conditional deploys on the SOURCE existing
		// (`[ -f "$DOTFILES_DIR/.gitconfig" ]`), so an absent source means "not
		// provisioned on this box", never "deploy failed".
		if !pathExists(src) {
			rep.Skip(e.dst + " not provisioned (" + e.src + " absent from deploy-dir)")
			continue
		}
		if isSymlink(dst) {
			rep.Fail(e.dst + " is a symlink (expected a regular file since ADR-012 — re-run setup)")
			continue
		}
		if !pathExists(dst) {
			rep.Fail(e.dst + " missing at " + dst + " (run setup-linux.sh)")
			continue
		}
		if !e.contentChecked {
			rep.Pass(e.dst + " exists (content drift expected: " + e.exemptReason + ")")
			continue
		}
		if filesEqual(src, dst) {
			rep.Pass(e.dst + " matches " + e.src)
			continue
		}
		rep.Fail(e.dst + " has drifted from " + src + " (edit in repo + run setup-linux.sh)")
	}
}

// setupDeployFileCall matches `deploy_file "$DOTFILES_DIR/<src>" "$HOME/<dst>"`,
// the only shape that deploys into $HOME. Calls targeting other roots
// (GEMINI_HOME, the MCP master config, the hive drop-in) are a different
// concern and are deliberately not matched.
var setupDeployFileCall = regexp.MustCompile(`deploy_file\s+"\$DOTFILES_DIR/([^"]+)"\s+"\$HOME/([^"]+)"`)

// setupHomeDeployPairs extracts src→dst from setup-linux.sh so the guard
// compares the map against the script rather than against a second hand-written
// list. Duplicate call sites (setup re-deploys the rc files at the tail) collapse
// into one entry.
func setupHomeDeployPairs(script string) map[string]string {
	pairs := map[string]string{}
	for _, m := range setupDeployFileCall.FindAllStringSubmatch(script, -1) {
		pairs[strings.TrimSpace(m[1])] = strings.TrimSpace(m[2])
	}
	return pairs
}
