package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// Enforcement is what the gate does when a forced skill has not been consumed.
type Enforcement string

const (
	// EnforceBlock stops the tool call: claude `exit 2`, pi's fail-safe error,
	// opencode's deny.
	EnforceBlock Enforcement = "block"
	// EnforceWarn emits and allows.
	EnforceWarn Enforcement = "warn"
	// EnforceUnset is a skill declared in the OLD flat form, carrying no
	// severity. It is deliberately NOT given a default — see SkillBinding.
	EnforceUnset Enforcement = ""
)

// SkillBinding is one forced skill and what failing to consume it costs.
//
// NO DEFAULT IS APPLIED to a skill that declares no `enforce`, and that is the
// design rather than an omission. Defaulting to `warn` would make every
// unmigrated persona's gate silently inert while every check reported it as
// wired — presence dressed as enforcement. Defaulting to `block` would turn
// every existing skill into a hard gate the moment this shipped. So an
// undeclared severity resolves to EnforceUnset, the gate refuses to act on it,
// and `UnmigratedSkills` makes it visible.
type SkillBinding struct {
	ID      string
	Enforce Enforcement
}

// Persona is an agent record under the manifest's `record_dir`.
type Persona struct {
	Name        string
	Kind        string
	Model       string // neutral tier: top | mid | low
	Description string
	Skills      []SkillBinding
	// Capabilities are the NEUTRAL verbs the record declares; the capability map
	// turns them into one harness's native tool grant. Read here so the tie
	// between the two frontmatter keys can be asserted: a record declaring
	// `skills:` without the `skill` capability deploys an agent that cannot
	// invoke the skills its own gate demands (#1420).
	Capabilities []string
	Targets      []string // empty means every harness
	Path         string
}

// personaFrontmatter mirrors the YAML. `Skills` is `any` because the field has
// two shapes during migration and telling them apart is the loader's job, not
// the caller's.
type personaFrontmatter struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Model        string   `yaml:"model"`
	Description  string   `yaml:"description"`
	Targets      []string `yaml:"targets"`
	Skills       any      `yaml:"skills"`
	Capabilities []string `yaml:"capabilities"`
}

// LoadPersona reads and parses one agent record.
//
// It uses a real YAML parser rather than the line-scanning readers elsewhere in
// this repository, and that is a correctness decision with a precedent behind
// it. `specs/HARNESS-046/check-roster-consistency.py` matches `skills:` with
// `^skills:\s*\[(.*?)\]` and falls back to `[]` when that fails, so the block
// form this spec introduces would make that guard return "no skills" in silence.
// `doctor.readAgentFrontmatter` skips indented lines outright, so it cannot see
// a block list either. Hand-rolled frontmatter parsing is how a schema change
// turns a guard into a no-op; the module already depends on yaml/v3.
func LoadPersona(path string) (*Persona, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- path is built from the manifest's own record_dir
	if err != nil {
		return nil, fmt.Errorf("read persona %s: %w", filepath.Base(path), err)
	}
	body, err := frontmatterBlock(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}

	var fm personaFrontmatter
	if err := yaml.Unmarshal(body, &fm); err != nil {
		return nil, fmt.Errorf("%s: frontmatter is not valid YAML: %w", filepath.Base(path), err)
	}

	skills, err := parseSkills(fm.Skills)
	if err != nil {
		// C15, applied to a persona: an unreadable `skills:` is never an empty
		// one. The two would produce identical downstream behaviour — a gate
		// that enforces nothing — and mean opposite things.
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}

	return &Persona{
		Name:         fm.Name,
		Kind:         fm.Kind,
		Model:        fm.Model,
		Description:  fm.Description,
		Skills:       skills,
		Capabilities: fm.Capabilities,
		Targets:      fm.Targets,
		Path:         path,
	}, nil
}

// LoadPersonas reads every record under dir, sorted by name for stable output.
func LoadPersonas(dir string) ([]*Persona, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read persona dir %s: %w", dir, err)
	}
	var out []*Persona
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name(), "AGENT.md")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		p, err := LoadPersona(path)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no persona records under %s — refusing to report an empty roster as a valid one", dir)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// frontmatterBlock extracts the YAML between the leading `---` fences.
func frontmatterBlock(raw []byte) ([]byte, error) {
	s := string(raw)
	s = strings.TrimPrefix(s, "\ufeff")
	if !strings.HasPrefix(s, "---") {
		return nil, fmt.Errorf("no frontmatter fence")
	}
	rest := s[3:]
	// Accept both \n and \r\n fences; a record deployed on Windows carries CRLF.
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, fmt.Errorf("frontmatter is not closed")
	}
	return []byte(rest[:end]), nil
}

// parseSkills reads either shape:
//
//	skills: [audit, verification-before-completion]      # legacy, no severity
//	skills:                                              # this spec's form
//	  - id: audit
//	    enforce: block
//
// A string entry yields EnforceUnset rather than a default. An entry that is
// neither a string nor an {id, enforce} map is an error, never a skip: silently
// dropping a malformed entry is how a gate ends up enforcing less than the
// record says it does.
func parseSkills(v any) ([]SkillBinding, error) {
	if v == nil {
		return nil, nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("`skills` is %T, want a list", v)
	}
	out := make([]SkillBinding, 0, len(items))
	for i, item := range items {
		switch t := item.(type) {
		case string:
			id := strings.TrimSpace(t)
			if id == "" {
				return nil, fmt.Errorf("`skills[%d]` is an empty string", i)
			}
			out = append(out, SkillBinding{ID: id, Enforce: EnforceUnset})
		case map[string]any:
			id, _ := t["id"].(string)
			if strings.TrimSpace(id) == "" {
				return nil, fmt.Errorf("`skills[%d]` has no `id`", i)
			}
			enf, _ := t["enforce"].(string)
			switch Enforcement(enf) {
			case EnforceBlock, EnforceWarn:
			case EnforceUnset:
				return nil, fmt.Errorf("`skills[%d]` (%s) declares no `enforce` — severity is required in the mapping form", i, id)
			default:
				return nil, fmt.Errorf("`skills[%d]` (%s) has enforce %q, want block or warn", i, id, enf)
			}
			out = append(out, SkillBinding{ID: strings.TrimSpace(id), Enforce: Enforcement(enf)})
		default:
			return nil, fmt.Errorf("`skills[%d]` is %T, want a string or an {id, enforce} mapping", i, item)
		}
	}
	return out, nil
}

// UnmigratedSkills lists skills still declared in the legacy flat form.
//
// The gate will not act on these, so they must be visible rather than inferred
// from a quiet absence of enforcement — the whole reason EnforceUnset exists.
func (p *Persona) UnmigratedSkills() []string {
	var out []string
	for _, s := range p.Skills {
		if s.Enforce == EnforceUnset {
			out = append(out, s.ID)
		}
	}
	return out
}

// Blocking lists the skills whose absence stops a tool call.
func (p *Persona) Blocking() []string {
	var out []string
	for _, s := range p.Skills {
		if s.Enforce == EnforceBlock {
			out = append(out, s.ID)
		}
	}
	return out
}

// AppliesTo reports whether this persona is emitted for a harness. An ABSENT
// targets list means every harness — the same default the render uses. Getting
// it backwards would scope a persona to one harness and fail it against every
// other, a false positive on correct data.
func (p *Persona) AppliesTo(harness string) bool {
	if len(p.Targets) == 0 {
		return true
	}
	for _, t := range p.Targets {
		if strings.EqualFold(strings.TrimSpace(t), harness) {
			return true
		}
	}
	return false
}
