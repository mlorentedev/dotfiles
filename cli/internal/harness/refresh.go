package harness

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Refresh is the Go form of compile-harness.sh --refresh (CLI-026 slice 7): it
// re-renders everything the checkout commits from the vault. Each enforced
// pattern section becomes its record under harness/enforced/, each target's
// GENERATED region is replaced with those records under a fresh sha, and each
// vault skill and agent directory is copied whole into its record directory
// with generated_* provenance stamped on SKILL.md or AGENT.md.
//
// Everything is read and checked before anything is written: a dangling
// trigger, a missing section, a broken marker pair, a record that fails its
// schema or a target over its line cap leaves the checkout as it was. The
// shell checked as it went and could stop halfway.
//
// Whether the vault is current enough to refresh from is the caller's question
// (`dotf harness refresh`, #2162); this reads whatever vault it is given.
func Refresh(repoRoot, vault string) ([]string, error) {
	m, err := loadCheckManifest(repoRoot)
	if err != nil {
		return nil, err
	}
	patDir := filepath.Join(vault, filepath.FromSlash(m.VaultSubpath))
	if !isDir(patDir) {
		return nil, fmt.Errorf("refresh needs the vault: patterns dir not found: %s", patDir)
	}
	if err := checkTriggerPatterns(repoRoot, patDir); err != nil {
		return nil, err
	}

	var w refreshWrites
	records := map[string]string{}
	for _, e := range m.Enforced {
		pat, slug, _ := strings.Cut(e.Source, "#")
		body, err := sectionOf(filepath.Join(patDir, filepath.FromSlash(pat)), slug)
		if err != nil {
			return nil, err
		}
		records[e.ID] = body
		w.file(filepath.Join(repoRoot, filepath.FromSlash(EnforcedDir), e.ID+".md"), body,
			fmt.Sprintf("record: %s/%s.md  <- %s#%s", EnforcedDir, e.ID, pat, slug))
	}
	for _, t := range m.Targets {
		content, err := refreshedTarget(repoRoot, t.File, t.Inject, records, m.VaultSubpath)
		if err != nil {
			return nil, err
		}
		w.file(filepath.Join(repoRoot, filepath.FromSlash(t.File)), content, "injected -> "+t.File)
	}
	for _, k := range []struct {
		rec  *recordKind
		file string
	}{{m.Skills, "SKILL.md"}, {m.Agents, "AGENT.md"}} {
		if k.rec == nil {
			continue
		}
		if err := w.records(repoRoot, vault, k.rec, k.file); err != nil {
			return nil, err
		}
	}
	return w.apply()
}

// RefreshPaths are the checkout paths Refresh writes: the harness directory
// and every target file. The caller guards them for uncommitted work and
// reports what the refresh changed in them.
func RefreshPaths(repoRoot string) ([]string, error) {
	m, err := loadCheckManifest(repoRoot)
	if err != nil {
		return nil, err
	}
	paths := []string{"harness/"}
	for _, t := range m.Targets {
		paths = append(paths, t.File)
	}
	return paths, nil
}

// checkTriggerPatterns refuses a refresh while harness/triggers.json names a
// pattern the vault does not have. A trigger whose pattern is gone still fires
// and sends a session looking for it; the refresh is the one step that holds
// both the triggers and the vault.
func checkTriggerPatterns(repoRoot, patDir string) error {
	raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(TriggersFile))) //nolint:gosec // repo-relative, fixed name
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	cfg, err := ParseTriggers(raw)
	if err != nil {
		return fmt.Errorf("%s cannot be read as trigger rules: %w", TriggersFile, err)
	}
	var dangling []string
	named := 0
	for _, t := range cfg.Triggers {
		if t.Pattern == "" {
			continue
		}
		named++
		if _, err := os.Stat(filepath.Join(patDir, t.Pattern+".md")); err != nil {
			dangling = append(dangling, fmt.Sprintf("trigger %q names pattern %q", t.ID, t.Pattern))
		}
	}
	if named == 0 {
		return fmt.Errorf("%s names no pattern, so nothing was checked", TriggersFile)
	}
	if len(dangling) > 0 {
		return fmt.Errorf("%s: %s, which the vault (%s) does not have; point each trigger at a pattern that exists, or remove it",
			TriggersFile, strings.Join(dangling, "; "), patDir)
	}
	return nil
}

var headingLine = regexp.MustCompile(`^#{1,6} `)

