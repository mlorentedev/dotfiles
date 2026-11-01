package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CheckDrift is the offline drift gate: the Go form of compile-harness.sh's
// do_check and check_coverage (CLI-026). It reads only the checkout: the
// manifest, the enforced records, the targets' GENERATED regions and the skill
// and agent records with their schemas. What a machine has deployed is
// `dotf doctor`'s question, not this one.

// CheckLine is one line of the report. A problem goes to stderr and fails the
// check; Detail holds the lines that explain it, such as the region diff.
type CheckLine struct {
	Problem bool
	Text    string
	Detail  []string
}

// checkManifest is the part of the manifest the check reads.
type checkManifest struct {
	VaultSubpath string `json:"vault_subpath"`
	Enforced     []struct {
		ID     string            `json:"id"`
		Source string            `json:"source"`
		OptOut map[string]string `json:"opt_out"`
	} `json:"enforced"`
	Targets []struct {
		File   string   `json:"file"`
		Inject []string `json:"inject"`
	} `json:"targets"`
	Doctrine *struct {
		Inject []string `json:"inject"`
	} `json:"doctrine"`
	Skills *recordKind `json:"skills"`
	Agents *recordKind `json:"agents"`
}

// recordKind is a manifest's skills or agents block: where the vault keeps the
// sources, where the checkout keeps the records, and the schema they meet.
type recordKind struct {
	VaultSubpath string `json:"vault_subpath"`
	RecordDir    string `json:"record_dir"`
	Schema       string `json:"schema"`
}

func loadCheckManifest(repoRoot string) (*checkManifest, error) {
	raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(ManifestFile))) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		return nil, fmt.Errorf("manifest not found: %w", err)
	}
	var m checkManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ManifestFile, err)
	}
	return &m, nil
}

// doctrineSurface is the coverage surface name of doctrine.inject.
const doctrineSurface = "doctrine"

// CheckDrift runs every check and returns the report in the shell's order:
// coverage, then each target region, then the skill and agent records. Drift is
// any line with Problem set; an error means the manifest could not be read.
func CheckDrift(repoRoot string) ([]CheckLine, error) {
	m, err := loadCheckManifest(repoRoot)
	if err != nil {
		return nil, err
	}
	out := checkCoverage(m)
	for _, t := range m.Targets {
		out = append(out, checkTarget(repoRoot, t.File, t.Inject))
	}
	if m.Skills != nil {
		out = append(out, checkRecords(repoRoot, "SKILL.md", m.Skills.RecordDir,
			orDefault(m.Skills.Schema, "harness/skill-frontmatter.schema.json"), "skill", "record")...)
	}
	if m.Agents != nil {
		out = append(out, checkRecords(repoRoot, "AGENT.md", m.Agents.RecordDir,
			orDefault(m.Agents.Schema, "harness/agent-frontmatter.schema.json"), "agent", "agent")...)
	}
	return out, nil
}

// HasDrift reports whether any line of a report is a problem.
func HasDrift(report []CheckLine) bool {
	for _, l := range report {
		if l.Problem {
			return true
		}
	}
	return false
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// checkCoverage asks, for every enforced id and every surface (each target
// file, and doctrine when the manifest has it), that the id is injected there
// or carries an opt_out reason for it.
func checkCoverage(m *checkManifest) []CheckLine {
	surfaces := map[string][]string{}
	var order []string
	for _, t := range m.Targets {
		if _, seen := surfaces[t.File]; !seen {
			order = append(order, t.File)
		}
		surfaces[t.File] = append(surfaces[t.File], t.Inject...)
	}
	if m.Doctrine != nil {
		order = append(order, doctrineSurface)
		surfaces[doctrineSurface] = m.Doctrine.Inject
	}
	var out []CheckLine
	for _, e := range m.Enforced {
		for _, s := range order {
			if contains(surfaces[s], e.ID) {
				continue
			}
			if reason := e.OptOut[s]; reason != "" {
				out = append(out, CheckLine{Text: fmt.Sprintf("[check] OK -> %s excluded from %s (%s)", e.ID, s, reason)})
				continue
			}
			out = append(out, CheckLine{Problem: true,
				Text:   fmt.Sprintf("[GAP] enforced region %q reaches neither surface %q nor an opt_out for it", e.ID, s),
				Detail: []string{fmt.Sprintf("      -> inject it there, or record enforced[id=%s].opt_out[%q] with the reason", e.ID, s)}})
		}
	}
	return out
}

// checkTarget compares a target's GENERATED region with the enforced records
// it injects.
func checkTarget(repoRoot, file string, ids []string) CheckLine {
	path := filepath.Join(repoRoot, filepath.FromSlash(file))
	raw, err := os.ReadFile(path) //nolint:gosec // a manifest target in the checkout
	if err != nil {
		return CheckLine{Problem: true, Text: fmt.Sprintf("[ERROR] %s: %v", path, err)}
	}
	lines := recordLines(normalizeLF(raw))
	if msg := regionMarkerError(lines); msg != "" {
		return CheckLine{Problem: true, Text: fmt.Sprintf("[ERROR] %s: %s", path, msg)}
	}
	want, err := renderRegion(repoRoot, ids)
	if err != nil {
		return CheckLine{Problem: true, Text: "[ERROR] " + err.Error()}
	}
	got := regionContent(lines)
	if got == want {
		return CheckLine{Text: "[check] OK -> " + file}
	}
	return CheckLine{Problem: true,
		Text:   fmt.Sprintf("[DRIFT] %s: managed region differs from harness/enforced/ (run --refresh)", file),
		Detail: firstDifference(got, want)}
}

// regionMarkerError is the shell's validate_markers: one line starting with the
// BEGIN prefix, one line containing the END marker, BEGIN first.
func regionMarkerError(lines []string) string {
	begins, ends, firstBegin, firstEnd := 0, 0, -1, -1
	for i, l := range lines {
		if strings.HasPrefix(l, GeneratedBeginPrefix) {
			if begins++; firstBegin < 0 {
				firstBegin = i
			}
		}
		if strings.Contains(l, GeneratedEndMarker) {
			if ends++; firstEnd < 0 {
				firstEnd = i
			}
		}
	}
	if begins != 1 || ends != 1 {
		return fmt.Sprintf("need exactly 1 BEGIN + 1 END HARNESS marker (found %d/%d)", begins, ends)
	}
	if firstBegin >= firstEnd {
		return "END marker precedes BEGIN marker"
	}
	return ""
}

// regionContent is the text strictly between the markers. The END line must be
// the marker alone: one with a tail does not close the region, which then runs
// to the end of the file and reads as drift.
func regionContent(lines []string) string {
	var b strings.Builder
	in := false
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, GeneratedBeginPrefix):
			in = true
		case l == GeneratedEndMarker:
			in = false
		case in:
			b.WriteString(l + "\n")
		}
	}
	return b.String()
}

