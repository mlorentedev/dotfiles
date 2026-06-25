package secrets

import (
	"fmt"
	"slices"

	yaml "go.yaml.in/yaml/v3"
)

// Registry is secrets/registry.yaml — the mapping SSOT of ADR-028 §2: it ties each
// logical secret (a stable kebab id) to its store source, the env/file it exposes,
// and its consumers/rotation. This package reads the age backend (current reality);
// `backend: bw` resolution lands when items migrate (ADR-028 Phase 3).
type Registry struct {
	Version int      `yaml:"version"`
	Secrets []Secret `yaml:"secrets"`
}

// Secret is one registry entry. Age is the base name under sensitive/ (no
// .secret.age) used as the source for age/age-offline backends, unless an
// expose.env var overrides it per-var.
type Secret struct {
	ID        string   `yaml:"id"`
	Plane     string   `yaml:"plane"`   // app | infra | personal | floor
	Backend   string   `yaml:"backend"` // age | age-offline | bw
	Age       string   `yaml:"age"`
	Expose    Expose   `yaml:"expose"`
	Consumers []string `yaml:"consumers"`
	Rotate    string   `yaml:"rotate"`
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

// EnvVar is one exposed env var with an optional per-var age source override.
type EnvVar struct {
	Name string
	Age  string // "" → use the secret's top-level Age
}

// EnvExpose normalizes the three YAML shapes of expose.env:
//
//	env: VAR                    → one var (source = secret.age)
//	env: [VAR1, VAR2]           → many vars, same source
//	env: { VAR: {age: file} }   → per-var sources
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
				Age string `yaml:"age"`
			}
			if err := node.Content[i+1].Decode(&src); err != nil {
				return err
			}
			e.Vars = append(e.Vars, EnvVar{Name: node.Content[i].Value, Age: src.Age})
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

		// age/age-offline must resolve a source for everything they expose; bw
		// sources are validated when the bw backend lands (Phase 3).
		if s.Backend == "age" || s.Backend == "age-offline" {
			if err := s.checkAgeSources(); err != nil {
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

// Entries flattens the registry into the same []Entry the age Loader consumes, so
// `run` resolves through the existing decrypt path. File exposes become IsFile
// entries (Dest = ~-expanded path); env exposes become one entry per var, each
// pointing at its per-var or the secret's top-level age source.
func (r *Registry) Entries(home string) []Entry {
	var es []Entry
	for i := range r.Secrets {
		s := &r.Secrets[i]
		if s.Expose.File != nil {
			es = append(es, Entry{
				Var:    s.Expose.File.Var,
				File:   s.Age,
				IsFile: true,
				Dest:   expandHome(s.Expose.File.Path, home),
			})
			continue
		}
		for _, v := range s.Expose.Env.Vars {
			src := v.Age
			if src == "" {
				src = s.Age
			}
			es = append(es, Entry{Var: v.Name, File: src})
		}
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
