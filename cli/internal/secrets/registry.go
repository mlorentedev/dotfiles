package secrets

import (
	"fmt"
	"slices"

	yaml "go.yaml.in/yaml/v3"
)

// Registry is secrets/registry.yaml — the mapping SSOT of ADR-028 §2: it ties each
// logical secret (a stable kebab id) to its store source, the env/file it exposes,
// and its consumers/rotation. Both the age backend (the local age store) and the bw
// backend (Bitwarden, ADR-028 Phase 3) resolve here; the Loader dispatches per entry.
type Registry struct {
	Version int      `yaml:"version"`
	Secrets []Secret `yaml:"secrets"`
}

// Secret is one registry entry. Age is the base name under sensitive/ (no
// .secret.age) used as the source for age/age-offline backends, unless an
// expose.env var overrides it per-var. BW is the Bitwarden source for the bw
// backend, unless an expose.env var overrides the field per-var.
type Secret struct {
	ID        string    `yaml:"id"`
	Plane     string    `yaml:"plane"`   // app | infra | personal | floor
	Backend   string    `yaml:"backend"` // age | age-offline | bw
	Age       string    `yaml:"age"`
	BW        *BWSource `yaml:"bw"`
	Expose    Expose    `yaml:"expose"`
	Consumers []string  `yaml:"consumers"`
	Rotate    string    `yaml:"rotate"`
}

// BWSource is a bw backend source: the Bitwarden item (its unique name or id) and,
// for a single-var or file secret, the field within it. Multi-var secrets share
// the item and set the field per-var (expose.env: { VAR: { field: ... } }). The
// Bitwarden folder is org metadata, not a lookup key — `bw get` resolves by item
// name/id (ADR-028 §2; folder taxonomy is the curation issue).
type BWSource struct {
	Item  string `yaml:"item"`
	Field string `yaml:"field"`
}

// Expose is the consumer contract: exactly one of env (one or many vars) or file.
type Expose struct {
	Env  EnvExpose   `yaml:"env"`
	File *FileExpose `yaml:"file"`
}

// FileExpose materializes the source to Path (Mode, 0600 default) and sets Var to
// the path — the registry form of env-mapping's @VAR=file>dest.
type FileExpose struct {
	Var  string `yaml:"var"`
	Path string `yaml:"path"`
	Mode string `yaml:"mode"`
}

// EnvVar is one exposed env var with optional per-var source overrides: Age for
// age backends, Field for the bw backend.
type EnvVar struct {
	Name  string
	Age   string // "" → use the secret's top-level Age
	Field string // "" → use the secret's top-level BW.Field
}

// EnvExpose normalizes the three YAML shapes of expose.env:
//
//	env: VAR                     → one var (source = secret.age / secret.bw)
//	env: [VAR1, VAR2]            → many vars, same source
//	env: { VAR: {age: file} }    → per-var age source
//	env: { VAR: {field: name} }  → per-var bw field
type EnvExpose struct {
	Vars []EnvVar
}

// UnmarshalYAML dispatches on the node kind so one Go type accepts scalar, list,
// and mapping forms (yaml.v3 cannot do this with struct tags alone).
func (e *EnvExpose) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		e.Vars = []EnvVar{{Name: node.Value}}
	case yaml.SequenceNode:
		for _, n := range node.Content {
			e.Vars = append(e.Vars, EnvVar{Name: n.Value})
		}
	case yaml.MappingNode:
		// Content is a flat [key, value, key, value, ...] list.
		for i := 0; i+1 < len(node.Content); i += 2 {
			var src struct {
				Age   string `yaml:"age"`
				Field string `yaml:"field"`
			}
			if err := node.Content[i+1].Decode(&src); err != nil {
				return err
			}
			e.Vars = append(e.Vars, EnvVar{Name: node.Content[i].Value, Age: src.Age, Field: src.Field})
		}
	default:
		return fmt.Errorf("expose.env: unsupported YAML node (kind %d)", node.Kind)
	}
	return nil
}

