package spec

import (
	"errors"
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

// stripCodeSpans removes inline code spans from one line, honouring CommonMark's
// rule that the opening and closing backtick runs must be the SAME length.
//
// A regexp cannot express that: Go's RE2 has no backreferences, and the obvious
// "`+[^`]*`+" accepts unequal runs. That form strips `[AGENT-DRAFT]“, which is
// NOT a code span — so a live marker written with mismatched backticks would be
// silently dropped and the spec would archive carrying it. For a guard, that
// false negative is the dangerous direction, and it is the same class of defect
// this whole change exists to remove.
//
// A run with no partner of equal length is left in place, so an unbalanced
// backtick makes the scanner REPORT rather than skip. Code spans that cross a
// line boundary are not tracked, for the same reason: both choices err toward
// refusing an archive, never toward passing one.
func stripCodeSpans(line string) string {
	var out strings.Builder
	for i := 0; i < len(line); {
		if line[i] != '`' {
			out.WriteByte(line[i])
			i++
			continue
		}
		open := i
		for i < len(line) && line[i] == '`' {
			i++
		}
		runLen := i - open

		end := -1
		for j := i; j < len(line); {
			if line[j] != '`' {
				j++
				continue
			}
			k := j
			for k < len(line) && line[k] == '`' {
				k++
			}
			if k-j == runLen {
				end = k
				break
			}
			j = k
		}
		if end == -1 {
			out.WriteString(line[open:i]) // no equal-length partner: not a span
			continue
		}
		i = end // drop the span, delimiters included
	}
	return out.String()
}

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
		if agentTagPattern.MatchString(stripCodeSpans(line)) {
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
	Abandoned          bool // route to specs/archive/_abandoned/<id>; status -> abandoned
	ForceWithDrafts    bool // archive even with unresolved [AGENT-*] tags
	ForceWithoutReview bool // archive even without a passing, fresh review.md
	// BypassReason is required whenever either Force flag is set, and it is
	// recorded — with what the skipped check would have refused — as a
	// `review_bypass:` line in the archived proposal.md (SDD-042).
	BypassReason string
	PRURL        string // when set, append an archived/PR provenance comment
	Date         string // YYYY-MM-DD for the PR comment (caller-supplied; deterministic in tests)

	// Staleness overrides how a review's freshness is decided. nil uses the
	// repository's git history; tests inject a fake to avoid building one.
	Staleness StalenessChecker

	// VaultRoot resolves the knowledge vault, where a promoted pattern lives.
	// It is called only when a promotion answer names a 00_meta/ path; nil
	// refuses such a path rather than passing it unchecked.
	VaultRoot func() (string, error)
}

// ReviewStateFiles are the files the REVIEW machinery writes into a spec
// folder, as opposed to artifacts the spec's author wrote. It is the one
// declared list of review state: the draft-tag scan skips these (below), and a
// byte-bound review (#1153, epic #1625 W3.6) must exclude them from the tree it
// hashes. review-request.json in particular is written at launch, after
// reviewed_sha is fixed, so hashing it would make every review stale on arrival.
var ReviewStateFiles = []string{
	ReviewFile, TranscriptFile, StderrPath(TranscriptFile), ReviewRequestFile,
	SecondSigner.Review, SecondSigner.Transcript, StderrPath(SecondSigner.Transcript), SecondSigner.Request,
}

// IsReviewState reports whether name is one of ReviewStateFiles.
//
// The tag scan must skip these, because scanning them makes the gate unpassable
// by construction (#998). The adversarial-review skill instructs the reviewer
// to check the spec for these very markers, so:
//
//   - review.md carries the literal in its own verdict table — a review that
//     certifies the spec is CLEAN is what makes the scan call it dirty.
//     Observed on HARNESS-072: one hit, on the row reading "No [AGENT-DRAFT]
//     tags | OK".
//   - the transcript, and its .stderr sibling, carry it throughout the
//     reviewer's raw event stream: 3593 hits in HARNESS-072's, inside 552 MB
//     that this function would otherwise also read into memory whole.
//
// So the gate demanded a review, and the review's own output then failed the
// gate. Every spec reviewed by this tool refused to archive, and the only exit
// was --force-with-drafts, which disables the real check too.
//
// Matched by exact name, never by shape or extension: everything else in the
// folder is still scanned, so an artifact added later (design.md, notes.md) is
// guarded with no code change. That direction is deliberate and matches
// ScanUnresolvedTags above — a deny-list errs toward refusing an archive, an
// allow-list would silently stop guarding a file someone adds.
func IsReviewState(name string) bool {
	for _, f := range ReviewStateFiles {
		if name == f {
			return true
		}
	}
	return false
}

// FindUnresolvedTags walks specDir and returns "relpath:line: text" for every
// line carrying an [AGENT-DRAFT] or [AGENT-SUGGESTION] marker, in walk order.
// An empty slice means the spec is clean. Files written by the review machinery
// are skipped — see IsReviewState.
func FindUnresolvedTags(specDir string) ([]string, error) {
	var hits []string
	err := filepath.WalkDir(specDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if IsReviewState(d.Name()) {
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
	if i := frontmatterStatusLine(lines); i >= 0 {
		m := statusLinePattern.FindStringSubmatch(lines[i])
		lines[i] = m[1] + newStatus + m[2]
	}
	return strings.Join(lines, "\n")
}

// frontmatterStatusLine returns the index of the first `status:` line inside the
// frontmatter block, which opens on the first line, or -1. Fences are compared
// trimmed, as in withFrontmatterField: a CRLF checkout's fence is "---\r", and
// an exact match never saw the block, so the archive reported a rewrite it had
// not made (CLI-095, #1976).
func frontmatterStatusLine(lines []string) int {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return -1 // a `---` further down is a thematic break, not frontmatter
	}
	for i := 1; i < len(lines) && strings.TrimSpace(lines[i]) != "---"; i++ {
		if statusLinePattern.MatchString(lines[i]) {
			return i
		}
	}
	return -1
}

// withStatus is setStatus that cannot report a rewrite it did not make: a
// proposal whose first frontmatter block has no status: line gets one, and one
// with no frontmatter at all (a pre-template proposal) gets a block, so the
// status the archive prints is always the status the file carries (CLI-095).
func withStatus(content, newStatus string) string {
	if frontmatterStatusLine(strings.Split(content, "\n")) >= 0 {
		return setStatus(content, newStatus)
	}
	return withFrontmatterLine(content, "status: "+newStatus)
}

// Archive performs the mechanical spec archive, the Go twin of archive-spec.sh:
// id validation, a tag pre-flight, a no-clobber move into the archive (or
// _abandoned) tree, a proposal status rewrite, and an optional PR provenance
// comment, and the archive checklist items it performed ticked in
// verification.md. It returns the absolute target directory on success. Promotions are
// checked, not performed: each candidate in verification.md must be answered,
// and each "yes" must name a file that exists (CheckPromotions). Writing the
// lesson, ADR or pattern, and the backlog tick, happen before the archive.
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

	bypass, err := checkBypassRequest(specDir, opts)
	if err != nil {
		return "", err
	}
	overrode, err := runPreflights(repoRoot, id, specDir, opts)
	if err != nil {
		return "", err
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
	// verification.md is read before anything moves: a missing one is a no-op,
	// but one that exists and cannot be read would leave the checklist
	// contradicting the tree, so that refuses here, while nothing has changed.
	checklist, readErr := os.ReadFile(filepath.Join(specDir, "verification.md"))
	if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
		return "", fmt.Errorf("reading verification.md: %w", readErr)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(specDir, target); err != nil {
		return "", fmt.Errorf("moving %s -> %s: %w", specDir, target, err)
	}

	// Update the moved proposal.md (best-effort: a spec missing proposal.md
	// still archives, matching the shell's `if -f` guard).
	// For Fast-Track Development, update spec.md instead.
	proposal := filepath.Join(target, "proposal.md")
	if _, err := os.Stat(filepath.Join(target, "spec.md")); err == nil {
		proposal = filepath.Join(target, "spec.md")
	}
	if data, readErr := os.ReadFile(proposal); readErr == nil {
		out := withStatus(string(data), newStatus)
		if bypass {
			out = withFrontmatterField(out, "review_bypass", bypassRecord(opts, overrode))
		}
		if opts.PRURL != "" {
			eol := lineEnding(out)
			out += fmt.Sprintf("%s<!-- archived %s — PR: %s -->%s", eol, opts.Date, opts.PRURL, eol)
		}
		if err := os.WriteFile(proposal, []byte(out), 0o644); err != nil {
			return target, fmt.Errorf("updating %s: %w", filepath.Base(proposal), err)
		}
	}

	if readErr == nil {
		if out := tickArchiveChecklist(string(checklist), opts.Abandoned); out != string(checklist) {
			if err := os.WriteFile(filepath.Join(target, "verification.md"), []byte(out), 0o644); err != nil {
				// The move already happened, so a retry would refuse as "already in
				// archive": say what is done and what is left.
				return target, fmt.Errorf("archived to %s, but ticking verification.md's archive checklist failed: %w; tick it by hand", target, err)
			}
		}
	}

	return target, nil
}

// archiveChecklistItems are the `## Archive checklist` items Archive itself
// performs or verifies, matched case-insensitively against an unticked item's
// text. The record otherwise said the archive never happened: 177 of 209
// archived specs left the list unticked, because nothing ticked it (#1990).
// The fragments cover the wordings in the corpus, not only the template's.
//
// The board item stays for the human: the command cannot see the board.
var archiveChecklistItems = []struct {
	fragment     string
	notAbandoned bool // the item describes the archive route, not _abandoned
}{
	{"status: archived", true}, // the status rewrite above
	{"folder moved", true},     // the rename into specs/archive/<id>
	// The promotion pre-flight ran and passed, or nothing moved: each "yes"
	// names a file that exists, and no flag skips that check.
	{"promotions above executed", false},
}

// uncheckedItemPattern captures an unticked list item's prefix up to the box.
var uncheckedItemPattern = regexp.MustCompile(`^(\s*[-*+]\s+\[) (\].*)$`)

// tickArchiveChecklist ticks, inside content's `## Archive checklist` section
// only, the unticked items archiveChecklistItems names. Everything else is
// returned byte for byte, CRLF included.
func tickArchiveChecklist(content string, abandoned bool) string {
	lines := strings.Split(content, "\n")
	eachUncheckedArchiveItem(lines, func(i int, m []string) {
		text := strings.ToLower(m[2])
		for _, item := range archiveChecklistItems {
			if item.notAbandoned && abandoned {
				continue
			}
			if strings.Contains(text, item.fragment) {
				lines[i] = m[1] + "x" + m[2]
				return
			}
		}
	})
	return strings.Join(lines, "\n")
}

// UntickedArchiveChecklist returns the text of every item still unticked in
// content's `## Archive checklist` section: what the archive left for a human
// (the board ticket, which the command cannot see).
func UntickedArchiveChecklist(content string) []string {
	var left []string
	eachUncheckedArchiveItem(strings.Split(content, "\n"), func(_ int, m []string) {
		left = append(left, strings.TrimSpace(strings.TrimPrefix(m[2], "]")))
	})
	return left
}

// eachUncheckedArchiveItem calls fn with the index and match of every unticked
// item inside the `## Archive checklist` section of lines.
func eachUncheckedArchiveItem(lines []string, fn func(i int, m []string)) {
	inSection := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			inSection = strings.EqualFold(trimmed, "## Archive checklist")
			continue
		}
		if m := uncheckedItemPattern.FindStringSubmatch(line); inSection && m != nil {
			fn(i, m)
		}
	}
}

