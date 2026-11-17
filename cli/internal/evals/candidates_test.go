package evals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodCandidates = `{"candidates":[
 {"id":"nan/glm5.3-flash","runner":"pi","provider":"nan","model":"glm5.3-flash","secret_id":"NAN_API_KEY","vendor":"zhipu"},
 {"id":"agy/gemini-4-hydrogen-high","runner":"agy","model":"gemini-4-hydrogen-high","auth":"login","vendor":"google"}
]}`

func writeCandidates(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, filepath.FromSlash(CandidatesFile))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestLoadCandidatesResolvesAnId(t *testing.T) {
	cs, err := LoadCandidates(writeCandidates(t, goodCandidates))
	if err != nil {
		t.Fatal(err)
	}
	e, err := cs.Resolve("agy/gemini-4-hydrogen-high")
	if err != nil || e.Model != "gemini-4-hydrogen-high" || e.Auth != "login" {
		t.Fatalf("Resolve = %+v, %v", e, err)
	}
	if _, err := cs.Resolve("nan/unknown"); err == nil || !strings.Contains(err.Error(), "nan/glm5.3-flash") {
		t.Errorf("an unknown id must be refused, listing the known ones: %v", err)
	}
}

// Each mutation breaks one property; an entry the launcher would run unsafely
// or ambiguously must not load.
func TestLoadCandidatesRefusesWhatTheLauncherCannotRunSafely(t *testing.T) {
	entry := `{"id":"nan/x","runner":"pi","provider":"nan","model":"x","secret_id":"NAN_API_KEY","vendor":"v"}`
	for _, tc := range []struct{ name, body, want string }{
		{"no secret and no login", `{"candidates":[{"id":"nan/x","runner":"pi","provider":"nan","model":"x","vendor":"v"}]}`, "secret_id"},
		{"both a secret and login", `{"candidates":[{"id":"nan/x","runner":"pi","provider":"nan","model":"x","secret_id":"K","auth":"login","vendor":"v"}]}`, "secret_id"},
		{"no model", `{"candidates":[{"id":"nan/x","runner":"pi","provider":"nan","model":"","secret_id":"K","vendor":"v"}]}`, "model"},
		{"pi without a provider", `{"candidates":[{"id":"nan/x","runner":"pi","model":"x","secret_id":"K","vendor":"v"}]}`, "provider"},
		{"an unknown runner", `{"candidates":[{"id":"nan/x","runner":"ollama","model":"x","secret_id":"K","vendor":"v"}]}`, "runner"},
		{"a duplicate id", `{"candidates":[` + entry + `,` + entry + `]}`, "twice"},
		{"an eval-prefixed id", `{"candidates":[{"id":"eval:x","runner":"pi","provider":"nan","model":"x","secret_id":"K","vendor":"v"}]}`, "eval:"},
		{"a pool-only field", `{"candidates":[{"id":"nan/x","runner":"pi","provider":"nan","model":"x","secret_id":"K","vendor":"v","signs":"first"}]}`, "signs"},
		{"no candidates", `{"candidates":[]}`, "no candidates"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadCandidates(writeCandidates(t, tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want a refusal naming %q, got %v", tc.want, err)
			}
		})
	}
	if _, err := LoadCandidates(t.TempDir()); err == nil {
		t.Error("a missing candidates file must fail loudly")
	}
}

func TestRepositoryCandidatesLoad(t *testing.T) {
	cs, err := LoadCandidates(repoFile(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Entries) == 0 {
		t.Fatal("harness/evals/candidates.json lists no candidate")
	}
}