// ParseRegistry parses and validates registry.yaml. Validation is fail-fast: a
// malformed registry is a configuration error, never silently tolerated.
func ParseRegistry(data []byte) (*Registry, error) {
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	if err := reg.validate(); err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *Registry) validate() error {
	if r.Version != 1 {
		return fmt.Errorf("registry version %d unsupported (want 1)", r.Version)
	}
	seen := make(map[string]bool, len(r.Secrets))
	for i := range r.Secrets {
		s := &r.Secrets[i]
		if s.ID == "" {
			return fmt.Errorf("secret #%d: empty id", i)
		}
		if seen[s.ID] {
			return fmt.Errorf("duplicate secret id %q", s.ID)
		}
		seen[s.ID] = true

		switch s.Backend {
		case "age", "age-offline", "bw":
		default:
			return fmt.Errorf("secret %q: unknown backend %q", s.ID, s.Backend)
		}

		hasEnv, hasFile := len(s.Expose.Env.Vars) > 0, s.Expose.File != nil
		if hasEnv == hasFile { // both, or neither
			return fmt.Errorf("secret %q: expose must have exactly one of env|file", s.ID)
		}

		// Each backend must resolve a source for everything it exposes.
		switch s.Backend {
		case "age", "age-offline":
			if err := s.checkAgeSources(); err != nil {
				return err
			}
		case "bw":
			if err := s.checkBwSources(); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkAgeSources verifies every exposed var/file has an age source to decrypt.
func (s *Secret) checkAgeSources() error {
	if s.Expose.File != nil {
		if s.Age == "" {
			return fmt.Errorf("secret %q: file expose needs an age source", s.ID)
		}
		if s.Expose.File.Var == "" || s.Expose.File.Path == "" {
			return fmt.Errorf("secret %q: file expose needs var+path", s.ID)
		}
		return nil
	}
	for _, v := range s.Expose.Env.Vars {
		if v.Age == "" && s.Age == "" {
			return fmt.Errorf("secret %q: env %q has no age source", s.ID, v.Name)
		}
	}
	return nil
}

// checkBwSources verifies every exposed var/file has a Bitwarden item + field.
func (s *Secret) checkBwSources() error {
	if s.BW == nil || s.BW.Item == "" {
		return fmt.Errorf("secret %q: bw backend needs bw.item", s.ID)
	}
	if s.Expose.File != nil {
		if s.BW.Field == "" {
			return fmt.Errorf("secret %q: bw file expose needs bw.field", s.ID)
		}
		if s.Expose.File.Var == "" || s.Expose.File.Path == "" {
			return fmt.Errorf("secret %q: file expose needs var+path", s.ID)
		}
		return nil
	}
	for _, v := range s.Expose.Env.Vars {
		if v.Field == "" && s.BW.Field == "" {
			return fmt.Errorf("secret %q: bw env %q has no field source", s.ID, v.Name)
		}
	}
	return nil
}

// Entries flattens the registry into the []Entry the Loader consumes, so run/show/
// render resolve through one path. Each entry is tagged with its Backend; the Loader
// dispatches to the matching Resolver. File exposes become IsFile entries (Dest =
// ~-expanded path); env exposes become one entry per var, each carrying its per-var
// (or the secret's top-level) source — an age base name, or a bw item+field.
func (r *Registry) Entries(home string) []Entry {
	var es []Entry
	for i := range r.Secrets {
		s := &r.Secrets[i]
		if s.Backend == "bw" {
			es = append(es, s.bwEntries(home)...)
			continue
		}
		es = append(es, s.ageEntries(home)...)
	}
	return es
}

// ageEntries flattens an age/age-offline secret. The per-var age override falls
// back to the secret's top-level Age source.
func (s *Secret) ageEntries(home string) []Entry {
	if s.Expose.File != nil {
		return []Entry{{
			Var:     s.Expose.File.Var,
			Backend: s.Backend,
			File:    s.Age,
			IsFile:  true,
			Dest:    expandHome(s.Expose.File.Path, home),
		}}
	}
	es := make([]Entry, 0, len(s.Expose.Env.Vars))
	for _, v := range s.Expose.Env.Vars {
		src := v.Age
		if src == "" {
			src = s.Age
		}
		es = append(es, Entry{Var: v.Name, Backend: s.Backend, File: src})
	}
	return es
}

// bwEntries flattens a bw secret. All vars share the item; the per-var field
// override falls back to the secret's top-level BW.Field.
func (s *Secret) bwEntries(home string) []Entry {
	var item, topField string
	if s.BW != nil {
		item, topField = s.BW.Item, s.BW.Field
	}
	if s.Expose.File != nil {
		return []Entry{{
			Var:     s.Expose.File.Var,
			Backend: "bw",
			Item:    item,
			Field:   topField,
			IsFile:  true,
			Dest:    expandHome(s.Expose.File.Path, home),
		}}
	}
	es := make([]Entry, 0, len(s.Expose.Env.Vars))
	for _, v := range s.Expose.Env.Vars {
		field := v.Field
		if field == "" {
			field = topField
		}
		es = append(es, Entry{Var: v.Name, Backend: "bw", Item: item, Field: field})
	}
	return es
}

// Vars lists the env-var names a secret exposes (the file var for a file secret).
func (s *Secret) Vars() []string {
	if s.Expose.File != nil {
		return []string{s.Expose.File.Var}
	}
	out := make([]string, 0, len(s.Expose.Env.Vars))
	for _, v := range s.Expose.Env.Vars {
		out = append(out, v.Name)
	}
	return out
}

// Selector resolves a --only token to the env-var names it selects: an **id**
// selects all of that secret's vars; an env/file **var name** selects just itself.
// The bool is false when the token matches neither (caller errors).
func (r *Registry) Selector(token string) ([]string, bool) {
	for i := range r.Secrets {
		if r.Secrets[i].ID == token {
			return r.Secrets[i].Vars(), true
		}
	}
	for i := range r.Secrets {
		if slices.Contains(r.Secrets[i].Vars(), token) {
			return []string{token}, true
		}
	}
	return nil, false
}

// ShowEntry resolves a single-env-var secret to its Entry for `show` (one value to
// stdout). File and multi-var secrets are rejected — a single value to stdout is
// ambiguous for them; use `run`. The returned Entry is backend-tagged (age or bw),
// reusing the same flattening as Entries (home is irrelevant: env vars never
// materialize a file).
func (r *Registry) ShowEntry(idOrVar string) (Entry, error) {
	s := r.Lookup(idOrVar)
	if s == nil {
		return Entry{}, fmt.Errorf("unknown secret %q (try `dotf secrets ls`)", idOrVar)
	}
	if s.Expose.File != nil {
		return Entry{}, fmt.Errorf("%q is a file secret; use `dotf secrets run --only %s -- <cmd>`", s.ID, s.ID)
	}
	if len(s.Expose.Env.Vars) != 1 {
		return Entry{}, fmt.Errorf("%q exposes %d vars; use `dotf secrets run --only %s -- <cmd>`", s.ID, len(s.Expose.Env.Vars), s.ID)
	}
	if s.Backend == "bw" {
		return s.bwEntries("")[0], nil
	}
	return s.ageEntries("")[0], nil
}

// Lookup resolves a token to its secret: id first (the stable handle), then an
// exposed env-var or file var name. Returns nil when unknown.
func (r *Registry) Lookup(idOrVar string) *Secret {
	for i := range r.Secrets {
		if r.Secrets[i].ID == idOrVar {
			return &r.Secrets[i]
		}
	}
	for i := range r.Secrets {
		s := &r.Secrets[i]
		if s.Expose.File != nil {
			if s.Expose.File.Var == idOrVar {
				return s
			}
			continue
		}
		for _, v := range s.Expose.Env.Vars {
			if v.Name == idOrVar {
				return s
			}
		}
	}
	return nil
}