// checkBypassRequest validates a bypass before anything moves: a Force flag
// needs a stated reason, and a proposal.md to record it in. It reports whether
// this archive is a bypass.
func checkBypassRequest(specDir string, opts ArchiveOptions) (bool, error) {
	if !opts.ForceWithDrafts && !opts.ForceWithoutReview {
		return false, nil
	}
	if strings.TrimSpace(opts.BypassReason) == "" {
		return true, errors.New(`--force-with-drafts and --force-without-review require --reason "<why>": ` +
			"a bypass is recorded in the archived proposal.md as review_bypass:, and the reason is what makes it auditable")
	}
	proposal := "proposal.md"
	if _, err := os.Stat(filepath.Join(specDir, "spec.md")); err == nil {
		proposal = "spec.md"
	}
	if _, err := os.Stat(filepath.Join(specDir, proposal)); err != nil {
		return true, fmt.Errorf("a bypass is recorded in %s, and this spec has none: add one, or satisfy the checks instead", proposal)
	}
	return true, nil
}

// runPreflights runs the pre-flights — the tag scan ("is the spec finished
// being written?"), the review gate (CLI-034: "did anyone independently argue
// against it?") and the promotion answers (HARNESS-160: "did the knowledge
// land?") — and returns the refusal of each one a Force flag overrode. No flag
// overrides the promotion check: the way past it is answering the line. A bypassed check is still RUN, so the record says what was
// overridden rather than which flag was typed: --force-without-review used to
// be recorded, if at all, as "without review" when what it skipped was a
// perfectly good review's freshness (#998).
func runPreflights(repoRoot, id, specDir string, opts ArchiveOptions) ([]string, error) {
	var overrode []string
	tags, err := FindUnresolvedTags(specDir)
	if err != nil {
		return nil, err
	}
	if len(tags) > 0 {
		if !opts.ForceWithDrafts {
			return nil, fmt.Errorf("unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags found:\n  %s\n"+
				"resolve them (accept, edit or delete each one) before archiving",
				strings.Join(tags, "\n  "))
		}
		overrode = append(overrode, fmt.Sprintf("%d unresolved draft tag(s)", len(tags)))
	}

	isFastTrack := false
	if _, err := os.Stat(filepath.Join(specDir, "spec.md")); err == nil {
		isFastTrack = true
	}

	if !isFastTrack {
		if gateErr := checkReviewGate(repoRoot, id, specDir, opts.Staleness); gateErr != nil {
			if !opts.ForceWithoutReview {
				return nil, gateErr
			}
			headline, _, _ := strings.Cut(gateErr.Error(), "\n")
			overrode = append(overrode, headline)
		}
	}

	if problems := CheckPromotions(repoRoot, specDir, opts.VaultRoot); len(problems) > 0 {
		docName := "verification.md"
		if isFastTrack {
			docName = "spec.md"
		}
		return nil, fmt.Errorf("promotion candidates in %s are not all answered:\n  %s\n"+
			`answer each as "yes: <path of the promoted file>" or "no: <reason>"; no flag skips this check`,
			docName, strings.Join(problems, "\n  "))
	}
	return overrode, nil
}

