package spec

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// agentTagPattern matches the editorial markers a spec must not carry into the
// archive, in BOTH shapes the tooling produces: the bare `[AGENT-DRAFT]` and the
// suffixed `[AGENT-DRAFT — review before archive]` that /spec fill actually
// writes (harness/skills/spec/SKILL.md, "Capture rules").
//
// The original pattern required `]` immediately after the keyword, mirroring
// archive-spec.sh's grep. That never matched the canonical emitted form, so the
// archive lock did not lock: specs/archive/CLI-002-repo-structure/proposal.md:19
// was archived still carrying a live `[AGENT-SUGGESTION — accept or remove]`.
// Inverted twice over, in fact — the only shape it did match, the bare one, is
// the shape this repo writes when documenting the markers rather than using them.
var agentTagPattern = regexp.MustCompile(`\[AGENT-(DRAFT|SUGGESTION)\b[^\]]*\]`)

// fenceLinePattern matches a fenced-code delimiter, capturing the run of fence
// characters so a closing fence can be required to be at least as long as the
// opening one (CommonMark), and of the same character.
var fenceLinePattern = regexp.MustCompile("^\\s{0,3}(`{3,}|~{3,})")

// completedTaskPattern matches a ticked checklist item. A tag on such a line is
// by definition resolved — the line records work already done.
var completedTaskPattern = regexp.MustCompile(`^\s*[-*+]\s+\[[xX]\]`)

// codeSpanPattern matches an inline code span, so its contents can be dropped
// before scanning. Backtick runs are matched symmetrically enough for prose.
var codeSpanPattern = regexp.MustCompile("`+[^`]*`+")

// ScanUnresolvedTags returns the 1-based numbers of lines in content that carry
// an UNRESOLVED agent tag. A tag that is quoted rather than live is not a hit:
//
//   - inside a fenced code block or an inline code span — documentation ABOUT the
//     markers, which this repo writes constantly because it builds the tooling
//     that emits them;
//   - on a completed checklist line (`- [x]`), which records finished work.
//
// The exclusions are shape-based rather than a real markdown parse on purpose: a
// spec is not arbitrary markdown, and a parser would be a far heavier dependency
// than the two shapes that actually produce false positives here. Both shapes are
// pinned by tests in the red direction — a genuine tag must still be found.
func ScanUnresolvedTags(content string) []int {
	var (
		hits      []int
		fenceMark string
	)
	for i, line := range strings.Split(content, "\n") {
		if m := fenceLinePattern.FindStringSubmatch(line); m != nil {
			switch {
			case fenceMark == "":
				fenceMark = m[1]
			case m[1][0] == fenceMark[0] && len(m[1]) >= len(fenceMark):
				fenceMark = ""
			}
			continue
		}
		if fenceMark != "" || completedTaskPattern.MatchString(line) {
			continue
		}
		if agentTagPattern.MatchString(codeSpanPattern.ReplaceAllString(line, "")) {
			hits = append(hits, i+1)
		}
	}
	return hits
}

// statusLinePattern matches a `status:` line, capturing the `status: ` prefix
// (group 1) separately from the value token so only the value is rewritten.
// Anything after the value (a trailing comment) is preserved — protecting
// meaningful comments like `status: abandoned # superseded by ADR-020` (DX-002),
// which the awk in archive-spec.sh would have destroyed.
var statusLinePattern = regexp.MustCompile(`^(status:\s+)\S+(.*)$`)

// ArchiveOptions configures Archive.
type ArchiveOptions struct {
	Abandoned       bool   // route to specs/archive/_abandoned/<id>; status -> abandoned
	ForceWithDrafts bool   // archive even with unresolved [AGENT-*] tags
	PRURL           string // when set, append an archived/PR provenance comment
	Date            string // YYYY-MM-DD for the PR comment (caller-supplied; deterministic in tests)
}

