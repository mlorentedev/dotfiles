package secrets

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// validVarName is the env-identifier grammar a var must satisfy to be injectable and
// {env:VAR}-referenceable. validAgeBase guards an age source base name against path
// traversal: it is joined into sensitive/<base>.secret.age, so a "/" or ".." could
// read outside the store. Both are enforced at parse time (#612 B1/B5).
var (
	validVarName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	validAgeBase = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
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
	Validate  string    `yaml:"validate"` // optional liveness check on sync (e.g. "github-token")
	// Recipient is the age PUBLIC recipient a file-authority secret must derive to.
	// Public by construction — it is what someone uses to encrypt TO this key — so
	// it is committed, unlike everything else this file points at.
	//
	// Optional: without it the secret verifies exactly as before. With it, verify
	// answers a question nothing else can — is the key on this disk still the key
	// that was declared — which a replaced, truncated or wrongly-restored root
	// otherwise passes right up until the day it is needed (#1000 AC3).
	Recipient string `yaml:"recipient"`
}

// BWSource is a bw backend source: the Bitwarden item (its unique name or id) and,
// for a single-var or file secret, the field within it. Multi-var secrets share
// the item and set the field per-var (expose.env: { VAR: { field: ... } }). The
// Bitwarden folder is not a lookup key — `bw get` resolves by item name/id
// regardless of folder — but it IS placement metadata a newly created item is
// filed under (OPS-028; ADR-028 §"Bitwarden folder taxonomy").
type BWSource struct {
	Item   string `yaml:"item"`
	Field  string `yaml:"field"`
	Folder string `yaml:"folder"` // "" → unfoldered; else one of validBWFolders
}

// validBWFolders is ADR-028's ratified Bitwarden folder taxonomy for dotf-secrets-
// managed items. floor is deliberately absent (floor secrets never carry a
// bw: block — age-only) and so is a personal-plane folder (no taxonomy exists yet for
// plane: personal, deferred to #586) — declaring either here would validate a
// placement nothing can actually honour yet.
var validBWFolders = map[string]bool{
	"apps":  true,
	"infra": true,
}

// planeFolder is the required bw.folder for a plane that has one — the ratified-set
// check alone (validBWFolders) would let an app-plane secret declare infra
// and pass, since both strings are individually valid; this closes that gap (OPS-028
// adversarial review, Minor finding). A plane absent here (personal, floor) has no
// required folder and is left to the ratified-set check alone.
var planeFolder = map[string]string{
	"app":   "apps",
	"infra": "infra",
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

// SecretDefect is one secret that failed validation, kept apart from the registry so a
// caller can report it instead of being stopped by it.
type SecretDefect struct {
	ID  string // the secret's id, or "#<index>" when the id itself is missing
	Err error
}

// ParseRegistry parses and validates the registry, failing on the FIRST bad secret.
//
// This is the strict door, and it stays the default for every caller: a half-valid
// registry is exactly the state in which `set`, `migrate` and `render` must not run,
// because they write. Only `dotf secrets verify` uses the partial door below.
//
// It is implemented ON TOP of ParseRegistryPartial rather than beside it — one set of
// per-secret checks, two policies over the result. Two independent validation paths is
// the shape that produced BUG-084 (#993): the moment one moves, they disagree, and the
// disagreement is silent.
func ParseRegistry(data []byte) (*Registry, error) {
	reg, defects, err := ParseRegistryPartial(data)
	if err != nil {
		return nil, err
	}
	if len(defects) > 0 {
		return nil, defects[0].Err
	}
	return reg, nil
}

// ParseRegistryPartial parses the registry and validates each secret INDEPENDENTLY,
// returning the well-formed ones plus a defect per secret that failed.
//
// It exists for BUG-086 (#1004): `verify` is a health check, and a health check whose
// job is to tell you what is broken must not be the first thing to break. Before this,
// one malformed entry made the whole registry unloadable, so a typo in a secret the
// caller never asked about hid the health of all the others.
//
// Structural failures are still fatal, and returned as an error rather than a defect:
// unparseable YAML or an unsupported version leave nothing to report per-secret. The
// distinction is "cannot read the document" versus "read it, and this entry is wrong".
//
// Defective secrets are EXCLUDED from the returned registry. They are not merely
// flagged, because the rest of this package treats validation as having happened —
// parseFileMode, for one, documents that a bad mode "is impossible in practice" because
// validate() rejected it. Letting an invalid secret reach Entries() would trade a clear
// failure for an undefined one.
func ParseRegistryPartial(data []byte) (*Registry, []SecretDefect, error) {
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, nil, fmt.Errorf("parse registry: %w", err)
	}
	if reg.Version != 1 {
		return nil, nil, fmt.Errorf("registry version %d unsupported (want 1)", reg.Version)
	}

	seen := make(map[string]bool, len(reg.Secrets))
	seenVar := make(map[string]string) // var name -> first secret id that exposed it
	kept := make([]Secret, 0, len(reg.Secrets))
	var defects []SecretDefect

	for i := range reg.Secrets {
		s := &reg.Secrets[i]
		if err := validateSecret(s, i, seen, seenVar); err != nil {
			defects = append(defects, SecretDefect{ID: secretLabel(s, i), Err: err})
			continue
		}
		// Registered only on success, so a defective secret cannot claim an id or a
		// var name and make the NEXT valid secret look like the duplicate.
		seen[s.ID] = true
		for _, v := range s.Vars() {
			seenVar[v] = s.ID
		}
		kept = append(kept, *s)
	}
	reg.Secrets = kept
	return &reg, defects, nil
}

