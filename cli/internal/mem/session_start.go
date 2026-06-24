// session-start is the second piece of the `dotf mem` noun (CLI-025 PR2a): the
// agent-AGNOSTIC session-brief core, ported faithfully from scripts/session-brief.sh
// (ADR-023, HARNESS-026). It owns the agent-independent vault signals — vault
// detection, vault-health, active/archived spec counts, lessons-staleness, and
// vault-baseline integrity — and renders them via --format=stdout|markdown.
//
// This is the layer opencode/agy/copilot consume (out-of-band, via the HARNESS-001
// compiler) and that the Claude adapter (PR2b) wraps in its additionalContext JSON.
// The output is contractually byte-equivalent to session-brief.sh, so every emitter
// mirrors the shell's exact spacing, leading-newline framing, and message text.
package mem

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// BriefOptions configures the agnostic session-brief core. Now and StaleDays are
// injected for deterministic staleness; ScriptsDir locates the sibling vault-health.sh
// (injected in tests, resolved from the env-contract in production).
type BriefOptions struct {
	Cwd        string
	ScriptsDir string
	StaleDays  int
	Now        time.Time
}

// ansiSeq strips the SGR colour codes vault-health.sh emits, matching the shell's
// `sed 's/\x1b\[[0-9;]*m//g'`.
var ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*m")

// Brief assembles the agnostic session-brief in session-brief.sh's canonical order
// — headline, vault-health, specs, lessons, baseline — and drops a leading blank
// line when there is no headline (the shell's `sed '1{/^$/d;}'`). The returned text
// carries no format framing; pass it to RenderBrief.
func Brief(opts BriefOptions) string {
	staleDays := opts.StaleDays
	if staleDays == 0 {
		staleDays = 14
	}

	vaultRoot := findVaultRoot(opts.Cwd)
	vaultName := ""
	if vaultRoot != "" {
		vaultName = filepath.Base(vaultRoot)
	}

	brief := vaultDetect(vaultRoot)
	brief += vaultHealth(vaultRoot, vaultName, opts.ScriptsDir)
	brief += specs(opts.Cwd)
	brief += lessonsStaleness(opts.Cwd, staleDays, opts.Now)
	brief += vaultBaseline(vaultRoot)

	// Drop a leading blank line when there was no headline (no-vault CWD): the
	// shell deletes line 1 only when empty, i.e. trims exactly one leading newline.
	return strings.TrimPrefix(brief, "\n")
}

// RenderBrief frames a brief for the given format, mirroring session-brief.sh's
// --format runner: "stdout" is the plain brief plus a trailing newline; "markdown"
// wraps it in a ```text fence. Any other value is the shell's usage error.
func RenderBrief(brief, format string) (string, error) {
	switch format {
	case "stdout":
		return brief + "\n", nil
	case "markdown":
		return "```text\n" + brief + "\n```\n", nil
	default:
		return "", fmt.Errorf("--format must be stdout or markdown (got: %s)", format)
	}
}

// findVaultRoot walks up from dir to the nearest .obsidian/ vault root, returning
// the root or "" — the shell's find_vault_root.
func findVaultRoot(dir string) string {
	if dir == "" {
		return ""
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, ".obsidian")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// vaultDetect is the "Obsidian vault detected" headline with no surrounding
// newlines (the caller owns the framing); empty when there is no root.
func vaultDetect(vaultRoot string) string {
	if vaultRoot == "" {
		return ""
	}
	return fmt.Sprintf("Obsidian vault detected: %s (%s)", filepath.Base(vaultRoot), vaultRoot)
}

// vaultHealth runs vault-health.sh and formats the GUI-down / pass / fail /
// not-installed cases with a single leading newline — the shell's sb_vault_health.
func vaultHealth(vaultRoot, vaultName, scriptsDir string) string {
	script := filepath.Join(scriptsDir, "vault-health.sh")
	if !isExecutable(script) {
		return fmt.Sprintf("\nvault-health.sh not found at %s — run dotfiles setup to install.", script)
	}

	// `bash <script>` (no `-c`): the script path is a trusted env-contract location,
	// not user input, and is passed as an argv element, so there is no shell-metachar
	// injection surface. This is the faithful port of the shell's `bash "$vault_health"`.
	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(), "VAULT_DIR="+vaultRoot, "VAULT_NAME="+vaultName)
	out, err := cmd.CombinedOutput()
	healthExit := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			healthExit = ee.ExitCode()
		} else {
			healthExit = -1 // bash unavailable / spawn failure
		}
	}
	clean := ansiSeq.ReplaceAllString(string(out), "")

	switch healthExit {
	case 2:
		// GUI down: GUI-dependent checks skipped, but the git-based integrity
		// check ran anyway — surface any FAIL lines.
		if fails := grepLines(clean, func(l string) bool { return strings.Contains(l, "FAIL") }); fails != "" {
			return fmt.Sprintf("\nObsidian GUI not running — GUI-dependent checks skipped. Integrity issues found:\n%s", fails)
		}
		return "\nObsidian GUI not running — vault health skipped. Run 'vault-health.sh' manually when GUI is up."
	case 0:
		return "\nVault health: ALL CHECKS PASSED"
	default:
		summary := grepLines(clean, func(l string) bool { return strings.HasPrefix(l, "Results:") })
		failures := grepLines(clean, func(l string) bool { return strings.Contains(l, "FAIL") })
		return fmt.Sprintf("\nVault health: %s\nIssues found:\n%s", summary, failures)
	}
}