// renderRegion is the full-surface render: the records concatenated in order,
// with trailing newlines collapsed to one and every full-only marker line
// removed. The markers are machinery for the compact doctrine, not rules.
func renderRegion(repoRoot string, ids []string) (string, error) {
	var b strings.Builder
	for _, id := range ids {
		raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(EnforcedDir), id+".md")) //nolint:gosec // a record the manifest names
		if err != nil {
			return "", fmt.Errorf("missing source-of-record: harness/enforced/%s.md (run --refresh)", id)
		}
		b.Write(normalizeLF(raw))
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

// firstDifference names the first line where the committed region (got) and
// the render (want) part, which is what a reader needs to find the drift.
func firstDifference(got, want string) []string {
	g, w := recordLines([]byte(got)), recordLines([]byte(want))
	for i := 0; i < len(g) || i < len(w); i++ {
		gl, wl := lineAt(g, i), lineAt(w, i)
		if gl != wl {
			return []string{
				fmt.Sprintf("    region line %d differs from the render:", i+1),
				"    - " + gl,
				"    + " + wl,
			}
		}
	}
	return nil
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "<end of region>"
}

// checkRecords validates each record's frontmatter against the schema's
// required keys. The Go renders are total, so a record that validates renders;
// the shell's separate render step had nothing left to catch.
func checkRecords(repoRoot, file, recordDir, schema, noun, okNoun string) []CheckLine {
	recDir := filepath.Join(repoRoot, filepath.FromSlash(recordDir))
	names, err := recordNamesOf(recDir, file)
	if err != nil {
		return []CheckLine{{Problem: true, Text: fmt.Sprintf("[DRIFT] no %s records at %s (run --refresh)", noun, recDir)}}
	}
	schemaPath := filepath.Join(repoRoot, filepath.FromSlash(schema))
	required, err := schemaRequired(schemaPath)
	if err != nil {
		return []CheckLine{{Problem: true, Text: fmt.Sprintf("[ERROR] %s schema %s: %v", noun, schemaPath, err)}}
	}
	var out []CheckLine
	for _, name := range names {
		p := filepath.Join(recDir, name, file)
		if msg := frontmatterError(p, required); msg != "" {
			out = append(out, CheckLine{Problem: true, Text: "[ERROR] " + msg})
			continue
		}
		out = append(out, CheckLine{Text: fmt.Sprintf("[check] OK -> %s %s", okNoun, name)})
	}
	return out
}

func schemaRequired(path string) ([]string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // a schema the manifest names
	if err != nil {
		return nil, err
	}
	var s struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("not valid JSON: %w", err)
	}
	return s.Required, nil
}

// frontmatterError is the shell's validate_skill_frontmatter: `---` on line 1,
// a closing fence, and each required key present with a non-empty value on
// its own line. Values are not type-checked, as in the shell.
func frontmatterError(path string, required []string) string {
	raw, err := os.ReadFile(path) //nolint:gosec // a record under the manifest's record_dir
	if err != nil {
		return fmt.Sprintf("%s: %v", path, err)
	}
	lines := recordLines(normalizeLF(raw))
	if len(lines) == 0 || lines[0] != "---" {
		return path + ":1: missing YAML frontmatter (expected --- on line 1)"
	}
	closed := false
	for _, l := range lines[1:] {
		if fenceLine.MatchString(l) {
			closed = true
			break
		}
	}
	if !closed {
		return path + ":1: unterminated frontmatter (no closing ---)"
	}
	for _, key := range required {
		if !frontmatterHas(raw, regexp.MustCompile("^"+regexp.QuoteMeta(key)+`:[[:space:]]*[^[:space:]]`)) {
			return fmt.Sprintf("%s: required frontmatter key %q missing or empty", path, key)
		}
	}
	return ""
}