// sectionOf returns the body under the heading whose GitHub-style slug is
// slug, up to the next heading of the same or a higher level. A deeper heading
// is part of the body, not its end (ENGINE-002). Trailing newlines collapse to
// one, and a body that is empty after that is an error: the heading was
// renamed, or its rules moved.
func sectionOf(path, slug string) (string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // a vault pattern the manifest names
	if err != nil {
		return "", fmt.Errorf("section %q: %w", slug, err)
	}
	var b strings.Builder
	capLevel := 0
	for _, l := range recordLines(normalizeLF(raw)) {
		if headingLine.MatchString(l) {
			level := len(l) - len(strings.TrimLeft(l, "#"))
			if capLevel > 0 && level <= capLevel {
				break
			}
			if capLevel == 0 && headingSlug(strings.TrimLeft(strings.TrimLeft(l, "#"), " ")) == slug {
				capLevel = level
				continue
			}
		}
		if capLevel > 0 {
			b.WriteString(l + "\n")
		}
	}
	body := strings.TrimRight(b.String(), "\n")
	if body == "" {
		return "", fmt.Errorf("section %q not found (or empty) in %s", slug, path)
	}
	return body + "\n", nil
}

var (
	slugDrop   = regexp.MustCompile(`[^a-z0-9 -]`)
	slugSpaces = regexp.MustCompile(` +`)
	slugDashes = regexp.MustCompile(`-+`)
)

// headingSlug lowercases ASCII only, as awk's tolower does, so a non-ASCII
// letter is dropped rather than folded into one the slug then keeps.
func headingSlug(s string) string {
	s = strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return r
	}, s)
	s = slugDrop.ReplaceAllString(s, "")
	s = slugSpaces.ReplaceAllString(s, "-")
	return slugDashes.ReplaceAllString(s, "-")
}

// targetLineCaps are the deployed line caps of the files that have one.
var targetLineCaps = map[string]int{"ai/claude/CLAUDE.md": 100, "ai/agy/AGY.md": 50}

// refreshedTarget returns a target with its GENERATED region replaced by the
// render of the given records, under a BEGIN line carrying the render's sha.
func refreshedTarget(repoRoot, file string, ids []string, records map[string]string, vaultSubpath string) (string, error) {
	path := filepath.Join(repoRoot, filepath.FromSlash(file))
	raw, err := os.ReadFile(path) //nolint:gosec // a manifest target in the checkout
	if err != nil {
		return "", err
	}
	lines := recordLines(normalizeLF(raw))
	if msg := regionMarkerError(lines); msg != "" {
		return "", fmt.Errorf("%s: %s", path, msg)
	}
	region, err := renderRecords(ids, records)
	if err != nil {
		return "", err
	}
	begin := fmt.Sprintf("%s (sha256:%s) — SSOT: vault %s; edit there + re-run setup, do NOT edit between markers -->",
		GeneratedBeginPrefix, sha16(region), vaultSubpath)
	var b strings.Builder
	n, skip := 0, false
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, GeneratedBeginPrefix):
			b.WriteString(begin + "\n" + region)
			n += 1 + strings.Count(region, "\n")
			skip = true
			continue
		case l == GeneratedEndMarker:
			skip = false
		case skip:
			continue
		}
		b.WriteString(l + "\n")
		n++
	}
	if skip {
		// The END line carries text after the marker: it passed the marker
		// count, which matches by substring, and never closed the region. The
		// shell wrote this out and lost every line after it (#2884).
		return "", fmt.Errorf("%s: the END marker line has text after %s, so the region never closes; fix the line, then refresh", path, GeneratedEndMarker)
	}
	if limit := targetLineCaps[file]; limit > 0 && n > limit {
		return "", fmt.Errorf("%s would be %d lines after injection (cap %d)", file, n, limit)
	}
	return b.String(), nil
}

// renderRecords is renderRegion over records not yet written: the records in
// order, trailing newlines collapsed to one, full-only marker lines removed.
func renderRecords(ids []string, records map[string]string) (string, error) {
	var b strings.Builder
	for _, id := range ids {
		body, ok := records[id]
		if !ok {
			return "", fmt.Errorf("target injects %q, which is not an enforced id", id)
		}
		b.WriteString(body)
	}
	var out strings.Builder
	for _, l := range strings.Split(strings.TrimRight(b.String(), "\n"), "\n") {
		if strings.Contains(l, fullOnlyBegin) || strings.Contains(l, fullOnlyEnd) {
			continue
		}
		out.WriteString(l + "\n")
	}
	return out.String(), nil
}
