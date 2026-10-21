package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// Skill records are whole-file transforms: one committed
// harness/skills/<name>/SKILL.md becomes one agent-native output per
// skills.deploy[] entry. This is the Go form of compile-harness.sh's
// render_skill and skill_targets_agent, and of their PowerShell twins
// (Convert-SkillRecord, Test-SkillTargetsAgent) in setup-windows.ps1 (CLI-026).

// The three render kinds a skills.deploy[] entry may name.
const (
	RenderSkill   = "skill"   // full SKILL.md plus provenance fields (claude, agy, copilot, pi)
	RenderCommand = "command" // as skill, minus `name:` (opencode keys a command off its filename)
	RenderPrompt  = "prompt"  // body only, behind a provenance comment (agy's flat prompts)
)

// SkillDeployTarget is one manifest skills.deploy[] entry.
type SkillDeployTarget struct {
	Agent           string `json:"agent"`
	Render          string `json:"render"`
	Dir             string `json:"dir"`
	RequiresCommand string `json:"requires_command"`
}

// SkillCatalog is the manifest's skills.catalog: the agent with no per-skill
// mechanism, and the instructions file its catalog region lives in.
type SkillCatalog struct {
	Agent string `json:"agent"`
	File  string `json:"file"`
}

// SkillManifest is the manifest's skills block.
type SkillManifest struct {
	VaultSubpath string              `json:"vault_subpath"`
	RecordDir    string              `json:"record_dir"`
	Deploy       []SkillDeployTarget `json:"deploy"`
	Catalog      *SkillCatalog       `json:"catalog"`
}

// LoadSkills reads the manifest's skills block. A manifest without one yields
// nil and no error: the shell treated that as "nothing to deploy" too.
func LoadSkills(manifestPath string) (*SkillManifest, error) {
	raw, err := os.ReadFile(manifestPath) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", ManifestFile, err)
	}
	var m struct {
		Skills *SkillManifest `json:"skills"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ManifestFile, err)
	}
	if m.Skills == nil {
		return nil, nil
	}
	if m.Skills.RecordDir == "" {
		return nil, fmt.Errorf("%s: skills.record_dir is empty", ManifestFile)
	}
	for i, t := range m.Skills.Deploy {
		switch {
		case t.Agent == "" || t.Dir == "":
			return nil, fmt.Errorf("%s: skills.deploy[%d] needs both agent and dir", ManifestFile, i)
		case t.Render != RenderSkill && t.Render != RenderCommand && t.Render != RenderPrompt:
			return nil, fmt.Errorf("%s: skills.deploy[%d] has unknown render %q", ManifestFile, i, t.Render)
		}
	}
	return m.Skills, nil
}

// SkillTargetsAgent reports whether a record deploys to agent. An absent
// `targets:` means every agent. A declared one is matched by exact name,
// inline or block list alike: the shell matched the harness name as a
// substring of the first `targets:` line, so `[copilot]` reached pi and a
// block list reached nobody (#1733, audit C21).
func SkillTargetsAgent(record []byte, agent string) (bool, error) {
	front, _, err := splitRecord(normalizeLF(record))
	if err != nil {
		return false, err
	}
	var fm map[string]any
	if err := yaml.Unmarshal(front, &fm); err != nil {
		return false, fmt.Errorf("frontmatter is not valid YAML: %w", err)
	}
	v, declared := fm["targets"]
	if !declared {
		return true, nil
	}
	var names []string
	switch t := v.(type) {
	case nil:
	case string:
		names = []string{t}
	case []any:
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				return false, fmt.Errorf("targets entry %v is not a harness name", e)
			}
			names = append(names, s)
		}
	default:
		return false, fmt.Errorf("targets is neither a name nor a list of names")
	}
	for _, n := range names {
		if strings.TrimSpace(n) == agent {
			return true, nil
		}
	}
	return false, nil
}

var (
	fenceLine = regexp.MustCompile(`^---[[:space:]]*$`)
	keyLine   = regexp.MustCompile(`^[a-zA-Z0-9_-]+:`)
	// The execution fields agent runtimes read. Every other key (id, type,
	// paths, targets, ...) is store-only; dropping `paths:` is what lets Claude
	// Code load a skill unconditionally at session start.
	nativeKey = regexp.MustCompile(`^(name|description|allowed-tools|when_to_use|model|effort|context|argument-hint|arguments|user-invocable|disable-model-invocation):`)
	// A record carries its own generated_* fields, which describe it against
	// the vault. The render strips them and injects one fresh set describing
	// the output against the record, so no file stacks two sets (HARNESS-069).
	provenanceKey = regexp.MustCompile(`^generated(_from|_sha)?:`)
)

// RenderSkillRecord renders one record as kind. srcPath is the vault-relative
// path a human edits; the sha is of the record the output is rendered from.
// Line endings are normalized to LF first, so a CRLF checkout renders the same
// bytes and the same sha as an LF one.
func RenderSkillRecord(kind string, record []byte, srcPath string) string {
	record = normalizeLF(record)
	sha := PresenceSHA(string(record))
	lines := recordLines(record)
	var b strings.Builder
	emit := func(l string) { b.WriteString(l); b.WriteByte('\n') }
	if kind == RenderPrompt {
		emit("<!-- generated: true; from: " + srcPath + "; sha256:" + sha + "; edit the vault source + re-run setup -->")
		emit("")
		fences := 0
		for _, l := range lines {
			// Only the two frontmatter fences go. The shell dropped every `---`
			// line, so a rule in a prompt's body vanished on Linux and macOS
			// and survived on Windows.
			if fences < 2 && fenceLine.MatchString(l) {
				fences++
				continue
			}
			if fences != 1 {
				emit(l)
			}
		}
		return b.String()
	}
	fences, keep := 0, false
	for _, l := range lines {
		if fenceLine.MatchString(l) && fences < 2 {
			fences++
			emit(l)
			if fences == 1 {
				emit("generated: true")
				emit("generated_from: " + srcPath)
				emit("generated_sha: " + sha)
			}
			continue
		}
		if fences == 1 {
			// A key line decides its own fate and that of the indented lines
			// under it; anything before the first key is dropped.
			if keyLine.MatchString(l) {
				keep = keepSkillKey(kind, l)
			}
			if keep {
				emit(l)
			}
			continue
		}
		emit(l)
	}
	return b.String()
}

func keepSkillKey(kind, line string) bool {
	if kind == RenderCommand && strings.HasPrefix(line, "name:") {
		return false
	}
	return !provenanceKey.MatchString(line) && nativeKey.MatchString(line)
}

// normalizeLF turns CRLF into LF, the one line ending every output carries.
func normalizeLF(b []byte) []byte {
	return []byte(strings.ReplaceAll(string(b), "\r\n", "\n"))
}

// recordLines splits LF text the way awk reads it: a final line without a
// newline is still a line, and a trailing newline opens no empty one.
func recordLines(b []byte) []string {
	s := string(b)
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