// FindUnresolvedTags walks specDir and returns "relpath:line: text" for every
// line carrying an [AGENT-DRAFT] or [AGENT-SUGGESTION] marker, in walk order.
// An empty slice means the spec is clean.
func FindUnresolvedTags(specDir string) ([]string, error) {
	var hits []string
	err := filepath.WalkDir(specDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(specDir, path)
		if relErr != nil {
			rel = path
		}
		lines := strings.Split(string(data), "\n")
		for _, n := range ScanUnresolvedTags(string(data)) {
			hits = append(hits, fmt.Sprintf("%s:%d: %s", rel, n, strings.TrimSpace(lines[n-1])))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning %s for unresolved tags: %w", specDir, err)
	}
	return hits, nil
}

// setStatus rewrites the value of the `status:` line inside the FIRST YAML
// frontmatter block (between the first two `---` fences) to newStatus. It is the
// union of archive-spec.sh's awk scoping (first block only) and
// archive-spec.ps1's value-only replacement (trailing comment preserved): a
// `status:` token in the body, or in a second fenced block, is left untouched.
// Content without frontmatter is returned unchanged.
func setStatus(content, newStatus string) string {
	lines := strings.Split(content, "\n")
	fences := 0
	for i, line := range lines {
		if line == "---" {
			fences++
			if fences == 2 {
				break // end of the first frontmatter block
			}
			continue
		}
		if fences == 1 {
			if m := statusLinePattern.FindStringSubmatch(line); m != nil {
				lines[i] = m[1] + newStatus + m[2]
				break // only the first status: line in the block
			}
		}
	}
	return strings.Join(lines, "\n")
}

// Archive performs the mechanical spec archive, the Go twin of archive-spec.sh:
// id validation, a tag pre-flight, a no-clobber move into the archive (or
// _abandoned) tree, a proposal status rewrite, and an optional PR provenance
// comment. It returns the absolute target directory on success. Vault promotion
// and backlog ticks are out of scope — do those via "/spec archive" in an agent.
//
// id is validated first: ValidateID's grammar admits no path separators or "..",
// so it doubles as the guard that keeps a crafted id (e.g. "../../etc") from
// escaping specs/ through the filepath.Join calls below.
func Archive(repoRoot, id string, opts ArchiveOptions) (target string, err error) {
	if err := ValidateID(id); err != nil {
		return "", err
	}
	specDir := filepath.Join(repoRoot, "specs", id)
	if info, statErr := os.Stat(specDir); statErr != nil || !info.IsDir() {
		return "", fmt.Errorf("spec not found: %s", specDir)
	}

	if !opts.ForceWithDrafts {
		tags, tagErr := FindUnresolvedTags(specDir)
		if tagErr != nil {
			return "", tagErr
		}
		if len(tags) > 0 {
			return "", fmt.Errorf("unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags found:\n  %s\n"+
				"resolve them (accept/edit/delete) before archiving, or use --force-with-drafts",
				strings.Join(tags, "\n  "))
		}
	}

	newStatus := "archived"
	target = filepath.Join(repoRoot, "specs", "archive", id)
	if opts.Abandoned {
		newStatus = "abandoned"
		target = filepath.Join(repoRoot, "specs", "archive", "_abandoned", id)
	}

	if _, statErr := os.Stat(target); statErr == nil {
		return "", fmt.Errorf("already in archive: %s", target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(specDir, target); err != nil {
		return "", fmt.Errorf("moving %s -> %s: %w", specDir, target, err)
	}

	// Update the moved proposal.md (best-effort: a spec missing proposal.md
	// still archives, matching the shell's `if -f` guard).
	proposal := filepath.Join(target, "proposal.md")
	if data, readErr := os.ReadFile(proposal); readErr == nil {
		out := setStatus(string(data), newStatus)
		if opts.PRURL != "" {
			out += fmt.Sprintf("\n<!-- archived %s — PR: %s -->\n", opts.Date, opts.PRURL)
		}
		if err := os.WriteFile(proposal, []byte(out), 0o644); err != nil {
			return target, fmt.Errorf("updating %s: %w", proposal, err)
		}
	}

	return target, nil
}