// specs reports active/archived spec counts under <cwd>/specs, flagging any spec
// carrying unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags. Leading newline, or
// nothing when there is no specs/ dir or it is entirely empty — the shell's sb_specs.
func specs(cwd string) string {
	specsDir := filepath.Join(cwd, "specs")
	if info, err := os.Stat(specsDir); err != nil || !info.IsDir() {
		return ""
	}

	entries, _ := os.ReadDir(specsDir) // sorted, matching the shell's glob order
	activeCount := 0
	var drafts []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		activeCount++
		if dirHasDraftTag(filepath.Join(specsDir, e.Name())) {
			drafts = append(drafts, e.Name())
		}
	}

	archiveCount := 0
	if archive, err := os.ReadDir(filepath.Join(specsDir, "archive")); err == nil {
		for _, e := range archive {
			if e.IsDir() {
				archiveCount++
			}
		}
	}

	if activeCount == 0 && archiveCount == 0 {
		return ""
	}

	var msg strings.Builder
	fmt.Fprintf(&msg, "[specs] %d active, %d archived", activeCount, archiveCount)
	if len(drafts) > 0 {
		fmt.Fprintf(&msg, " — %d with unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags:", len(drafts))
		for _, name := range drafts {
			msg.WriteString("\n  - " + name)
		}
	}
	return "\n" + msg.String()
}

// dirHasDraftTag reports whether any file under dir contains a literal
// [AGENT-DRAFT] or [AGENT-SUGGESTION] tag — the shell's recursive grep -rlE.
func dirHasDraftTag(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if bytes.Contains(b, []byte("[AGENT-DRAFT]")) || bytes.Contains(b, []byte("[AGENT-SUGGESTION]")) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// lessonsStaleness flags a stale docs/lessons.md in the current repo (HARNESS-024):
// leading newline when older than staleDays, nothing when absent or fresh.
func lessonsStaleness(cwd string, staleDays int, now time.Time) string {
	lessons := filepath.Join(cwd, "docs", "lessons.md")
	info, err := os.Stat(lessons)
	if err != nil || info.IsDir() {
		return ""
	}
	if now.Sub(info.ModTime()) > time.Duration(staleDays)*24*time.Hour {
		return fmt.Sprintf("\n[lessons] docs/lessons.md not updated in >%d days — capture fixes/discoveries from this session (handoff Step 2) or note the gap (HARNESS-024).", staleDays)
	}
	return ""
}

// vaultBaseline is the defensive check that every 00_meta/skills/<n>/ has a
// non-empty SKILL.md and the static critical files exist (SDD-017b). Leading
// newline when issues found, otherwise nothing — the shell's sb_vault_baseline.
func vaultBaseline(vaultRoot string) string {
	if vaultRoot == "" {
		return ""
	}
	if info, err := os.Stat(filepath.Join(vaultRoot, "00_meta")); err != nil || !info.IsDir() {
		return ""
	}

	var issues strings.Builder
	skillsDir := filepath.Join(vaultRoot, "00_meta", "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillMd := filepath.Join(skillsDir, e.Name(), "SKILL.md")
			info, err := os.Stat(skillMd)
			switch {
			case err != nil:
				fmt.Fprintf(&issues, "\n        MISSING: 00_meta/skills/%s/SKILL.md (skill dir exists but SKILL.md gone)", e.Name())
			case info.Size() == 0:
				fmt.Fprintf(&issues, "\n        EMPTY: 00_meta/skills/%s/SKILL.md (0 bytes — likely truncated)", e.Name())
			}
		}
	}

	// AGENTS.md is intentionally NOT required here (ADR-010 single-SSOT in dotfiles,
	// deployed not duplicated to the vault — BUG-023).
	for _, cf := range []string{"00_meta/patterns/_index.md", "00_meta/skills/README.md", "README.md"} {
		if _, err := os.Stat(filepath.Join(vaultRoot, filepath.FromSlash(cf))); err != nil {
			fmt.Fprintf(&issues, "\n        MISSING: %s (canonical file)", cf)
		}
	}

	if issues.Len() == 0 {
		return ""
	}
	return fmt.Sprintf("\nVault baseline FAIL — canonical artifacts missing (likely auto-committed delete):%s\n        Recovery: cd %s; git log --diff-filter=D --name-only --pretty=format: -5 | sort -u; git show <commit>:<path> > <path>", issues.String(), vaultRoot)
}

// grepLines keeps the lines of s matching pred and joins them with newlines, with
// no trailing newline — the shell's `echo "$s" | grep … || echo ""` idiom under
// command substitution.
func grepLines(s string, pred func(string) bool) string {
	var kept []string
	for line := range strings.SplitSeq(s, "\n") {
		if pred(line) {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

// isExecutable mirrors the shell's `[ -x file ]`: on POSIX it requires the exec
// bit; on Windows (no exec bit) existence is the Git Bash equivalent.
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}