// bypassRecord is the one-line value of `review_bypass:`.
func bypassRecord(opts ArchiveOptions, overrode []string) string {
	var flags []string
	if opts.ForceWithDrafts {
		flags = append(flags, "force-with-drafts")
	}
	if opts.ForceWithoutReview {
		flags = append(flags, "force-without-review")
	}
	what := "nothing (the checks would have passed)"
	if len(overrode) > 0 {
		what = strings.Join(overrode, " | ")
	}
	return fmt.Sprintf("%s; overrode: %s; reason: %s; date: %s",
		strings.Join(flags, "+"), what, oneLine(opts.BypassReason), opts.Date)
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// withFrontmatterField adds `key: "value"` as the last line of content's
// frontmatter block, creating the block when the file has none. The value is
// double-quoted with `\` and `"` escaped, so frontmatterFields reads it back
// verbatim, `#` included.
func withFrontmatterField(content, key, value string) string {
	quoted := `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
	return withFrontmatterLine(content, key+": "+quoted)
}

// withFrontmatterLine appends line as the last line of content's frontmatter
// block, creating the block when the file has none. A CRLF block gets a CRLF
// line, so the insertion does not leave mixed line endings behind.
func withFrontmatterLine(content, line string) string {
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		if strings.HasSuffix(lines[0], "\r") {
			line += "\r"
		}
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				return strings.Join(append(lines[:i:i], append([]string{line}, lines[i:]...)...), "\n")
			}
		}
	}
	eol := lineEnding(content)
	return "---" + eol + line + eol + "---" + eol + content
}

// lineEnding is the line ending content already uses, so text added to a CRLF
// file is CRLF too.
func lineEnding(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
}
