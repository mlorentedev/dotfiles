package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// The compact doctrine payload (HARNESS-054): the harnesses that cannot take
// the whole AGENTS.md (agy caps a rules file at 12000 characters, codex stops
// reading at 32 KiB) get the enforced rules and the persona roster instead, in
// a GENERATED region of their rules file. This is the Go form of
// compile-harness.sh's deploy_doctrine, render_region_compact and
// fold_to_ascii (CLI-026); the reasons behind each step are recorded there and
// in harness/manifest.json next to each row.

// EnforcedDir holds the enforced records the doctrine payload is built from.
const EnforcedDir = "harness/enforced"

// DoctrinePreamble is the line a rules file the deploy creates starts with.
// ASCII, so a file the deploy starts is ASCII and its cap is unambiguous.
const DoctrinePreamble = "> Cross-agent doctrine. The marked region is generated; edit the vault pattern and re-run setup."

// legacyDoctrinePreamble is the same line as written before it was ASCII. A
// file carrying it exactly is brought up to date; a user's look-alike is not.
const legacyDoctrinePreamble = "> Cross-agent doctrine. The marked region is generated \u2014 edit the vault pattern and re-run setup."

// DoctrineTarget is one manifest doctrine.deploy[] entry.
type DoctrineTarget struct {
	Agent      string `json:"agent"`
	File       string `json:"file"`
	CharCap    int    `json:"char_cap"`
	ShadowedBy string `json:"shadowed_by"`
}

// DoctrineManifest is what the doctrine deploy reads from the manifest.
type DoctrineManifest struct {
	VaultSubpath string
	Inject       []string
	Deploy       []DoctrineTarget
}

