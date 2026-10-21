package harness

import (
	"os"
	"path/filepath"
	"strings"
)

// The copilot skill catalog: copilot has no per-skill mechanism, so every
// record that targets it is listed, one bullet each, in the GENERATED region of
// its $HOME instructions file. This is the Go form of compile-harness.sh's
// build_skill_catalog and the catalog step of deploy_skills (CLI-026).

// deployCatalog rewrites the catalog region of the manifest's catalog file. A
// file that is absent, or does not carry exactly one GENERATED region, is
// skipped and left untouched: `dotf harness instructions` writes the file and
// its slot, and the catalog never creates one.
func deployCatalog(m *SkillManifest, recDir string, names []string, home string, dryRun bool) (*SkillOutcome, error) {
	if m.Catalog == nil {
		return nil, nil
	}
	file := filepath.Join(home, filepath.FromSlash(m.Catalog.File))
	skipped := &SkillOutcome{Agent: m.Catalog.Agent, Path: file, Status: SkillSkipped, Catalog: true, Reason: "catalog target absent or unmarked"}
	current, err := os.ReadFile(file) //nolint:gosec // the manifest's catalog file under home
	if err != nil {
		return skipped, nil
	}
	bullets := catalogBullets(recDir, names, m.Catalog.Agent)
	begin := GeneratedBeginPrefix + " (sha256:" + PresenceSHA(bullets) + ") — skill catalog from vault " +
		m.VaultSubpath + "; edit there + re-run setup, do NOT edit between markers -->"
	want, ok := replaceCatalogRegion(string(current), begin, bullets)
	if !ok {
		return skipped, nil
	}
	wrote, err := writeIfChanged(file, []byte(want), 0o644, dryRun)
	if err != nil {
		return nil, err
	}
	status := SkillCurrent
	if wrote {
		status = SkillDeployed
	}
	return &SkillOutcome{Agent: m.Catalog.Agent, Path: file, Status: status, Catalog: true}, nil
}

// catalogBullets lists, in names' order, each record that targets agent as
// `- **name** — description`. A record that cannot be read or parsed is left
// out here; the per-target deploy already reports it.
func catalogBullets(recDir string, names []string, agent string) string {
	var b strings.Builder
	for _, name := range names {
		record, err := os.ReadFile(filepath.Join(recDir, name, "SKILL.md")) //nolint:gosec // a record under the manifest's record_dir
		if err != nil {
			continue
		}
		if targeted, err := SkillTargetsAgent(record, agent); err != nil || !targeted {
			continue
		}
		b.WriteString("- **" + name + "** — " + skillField(record, "description") + "\n")
	}
	return b.String()
}

// skillField is the raw value of key on its frontmatter line, quotes and
// trailing space kept, as the shell's skill_field read it: the first match
// wins, and a block scalar yields only its indicator.
func skillField(record []byte, key string) string {
	fences := 0
	for _, l := range recordLines(normalizeLF(record)) {
		if fenceLine.MatchString(l) {
			fences++
			if fences >= 2 {
				return ""
			}
			continue
		}
		if fences == 1 && strings.HasPrefix(l, key+":") {
			return strings.TrimLeft(strings.TrimPrefix(l, key+":"), " \t\v\f\r")
		}
	}
	return ""
}

// replaceCatalogRegion swaps the one GENERATED region of file for begin, the
// content and the END marker, ending every line with a newline. It refuses,
// as the shell's validate_markers did, a file without exactly one BEGIN line
// before exactly one END line. The END line must be exactly the marker: the
// shell counted any line containing it, then skipped to the end of the file
// when none matched exactly.
func replaceCatalogRegion(file, begin, content string) (string, bool) {
	lines := recordLines(normalizeLF([]byte(file)))
	bi, ei := -1, -1
	for i, l := range lines {
		switch {
		case strings.HasPrefix(l, GeneratedBeginPrefix):
			if bi >= 0 {
				return "", false
			}
			bi = i
		case strings.Contains(l, GeneratedEndMarker):
			if ei >= 0 || l != GeneratedEndMarker {
				return "", false
			}
			ei = i
		}
	}
	if bi < 0 || ei < bi {
		return "", false
	}
	var b strings.Builder
	for _, l := range lines[:bi] {
		b.WriteString(l + "\n")
	}
	b.WriteString(begin + "\n" + content)
	for _, l := range lines[ei:] {
		b.WriteString(l + "\n")
	}
	return b.String(), true
}
