package harness

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Agent presence is the forced-skills roster every harness's always-loaded
// instructions file carries between AGENT-PRESENCE markers: one line per
// invocable persona that targets the harness, naming the skills it MUST
// consume. It is how a persona's enforcement reaches a harness that has no
// provider hook — uniform text injection, the same on every OS.
//
// It used to live in compile-harness.sh only (build_agent_presence /
// inject_agent_presence), which setup-linux.sh runs and setup-windows.ps1 never
// ported: measured 2026-08-27, zero AGENT-PRESENCE regions in any of the four
// instructions files on the Windows box (HARNESS-092, #1326). This package is
// the one implementation both setups call; the shell twin delegates here.

const (
	// PresenceBeginPrefix opens the region; the full begin line carries the
	// block's sha so a reader can tell "current" from "stale" without
	// re-rendering. PresenceEndMarker closes it. Both spellings are shared
	// with doctor's region stripper and must not change independently.
	PresenceBeginPrefix = "<!-- BEGIN HARNESS AGENT-PRESENCE"
	PresenceEndMarker   = "<!-- END HARNESS AGENT-PRESENCE -->"

	presenceHeader = "## Active agent personas — forced skills\n\nWhen acting as one, you MUST consume its skills.\n\n"
	presenceNote   = " — agent presence from vault agent definitions; edit there + re-run setup, do NOT edit between markers -->"
)

// PresenceTarget is one manifest `agents.presence[]` entry: which harness, which
// file under $HOME, and (optionally) a command that must exist for the file to
// be a real surface at all.
type PresenceTarget struct {
	Agent           string `json:"agent"`
	File            string `json:"file"`
	RequiresCommand string `json:"requires_command"`
}

// PresenceState is what doctor reports per target.
type PresenceState string

const (
	PresenceCurrent PresenceState = "current" // region present, sha matches the roster
	PresenceStale   PresenceState = "stale"   // region present, roster changed since
	PresenceMissing PresenceState = "missing" // file exists, no region
)

// PresenceOutcome is what DeployPresence did for one target.
type PresenceOutcome struct {
	Agent  string
	File   string // absolute path
	Status string // "injected" | "unchanged" | "absent" | "empty"
}