// secretLabel names a secret for a defect message, falling back to its position when
// the id is the very thing that is missing.
func secretLabel(s *Secret, i int) string {
	if s.ID == "" {
		return fmt.Sprintf("#%d", i)
	}
	return s.ID
}

// validateSecret runs every per-secret rule. seen/seenVar carry the cross-secret
// uniqueness state; this function only READS them — registration is the caller's, so
// that a rejected secret never reserves anything.
func validateSecret(s *Secret, i int, seen map[string]bool, seenVar map[string]string) error {
	if s.ID == "" {
		return fmt.Errorf("secret #%d: empty id", i)
	}
	if seen[s.ID] {
		return fmt.Errorf("duplicate secret id %q", s.ID)
	}
	// ValidBackends is the SSOT for this list; a resolver-coverage test binds
	// it to the Loader, so a backend accepted here always has somewhere to go.
	if !slices.Contains(ValidBackends(), s.Backend) {
		return fmt.Errorf("secret %q: unknown backend %q", s.ID, s.Backend)
	}
	if err := checkExpose(s); err != nil {
		return err
	}
	// Each backend must resolve a source for everything it exposes.
	switch s.Backend {
	case BackendAge, BackendAgeOffline:
		if err := s.checkAgeSources(); err != nil {
			return err
		}
	case BackendBW:
		if err := s.checkBwSources(); err != nil {
			return err
		}
	case BackendFileAuthority:
		if err := s.checkFileAuthoritySources(); err != nil {
			return err
		}
	}
	if err := checkRecipient(s); err != nil {
		return err
	}
	if err := checkBWFolder(s); err != nil {
		return err
	}
	return checkVarNames(s, seenVar)
}

// checkExpose enforces that a secret exposes exactly one of env|file, and that a file
// expose's mode is octal. The mode is applied at materialization (#612 B2), so it is
// rejected here rather than degrading to a silent 0600 much later.
func checkExpose(s *Secret) error {
	hasEnv, hasFile := len(s.Expose.Env.Vars) > 0, s.Expose.File != nil
	if hasEnv == hasFile { // both, or neither
		return fmt.Errorf("secret %q: expose must have exactly one of env|file", s.ID)
	}
	if hasFile && s.Expose.File.Mode != "" {
		if _, err := strconv.ParseUint(s.Expose.File.Mode, 8, 32); err != nil {
			return fmt.Errorf("secret %q: invalid file mode %q (want octal, e.g. 0600)", s.ID, s.Expose.File.Mode)
		}
	}
	return nil
}

// checkBWFolder validates bw.folder against the ratified taxonomy and the secret's plane.
//
// bw.folder is pre-declared dormant metadata (ADR-028 §2 addendum): a secret's bw: block,
// folder included, is written up front while backend is still age and only activated on
// migrate. Gating this on backend == "bw" (as checkBwSources does for item/field) would
// leave every dormant folder value unvalidated until the moment migrate reads it and
// hands it straight to ResolveFolder — which CREATES an arbitrary Bitwarden folder for a
// typo, exactly the drift this taxonomy exists to prevent (OPS-028 adversarial review,
// Major finding). So this runs for every secret carrying a bw: block, whatever its
// current backend.
func checkBWFolder(s *Secret) error {
	if s.BW == nil || s.BW.Folder == "" {
		return nil
	}
	if !validBWFolders[s.BW.Folder] {
		return fmt.Errorf("secret %q: bw.folder %q is not in the ratified taxonomy (apps, infra)", s.ID, s.BW.Folder)
	}
	if want := planeFolder[s.Plane]; want != "" && s.BW.Folder != want {
		return fmt.Errorf("secret %q: bw.folder %q does not match plane %q (want %q)", s.ID, s.BW.Folder, s.Plane, want)
	}
	return nil
}

