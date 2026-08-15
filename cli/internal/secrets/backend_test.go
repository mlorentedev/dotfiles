package secrets

import (
	"strings"
	"testing"
)

// The registry's parser and the Loader's resolver map are two independent
// spellings of "which backends exist". REFACTOR-012 is the class of bug that
// gap produces: a consumer taught about one variant and forgotten for another,
// failing silently because the missing tag reads as Go's zero value rather than
// as an error. This binds the two spellings together — a backend added to
// ValidBackends without a resolver fails here instead of on a user's machine.
func TestResolversCoverEveryValidBackend(t *testing.T) {
	got := (&Loader{}).resolvers()

	for _, b := range ValidBackends() {
		if _, ok := got[b]; !ok {
			t.Errorf("backend %q is accepted by the registry but has no resolver — "+
				"EnvFor would fail with %q at runtime", b, "unknown backend")
		}
	}

	// "" is deliberately NOT a valid registry backend but IS resolvable: a
	// hand-built Entry{Var, File} from a pre-bw caller carries no tag and must
	// still resolve as age. Pinned so a tightening of resolvers() cannot drop
	// the back-compat entry unnoticed.
	if _, ok := got[BackendDefault]; !ok {
		t.Error("the empty backend tag must resolve (as age) for pre-bw callers")
	}
	if _, isAge := got[BackendDefault].(ageResolver); !isAge {
		t.Errorf("the empty backend tag must resolve as age, got %T", got[BackendDefault])
	}
}

// The parser must accept exactly the canonical list and nothing else, so
// ValidBackends is the single answer to "what may a registry declare" rather
// than a list that drifts from the switch it was extracted from.
func TestParseRegistry_AcceptsExactlyValidBackends(t *testing.T) {
	tmpl := `
version: 1
secrets:
  - id: s
    plane: app
    backend: %s
    age: some.src
    bw: { item: some-item, field: api-token }
    expose: { env: SOME_VAR }
`
	for _, b := range ValidBackends() {
		if _, err := ParseRegistry([]byte(strings.Replace(tmpl, "%s", b, 1))); err != nil {
			t.Errorf("backend %q is in ValidBackends but the parser rejects it: %v", b, err)
		}
	}
	if _, err := ParseRegistry([]byte(strings.Replace(tmpl, "%s", "vault", 1))); err == nil {
		t.Error("parser accepted a backend outside ValidBackends")
	}
}

// SourceID is the union's own identity: two entries are the same underlying
// secret only if their backend AND that backend's source fields agree. Before
// REFACTOR-012 consumers compared Entry.File alone, which is "" for every bw
// entry — so all bw secrets compared equal to each other.
func TestEntrySourceID_DistinguishesBackendSources(t *testing.T) {
	cases := []struct {
		name  string
		a, b  Entry
		equal bool
	}{
		{
			name:  "same age source",
			a:     Entry{Backend: BackendAge, File: "github.token"},
			b:     Entry{Backend: BackendAge, File: "github.token"},
			equal: true,
		},
		{
			name:  "different age sources",
			a:     Entry{Backend: BackendAge, File: "github.token"},
			b:     Entry{Backend: BackendAge, File: "nan.api-key"},
			equal: false,
		},
		{
			// The regression this type exists for: distinct bw secrets both
			// carry File == "" and used to compare equal.
			name:  "different bw items",
			a:     Entry{Backend: BackendBW, Item: "github-cli-pat", Field: "api-token"},
			b:     Entry{Backend: BackendBW, Item: "github-bitacora-pat", Field: "api-token"},
			equal: false,
		},
		{
			name:  "same bw item, different field",
			a:     Entry{Backend: BackendBW, Item: "dockerhub", Field: "password"},
			b:     Entry{Backend: BackendBW, Item: "dockerhub", Field: "username"},
			equal: false,
		},
		{
			name:  "same bw item and field",
			a:     Entry{Backend: BackendBW, Item: "dockerhub", Field: "password"},
			b:     Entry{Backend: BackendBW, Item: "dockerhub", Field: "password"},
			equal: true,
		},
		{
			// A bw entry and an age entry are never the same source, even
			// though both have an empty counterpart field.
			name:  "across backends",
			a:     Entry{Backend: BackendAge, File: ""},
			b:     Entry{Backend: BackendBW, Item: "", Field: ""},
			equal: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.a.SourceID() == tc.b.SourceID(); got != tc.equal {
				t.Errorf("SourceID equality = %v, want %v (%q vs %q)",
					got, tc.equal, tc.a.SourceID(), tc.b.SourceID())
			}
		})
	}
}

// envSourceMap's duplicate-var guard is defense-in-depth: ParseRegistry rejects a
// duplicate var for every backend (#612 B1), so this path is only reached by a
// Registry built in code. That is exactly why it must work — it is the net under
// the parser, and a net with a bw-shaped hole is not one. Before REFACTOR-012 the
// guard compared File, "" for both bw entries, so it never fired.
func TestEnvSourceMap_RejectsTwoBwSecretsExposingOneVar(t *testing.T) {
	reg := &Registry{
		Version: 1,
		Secrets: []Secret{
			{
				ID:      "a",
				Plane:   "app",
				Backend: BackendBW,
				BW:      &BWSource{Item: "item-a", Field: "api-token"},
				Expose:  Expose{Env: EnvExpose{Vars: []EnvVar{{Name: "SHARED"}}}},
			},
			{
				ID:      "b",
				Plane:   "app",
				Backend: BackendBW,
				BW:      &BWSource{Item: "item-b", Field: "api-token"},
				Expose:  Expose{Env: EnvExpose{Vars: []EnvVar{{Name: "SHARED"}}}},
			},
		},
	}

	_, err := envSourceMap(reg, "/home/test")
	if err == nil {
		t.Fatal("envSourceMap accepted SHARED exposed by two distinct bw secrets — " +
			"the non-deterministic-registry guard did not fire")
	}
	// The message must name both sources; "(, )" is the pre-fix output and is
	// useless to whoever has to fix the registry.
	for _, want := range []string{"item-a", "item-b", "SHARED"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must name %q so the registry is fixable, got: %v", want, err)
		}
	}
}
