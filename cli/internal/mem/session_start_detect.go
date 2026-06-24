package mem

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/memlink"
)

// This file ports the Claude session-start "detection" injectors from
// claude-session-start.sh (CLI-025 PR2b): env-contract drift (doctor --quick), hive
// project / work-SDK matching, and the auto-memory link. Like the metric injectors,
// each returns the exact CONTEXT_LINES contribution (with leading framing) or "".

// doctorDrift formats the env-contract drift block from a `dotf doctor --quick`
// run's output, surfacing only its [WARN]/[FAIL] lines. The block is double-newline
// framed (it follows the always-present [sdd] reminder). Empty when there is no
// drift. The caller runs doctor and gates on a deployed contract; this is the pure
// formatter so it stays unit-testable.
func doctorDrift(doctorQuickOutput string) string {
	var drift []string
	for line := range strings.SplitSeq(doctorQuickOutput, "\n") {
		if strings.HasPrefix(line, "  [WARN]") || strings.HasPrefix(line, "  [FAIL]") {
			drift = append(drift, line)
		}
	}
	if len(drift) == 0 {
		return ""
	}
	return "\n\n[doctor] env-contract drift detected (run 'dotf doctor --fix'):\n" + strings.Join(drift, "\n")
}

// hiveProject emits the hive-vault MCP hint when CWD maps to a vault project: first
// a personal 10_projects/<repo> match, else a 50_work/45-development work-SDK
// family/component slug match. Empty when neither matches.
func hiveProject(cwd, vault string) string {
	repo := filepath.Base(cwd)
	if isDir(filepath.Join(vault, "10_projects", repo)) {
		return hiveProjectBlock(repo)
	}
	return workSDKBlock(cwd, vault)
}

func hiveProjectBlock(repo string) string {
	return "\n[hive] Project '" + repo + "' found in vault. Use hive-vault MCP tools for on-demand context:" +
		"\n  - vault_query(project=\"" + repo + "\", section=\"context\") — project overview" +
		"\n  - vault_query(project=\"" + repo + "\", section=\"tasks\") — active backlog" +
		"\n  - vault_search(query=\"...\") — search across vault" +
		"\n  - vault_query(project=\"_meta\", path=\"patterns/...\") — cross-project patterns"
}

// workSDKBlock matches CWD path segments against vault work-SDK family/component dir
// slugs. It commits to the first family slug match: a component match yields the
// component block, otherwise the family-only block. Faithful to the shell's
// find_work_sdk_project (which returns on the first matching family).
func workSDKBlock(cwd, vault string) string {
	devDir := filepath.Join(vault, "50_work", "45-development")
	if !isDir(devDir) {
		return ""
	}
	cwdSlug := slugify(cwd, true)
	families, _ := os.ReadDir(devDir)
	for _, fam := range families {
		if !fam.IsDir() || !strings.Contains(cwdSlug, slugify(fam.Name(), false)) {
			continue
		}
		comps, _ := os.ReadDir(filepath.Join(devDir, fam.Name()))
		for _, comp := range comps {
			if comp.IsDir() && strings.Contains(cwdSlug, slugify(comp.Name(), false)) {
				return workSDKComponentBlock(fam.Name(), comp.Name())
			}
		}
		return workSDKFamilyBlock(fam.Name()) // family matched, no component
	}
	return ""
}

func workSDKComponentBlock(family, comp string) string {
	base := "50_work/45-development/" + family + "/" + comp
	return "\n[hive] Work SDK '" + comp + "' (family: " + family + ") found in vault. Use Hive MCP:" +
		"\n  - vault_query(project=\"_meta\", path=\"" + base + "/00-context.md\") — project context" +
		"\n  - vault_query(project=\"_meta\", path=\"" + base + "/memory/MEMORY.md\") — session memory" +
		"\n  - vault_query(project=\"_meta\", path=\"patterns/workflow-protocol\") — session protocol"
}

func workSDKFamilyBlock(family string) string {
	return "\n[hive] Work SDK family '" + family + "' found in vault. Use Hive MCP:" +
		"\n  - vault_query(project=\"_meta\", path=\"50_work/45-development/" + family + "/00-context.md\") — all repos in family"
}

// memorySymlink ensures the Claude per-project auto-memory link to the vault source
// and surfaces the "[auto-memory] Created …" line when it created one. Computes
// Claude's encoded target path and delegates the OS-agnostic link to memlink.
func memorySymlink(cwd, vault, home string) string {
	encoded := encodeProjectPath(cwd)
	target := filepath.Join(home, ".claude", "projects", encoded, "memory")
	msg, _ := memlink.Ensure(cwd, target, "", vault)
	if msg == "" {
		return ""
	}
	return "\n" + msg
}

// isDir reports whether p exists and is a directory.
func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// slugify lowercases s and keeps only [a-z0-9] (plus '/' when keepSlash), mirroring
// the shell's `tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9[/]'` used for path matching.
func slugify(s string, keepSlash bool) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case keepSlash && r == '/':
			b.WriteRune(r)
		}
	}
	return b.String()
}