// checkVarNames enforces that every exposed var is a valid env identifier (B5) and unique
// across the whole registry (B1): a var mapped by two secrets has no single source, and
// render's dedup was age-only, so run/show would silently resolve last-write.
//
// It only READS seenVar — registration is the caller's, so a rejected secret never
// reserves a name and cannot make the next valid secret look like the duplicate.
func checkVarNames(s *Secret, seenVar map[string]string) error {
	for _, v := range s.Vars() {
		if !validVarName.MatchString(v) {
			return fmt.Errorf("secret %q: invalid var name %q (want an env identifier [A-Za-z_][A-Za-z0-9_]*)", s.ID, v)
		}
		if first, dup := seenVar[v]; dup {
			return fmt.Errorf("var %q is exposed by both %q and %q — each var must map to exactly one secret", v, first, s.ID)
		}
	}
	return nil
}

// checkAgeBase rejects an age source name that is not a bare base name — anything
// with a path separator or ".." could escape sensitive/ when joined into
// <base>.secret.age (read-traversal). The charset matches the registry's real names
// (e.g. "openrouter.api.key", "id_ed25519").
func checkAgeBase(id, base string) error {
	if !validAgeBase.MatchString(base) || strings.Contains(base, "..") {
		return fmt.Errorf("secret %q: unsafe age source name %q (want a bare base name, no path/..)", id, base)
	}
	return nil
}

// checkBwName rejects a control character in a Bitwarden item/field name (it is
// passed to `bw get item` and echoed in errors). Spaces are allowed — bw item names
// are freeform.
func checkBwName(id, kind, name string) error {
	if strings.ContainsAny(name, "\n\r\t\x00") {
		return fmt.Errorf("secret %q: bw %s %q contains a control character", id, kind, name)
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
		return checkAgeBase(s.ID, s.Age)
	}
	for _, v := range s.Expose.Env.Vars {
		src := v.Age
		if src == "" {
			src = s.Age
		}
		if src == "" {
			return fmt.Errorf("secret %q: env %q has no age source", s.ID, v.Name)
		}
		if err := checkAgeBase(s.ID, src); err != nil {
			return err
		}
	}
	return nil
}

// checkFileAuthoritySources enforces the shape of a root. A file expose, because
// a key belongs on disk at a path with a mode — not in an environment variable
// where it would be inherited by every child of every process that read it. No
// age source, because declaring one names a ciphertext that cannot exist: this
// file is what decrypts ciphertexts, and encrypting it under itself is the
// chicken-and-egg #937 is about.
//
// A `bw:` block IS allowed and is the convenience copy. Authority stays on disk;
// the Bitwarden entry exists so a rebuilt machine can fetch it once, and so the
// two can be compared for drift (#1000). ADR-028 §2.
func (s *Secret) checkFileAuthoritySources() error {
	if s.Expose.File == nil {
		return fmt.Errorf("secret %q: file-authority exposes a file, never env vars", s.ID)
	}
	if s.Expose.File.Var == "" || s.Expose.File.Path == "" {
		return fmt.Errorf("secret %q: file expose needs var+path", s.ID)
	}
	if s.Age != "" {
		return fmt.Errorf("secret %q: file-authority takes no age source (got %q) — "+
			"the file IS the authority, and cannot be encrypted under itself", s.ID, s.Age)
	}
	return nil
}

// checkRecipient rejects a declared recipient on any backend that cannot act on it.
// Accepting it silently elsewhere would be the worse failure: a reader would take
// the key as pinned when nothing compares it, which is a declaration that lies.
func checkRecipient(s *Secret) error {
	if s.Recipient == "" {
		return nil
	}
	if s.Backend != BackendFileAuthority {
		return fmt.Errorf("secret %q: recipient is only meaningful on a file-authority secret, not %q — "+
			"nothing would compare it", s.ID, s.Backend)
	}
	if !strings.HasPrefix(s.Recipient, "age1") {
		return fmt.Errorf("secret %q: recipient %q is not an age public recipient (expected an age1... string)", s.ID, s.Recipient)
	}
	return nil
}