// LoadPresence reads the manifest's `agents.record_dir` and `agents.presence[]`.
// A manifest without a presence list yields no targets and no error: that is
// the manifest saying "no presence", which the shell also treated as a skip.
func LoadPresence(manifestPath string) (recordDir string, targets []PresenceTarget, err error) {
	raw, err := os.ReadFile(manifestPath) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		return "", nil, fmt.Errorf("reading %s: %w", ManifestFile, err)
	}
	var m struct {
		Agents struct {
			RecordDir string           `json:"record_dir"`
			Presence  []PresenceTarget `json:"presence"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", nil, fmt.Errorf("parsing %s: %w", ManifestFile, err)
	}
	for i, t := range m.Agents.Presence {
		if t.Agent == "" || t.File == "" {
			return "", nil, fmt.Errorf("%s: agents.presence[%d] needs both agent and file", ManifestFile, i)
		}
	}
	return m.Agents.RecordDir, m.Agents.Presence, nil
}

// BuildPresence renders the roster for one harness: the header, then one line
// per invocable persona that targets it, in persona-name order. An empty string
// means no persona targets this harness and the caller must inject nothing —
// an empty roster is not a roster, and a region announcing "no personas" would
// read as enforcement where there is none.
//
// The text is byte-identical to compile-harness.sh's build_agent_presence for
// every record, which is what lets the two shas agree across a Linux box that
// deployed from shell yesterday and a Windows box deploying from Go today.
func BuildPresence(personas []*Persona, agent string) string {
	var b strings.Builder
	for _, p := range personas {
		if p.Kind == "autonomous" || !p.AppliesTo(agent) {
			continue
		}
		if b.Len() == 0 {
			b.WriteString(presenceHeader)
		}
		skills := "none"
		if len(p.Skills) > 0 {
			ids := make([]string, 0, len(p.Skills))
			for _, s := range p.Skills {
				ids = append(ids, s.ID)
			}
			skills = "[" + strings.Join(ids, ", ") + "]"
		}
		fmt.Fprintf(&b, "- **%s** — MUST consume: %s\n", p.Name, skills)
	}
	return b.String()
}

// PresenceSHA is the first 16 hex characters of the block's sha256, computed
// over the LF form so a CRLF instructions file on Windows carries the same sha
// its Linux twin does (`sha256sum | cut -c1-16` in the shell).
func PresenceSHA(block string) string {
	sum := sha256.Sum256([]byte(strings.ReplaceAll(block, "\r\n", "\n")))
	return hex.EncodeToString(sum[:])[:16]
}

// presenceBeginLine is the full begin marker for a block.
func presenceBeginLine(block string) string {
	return PresenceBeginPrefix + " (sha256:" + PresenceSHA(block) + ")" + presenceNote
}

// ErrPresenceTargetAbsent marks an instructions file that does not exist: the
// harness's base file has not been deployed, so there is nothing to inject
// into. A genuine skip, not a failure — the shell returned 1 and its caller
// logged "target absent, skipping".
var ErrPresenceTargetAbsent = errors.New("presence target absent")

// InjectPresence writes the block into path between the markers: replacing an
// existing region in place, or appending a fresh one after the file's content.
// Everything outside the region — the user's prose, the GENERATED patterns
// region — is left byte-identical. The file's own line ending is honoured so a
// CRLF file stays CRLF. Returns false when the region already holds this exact
// block, in which case nothing is written.
func InjectPresence(path, block string) (bool, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // manifest-declared target under $HOME
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("%w: %s", ErrPresenceTargetAbsent, path)
		}
		return false, err
	}
	nl := "\n"
	if bytes.Contains(raw, []byte("\r\n")) {
		nl = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	region := append([]string{presenceBeginLine(block)}, strings.Split(strings.TrimSuffix(block, "\n"), "\n")...)
	region = append(region, PresenceEndMarker)

	begin, end := -1, -1
	for i, l := range lines {
		if begin < 0 && strings.HasPrefix(l, PresenceBeginPrefix) {
			begin = i
			continue
		}
		if begin >= 0 && l == PresenceEndMarker {
			end = i
			break
		}
	}
	var out []string
	switch {
	case begin >= 0 && end > begin:
		if equalLines(lines[begin:end+1], region) {
			return false, nil
		}
		out = append(out, lines[:begin]...)
		out = append(out, region...)
		out = append(out, lines[end+1:]...)
	default:
		// Append: a blank line, the region, and a final newline — the shell's
		// `printf '\n%s\n' "$begin"; cat content; printf '%s\n' end` shape.
		out = append(out, strings.TrimSuffix(strings.Join(lines, "\n"), "\n"))
		out = append(out, "")
		out = append(out, region...)
		out = append(out, "")
		out = strings.Split(strings.Join(out, "\n"), "\n")
	}
	data := []byte(strings.Join(out, nl))
	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // an instructions file, world-readable by design
		return false, err
	}
	return true, nil
}

func equalLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// RenderPresence returns the roster block for one harness from the checkout's
// records — the `--render` half of the verb, which writes nothing. It is what
// the shell's --check path compares a deployed region against.
func RenderPresence(repoRoot, agent string) (string, error) {
	recordDir, _, err := LoadPresence(filepath.Join(repoRoot, filepath.FromSlash(ManifestFile)))
	if err != nil {
		return "", err
	}
	if recordDir == "" {
		return "", fmt.Errorf("%s: agents.record_dir is not set", ManifestFile)
	}
	personas, err := LoadPersonas(filepath.Join(repoRoot, filepath.FromSlash(recordDir)))
	if err != nil {
		return "", err
	}
	return BuildPresence(personas, agent), nil
}

// DeployPresence renders and injects the roster for every manifest target,
// reading the records once. Outcomes are returned in manifest order so the
// caller can print exactly what happened per file; only a broken record or an
// unwritable file is an error.
func DeployPresence(repoRoot, home string) ([]PresenceOutcome, error) {
	recordDir, targets, err := LoadPresence(filepath.Join(repoRoot, filepath.FromSlash(ManifestFile)))
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, nil
	}
	personas, err := LoadPersonas(filepath.Join(repoRoot, filepath.FromSlash(recordDir)))
	if err != nil {
		return nil, err
	}
	out := make([]PresenceOutcome, 0, len(targets))
	for _, t := range targets {
		o := PresenceOutcome{Agent: t.Agent, File: filepath.Join(home, filepath.FromSlash(t.File))}
		block := BuildPresence(personas, t.Agent)
		if block == "" {
			o.Status = "empty"
			out = append(out, o)
			continue
		}
		changed, err := InjectPresence(o.File, block)
		switch {
		case errors.Is(err, ErrPresenceTargetAbsent):
			o.Status = "absent"
		case err != nil:
			return out, fmt.Errorf("presence for %s: %w", t.Agent, err)
		case changed:
			o.Status = "injected"
		default:
			o.Status = "unchanged"
		}
		out = append(out, o)
	}
	return out, nil
}

// PresenceStatus reads the region an instructions file carries and compares
// its sha with the block the records render today. It writes nothing; doctor
// calls it.
func PresenceStatus(path, block string) (PresenceState, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // manifest-declared target under $HOME
	if err != nil {
		return "", err
	}
	want := "(sha256:" + PresenceSHA(block) + ")"
	for _, l := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(l, PresenceBeginPrefix) {
			continue
		}
		if strings.Contains(l, want) {
			return PresenceCurrent, nil
		}
		return PresenceStale, nil
	}
	return PresenceMissing, nil
}