// LoadDoctrine reads the manifest's doctrine block. It yields nil when the
// manifest has no doctrine.deploy, or no agents block for the roster, as the
// shell deployed doctrine only when both were there.
func LoadDoctrine(manifestPath string) (*DoctrineManifest, error) {
	raw, err := os.ReadFile(manifestPath) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", ManifestFile, err)
	}
	var m struct {
		VaultSubpath string           `json:"vault_subpath"`
		Agents       *json.RawMessage `json:"agents"`
		Doctrine     *struct {
			Inject []string         `json:"inject"`
			Deploy []DoctrineTarget `json:"deploy"`
		} `json:"doctrine"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ManifestFile, err)
	}
	if m.Doctrine == nil || m.Doctrine.Deploy == nil || m.Agents == nil {
		return nil, nil
	}
	for i, t := range m.Doctrine.Deploy {
		if t.Agent == "" || t.File == "" {
			return nil, fmt.Errorf("%s: doctrine.deploy[%d] needs both agent and file", ManifestFile, i)
		}
	}
	return &DoctrineManifest{VaultSubpath: m.VaultSubpath, Inject: m.Doctrine.Inject, Deploy: m.Doctrine.Deploy}, nil
}

var (
	fullOnlyBegin  = "<!-- full-only:begin -->"
	fullOnlyEnd    = "<!-- full-only:end -->"
	injectedNotice = regexp.MustCompile(`^> Injected verbatim into every agent`)
	blankLine      = regexp.MustCompile(`^[[:space:]]*$`)
)

// CompactDoctrine is the enforced records named by ids, concatenated, without
// the provenance notice each opens with nor any full-only region, and without
// the blank line either leaves behind. Every rule survives byte for byte: it
// only ever deletes lines it can identify exactly. An unclosed full-only region
// is an error, since it would drop every rule after it and only make the
// payload smaller, which no cap check can tell from success.
func CompactDoctrine(repoRoot string, ids []string) (string, error) {
	var raw strings.Builder
	for _, id := range ids {
		b, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(EnforcedDir), id+".md")) //nolint:gosec // a record the manifest names
		if err != nil {
			return "", fmt.Errorf("missing source-of-record: %s/%s.md (run --refresh)", EnforcedDir, id)
		}
		raw.Write(b)
	}
	var b strings.Builder
	fullOnly, skipped := false, false
	for _, l := range recordLines([]byte(raw.String())) {
		switch {
		case strings.Contains(l, fullOnlyBegin):
			fullOnly = true
		case strings.Contains(l, fullOnlyEnd):
			fullOnly, skipped = false, true
		case fullOnly:
		case injectedNotice.MatchString(l):
			skipped = true
		case skipped && blankLine.MatchString(l):
			skipped = false
		default:
			skipped = false
			b.WriteString(l + "\n")
		}
	}
	if fullOnly {
		return "", errors.New("a full-only:begin has no matching full-only:end -- the capped payload would lose every rule after it")
	}
	return b.String(), nil
}

// asciiFold is the fixed table a capped payload is folded with. What it does
// not know is left alone and reported, never guessed at.
var asciiFold = strings.NewReplacer(
	"\u2014", "--", "\u2013", "-", // em dash, en dash
	"\u2019", "'", "\u2018", "'", "\u201c", `"`, "\u201d", `"`, // curly quotes
	"\u2026", "...", "\u2192", "->", "\u2190", "<-", // ellipsis, arrows
	"\u2265", ">=", "\u2264", "<=", "\u00d7", "x", // comparisons, multiplication sign
	"\u00a0", " ", "\u00a7", "Section ", // no-break space, section sign
	"\u00e1", "a", "\u00c1", "A", "\u00e9", "e", "\u00c9", "E", "\u00ed", "i", "\u00cd", "I",
	"\u00f3", "o", "\u00d3", "O", "\u00fa", "u", "\u00da", "U", "\u00fc", "u", "\u00dc", "U",
	"\u00f1", "n", "\u00d1", "N",
)

// FoldToASCII folds s with the fixed table, so a capped file has as many bytes
// as characters and its cap means the same in either unit.
func FoldToASCII(s string) string { return asciiFold.Replace(s) }

// nonASCIIRuns lists, as utf-8 hex, the distinct runs of non-ASCII bytes left
// in s, sorted, for the warning that names what the fold did not know.
func nonASCIIRuns(s string) []string {
	seen := map[string]bool{}
	var run []byte
	flush := func() {
		if len(run) > 0 {
			seen[fmt.Sprintf("%x", run)] = true
			run = run[:0]
		}
	}
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			run = append(run, s[i])
			continue
		}
		flush()
	}
	flush()
	var out []string
	for h := range seen {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

// DoctrineOutcome is what DeployDoctrine did for one rules file.
type DoctrineOutcome struct {
	Agent    string
	Path     string
	Status   string   // SkillDeployed or SkillCurrent
	Created  bool     // the rules file did not exist and was started
	Warnings []string // a shadowing file, a leftover non-ASCII run, a breached cap
}

// DeployDoctrine writes the compact payload for each doctrine.deploy[] target
// into the GENERATED region of its rules file under home, replacing the region
// in place or appending one, and starting the file when it is absent. The rest
// of the file is the user's and is kept, apart from the legacy preamble line.
func DeployDoctrine(repoRoot, home string, dryRun bool) ([]DoctrineOutcome, error) {
	m, err := LoadDoctrine(filepath.Join(repoRoot, filepath.FromSlash(ManifestFile)))
	if err != nil || m == nil {
		return nil, err
	}
	rules, err := CompactDoctrine(repoRoot, m.Inject)
	if err != nil {
		return nil, err
	}
	var out []DoctrineOutcome
	var errs []error
	for _, t := range m.Deploy {
		o, err := deployDoctrineTarget(repoRoot, home, m.VaultSubpath, rules, t, dryRun)
		if err != nil {
			errs = append(errs, fmt.Errorf("doctrine for %s: %w", t.Agent, err))
			continue
		}
		out = append(out, *o)
	}
	return out, errors.Join(errs...)
}

func deployDoctrineTarget(repoRoot, home, vaultSubpath, rules string, t DoctrineTarget, dryRun bool) (*DoctrineOutcome, error) {
	presence, err := RenderPresence(repoRoot, t.Agent)
	if err != nil {
		return nil, err
	}
	payload := "## Non-negotiable rules (harness-enforced)\n\n" + rules + "\n" + presence
	if t.CharCap != 0 {
		payload = FoldToASCII(payload)
	}
	file := filepath.Join(home, filepath.FromSlash(t.File))
	o := &DoctrineOutcome{Agent: t.Agent, Path: file, Status: SkillCurrent}
	if t.ShadowedBy != "" {
		if shadow := filepath.Join(home, filepath.FromSlash(t.ShadowedBy)); pathExists(shadow) {
			o.Warnings = append(o.Warnings, shadow+" shadows "+file+" — the deployed doctrine is never read")
		}
	}
	current, err := os.ReadFile(file) //nolint:gosec // a rules file the manifest names under home
	if errors.Is(err, os.ErrNotExist) {
		o.Created, current = true, []byte("# Global rules\n\n"+DoctrinePreamble+"\n")
	} else if err != nil {
		return nil, err
	}
	begin := GeneratedBeginPrefix + " (sha256:" + PresenceSHA(payload) + ") -- vault " + vaultSubpath +
		"; edit there + re-run setup, do NOT edit between markers -->"
	want, err := injectDoctrine(migratePreamble(string(current)), begin, payload)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	wrote, err := writeIfChanged(file, []byte(want), 0o644, dryRun)
	if err != nil {
		return nil, err
	}
	if wrote {
		o.Status = SkillDeployed
	}
	if t.CharCap != 0 {
		o.Warnings = append(o.Warnings, capWarnings(file, want, payload, t.CharCap)...)
	}
	return o, nil
}

func pathExists(p string) bool { _, err := os.Stat(p); return err == nil }

// migratePreamble swaps the legacy preamble line for the ASCII one, on an
// exact line match only.
func migratePreamble(file string) string {
	lines := recordLines([]byte(file))
	found := false
	for i, l := range lines {
		if l == legacyDoctrinePreamble {
			lines[i], found = DoctrinePreamble, true
		}
	}
	if !found {
		return file
	}
	return strings.Join(lines, "\n") + "\n"
}

// injectDoctrine replaces every line starting with the GENERATED prefix, and
// what follows it up to the END marker, with begin, the payload and the END
// marker, as the shell's awk did. A file without both a BEGIN line and an END
// marker gets the region appended after one blank line. A line carrying the
// END marker and anything else is refused, as the catalog refuses it: the
// shell counted it as an END, never matched it, and dropped the rest of the
// file.
func injectDoctrine(file, begin, payload string) (string, error) {
	lines := recordLines([]byte(file))
	hasBegin, hasEnd := false, false
	for _, l := range lines {
		hasBegin = hasBegin || strings.HasPrefix(l, GeneratedBeginPrefix)
		if strings.Contains(l, GeneratedEndMarker) {
			if l != GeneratedEndMarker {
				return "", fmt.Errorf("a line carries %s with more text around it: %q", GeneratedEndMarker, l)
			}
			hasEnd = true
		}
	}
	if !hasBegin || !hasEnd {
		return file + "\n" + begin + "\n" + payload + GeneratedEndMarker + "\n", nil
	}
	var b strings.Builder
	skip := false
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, GeneratedBeginPrefix):
			b.WriteString(begin + "\n")
			for _, p := range recordLines([]byte(payload)) {
				b.WriteString(p + "\n")
			}
			skip = true
		case l == GeneratedEndMarker && skip:
			b.WriteString(l + "\n")
			skip = false
		case skip:
		default:
			b.WriteString(l + "\n")
		}
	}
	return b.String(), nil
}

// capWarnings measures a capped rules file in both units, characters and
// bytes, against the larger, and says whose content breached the cap: the
// generated payload alone (ours to fix) or the file as a whole (the user's).
func capWarnings(file, content, payload string, limit int) []string {
	var w []string
	genChars, genBytes := utf8.RuneCountInString(payload), len(payload)
	if genBytes > genChars {
		w = append(w, fmt.Sprintf("%s: generated doctrine is %d chars / %d bytes; non-ASCII survives the fold (utf-8 hex: %s); "+
			"extend the fold table for it, or reword the vault record", file, genChars, genBytes, strings.Join(nonASCIIRuns(payload), " ")))
	}
	chars, bytes := utf8.RuneCountInString(content), len(content)
	switch {
	case max(genChars, genBytes) > limit:
		w = append(w, fmt.Sprintf("%s: the GENERATED doctrine alone is %d characters / %d bytes, over the %d cap; "+
			"this is ours, not the user's content — drop an id from doctrine.inject in harness/manifest.json, "+
			"or shorten the largest enforced records", file, genChars, genBytes, limit))
	case max(chars, bytes) > limit:
		w = append(w, fmt.Sprintf("%s is %d characters / %d bytes, over the %d the platform documents — content past the cap may never be read; "+
			"the generated doctrine is %d characters / %d bytes of that; the rest is the file's own content", file, chars, bytes, limit, genChars, genBytes))
	}
	return w
}