// checkBwSources verifies every exposed var/file has a Bitwarden item + field.
func (s *Secret) checkBwSources() error {
	if s.BW == nil || s.BW.Item == "" {
		return fmt.Errorf("secret %q: bw backend needs bw.item", s.ID)
	}
	if err := checkBwName(s.ID, "item", s.BW.Item); err != nil {
		return err
	}
	if err := checkBwName(s.ID, "field", s.BW.Field); err != nil {
		return err
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
		switch s.Backend {
		case BackendBW:
			es = append(es, s.bwEntries(home)...)
		case BackendFileAuthority:
			// Flattens identically to an age file secret — Var, Dest, Mode, and an
			// empty source — and keeps its own Backend tag, which is what routes it
			// to the resolver that refuses. Named explicitly rather than left to the
			// default: a future backend would otherwise inherit ageEntries' shape in
			// silence, which is how a consumer gets taught about one variant and
			// forgotten for another (REFACTOR-012).
			es = append(es, s.ageEntries(home)...)
		default:
			es = append(es, s.ageEntries(home)...)
		}
	}
	return es
}

// parseFileMode turns a FileExpose octal mode string ("0640") into an os.FileMode.
// "" → 0, which materialize reads as "apply the 0600 default". validate() has
// already rejected a non-octal string at parse, so a parse error here is impossible
// in practice; it degrades to 0 (the safe default) rather than panicking.
func parseFileMode(s string) os.FileMode {
	if s == "" {
		return 0
	}
	n, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0
	}
	return os.FileMode(n)
}

// ageEntries flattens an age/age-offline secret. The per-var age override falls
// back to the secret's top-level Age source.
func (s *Secret) ageEntries(home string) []Entry {
	if s.Expose.File != nil {
		return []Entry{{
			Var:       s.Expose.File.Var,
			Backend:   s.Backend,
			File:      s.Age,
			IsFile:    true,
			Dest:      expandHome(s.Expose.File.Path, home),
			Mode:      parseFileMode(s.Expose.File.Mode),
			Validate:  s.Validate,
			Recipient: s.Recipient,
		}}
	}
	es := make([]Entry, 0, len(s.Expose.Env.Vars))
	for _, v := range s.Expose.Env.Vars {
		src := v.Age
		if src == "" {
			src = s.Age
		}
		es = append(es, Entry{Var: v.Name, Backend: s.Backend, File: src, Validate: s.Validate})
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
			Var:      s.Expose.File.Var,
			Backend:  BackendBW,
			Item:     item,
			Field:    topField,
			IsFile:   true,
			Dest:     expandHome(s.Expose.File.Path, home),
			Mode:     parseFileMode(s.Expose.File.Mode),
			Validate: s.Validate,
		}}
	}
	es := make([]Entry, 0, len(s.Expose.Env.Vars))
	for _, v := range s.Expose.Env.Vars {
		field := v.Field
		if field == "" {
			field = topField
		}
		es = append(es, Entry{Var: v.Name, Backend: BackendBW, Item: item, Field: field, Validate: s.Validate})
	}
	return es
}

// BWTarget resolves the Bitwarden (item, field, isFile) that an exposed var of this
// secret writes to — the write-side counterpart of the bwEntries read flattening,
// consumed by `dotf secrets set`. With varName == "" it resolves the secret's sole
// target (an error if the secret is multi-var, forcing the caller to disambiguate). It
// requires a bw source; an age-only secret has no item to write to yet (migrate first).
func (s *Secret) BWTarget(varName string) (item, field string, isFile bool, err error) {
	if s.BW == nil || s.BW.Item == "" {
		return "", "", false, fmt.Errorf("secret %q has no bw source; `set` writes the Bitwarden backend (migrate it first)", s.ID)
	}
	item = s.BW.Item

	if s.Expose.File != nil {
		if varName != "" && varName != s.Expose.File.Var {
			return "", "", false, fmt.Errorf("secret %q exposes file var %q, not %q", s.ID, s.Expose.File.Var, varName)
		}
		return item, s.BW.Field, true, nil
	}

	fieldOf := func(v EnvVar) string {
		if v.Field != "" {
			return v.Field
		}
		return s.BW.Field
	}
	vars := s.Expose.Env.Vars
	if varName == "" {
		if len(vars) != 1 {
			return "", "", false, fmt.Errorf("secret %q exposes %d vars; name one: dotf secrets set %s <var> (vars: %s)", s.ID, len(vars), s.ID, strings.Join(s.Vars(), ", "))
		}
		return item, fieldOf(vars[0]), false, nil
	}
	for _, v := range vars {
		if v.Name == varName {
			return item, fieldOf(v), false, nil
		}
	}
	return "", "", false, fmt.Errorf("secret %q has no var %q (vars: %s)", s.ID, varName, strings.Join(s.Vars(), ", "))
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
