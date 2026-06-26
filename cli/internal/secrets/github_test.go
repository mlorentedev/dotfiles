package secrets

import (
	"slices"
	"testing"
)

// fakeSetter records uploads ("repo|name" -> value) so tests assert with no gh/network.
type fakeSetter map[string]string

func (f fakeSetter) SetSecret(repo, name, value string) error {
	f[repo+"|"+name] = value
	return nil
}

func TestGitHubSecretSetter_FakeRoundTrip(t *testing.T) {
	var s GitHubSecretSetter = fakeSetter{}
	if err := s.SetSecret("o/r", "TOK", "v"); err != nil {
		t.Fatal(err)
	}
	if s.(fakeSetter)["o/r|TOK"] != "v" {
		t.Error("fake setter did not record the upload")
	}
}

// selectRegistry mixes backends, repos, and every exclusion class.
const selectRegistry = `
version: 1
secrets:
  - {id: rel, plane: app, backend: age, age: rel.src, expose: {env: RELEASE_TOKEN}, consumers: ["ci:mlorentedev/dotfiles"]}
  - {id: bw1, plane: app, backend: bw, bw: {item: it, field: f}, expose: {env: BW_TOKEN}, consumers: ["ci:mlorentedev/dotfiles"]}
  - {id: other, plane: app, backend: age, age: other.src, expose: {env: OTHER_TOKEN}, consumers: ["ci:mlorentedev/payments"]}
  - {id: kc, plane: infra, backend: age, age: kc.src, bw: {item: kc, field: notes}, expose: {file: {var: KUBECONFIG, path: "~/.k"}}, consumers: ["ci:mlorentedev/dotfiles"]}
  - {id: floor, plane: floor, backend: age-offline, age: floor.src, expose: {env: FLOOR_TOKEN}, consumers: ["ci:mlorentedev/dotfiles"]}
  - {id: gh, plane: app, backend: age, age: gh.src, expose: {env: GITHUB_PAT}, consumers: ["ci:mlorentedev/dotfiles"]}
`

func TestSelectCI(t *testing.T) {
	reg, err := ParseRegistry([]byte(selectRegistry))
	if err != nil {
		t.Fatal(err)
	}
	sel := reg.SelectCI("mlorentedev/dotfiles")

	// Upload: the age + bw entries for this repo, flattened identically (backend-agnostic).
	var uploaded []string
	for _, e := range sel.Upload {
		uploaded = append(uploaded, e.Var)
	}
	slices.Sort(uploaded)
	want := []string{"BW_TOKEN", "RELEASE_TOKEN"}
	if !slices.Equal(uploaded, want) {
		t.Errorf("upload vars = %v, want %v (other-repo + file + floor + GITHUB_* excluded)", uploaded, want)
	}

	// Each exclusion class is reported (never silent).
	skips := map[string]string{}
	for _, sk := range sel.Skipped {
		skips[sk.ID] = sk.Reason
	}
	for _, id := range []string{"kc", "floor", "GITHUB_PAT"} {
		if _, ok := skips[id]; !ok {
			t.Errorf("expected %q to be reported as skipped; skips=%v", id, skips)
		}
	}
	// The other-repo secret is simply not selected (not a skip of this repo's set).
	if _, ok := skips["other"]; ok {
		t.Error("other-repo secret must not appear in this repo's skip list")
	}
}

func TestSelectCI_NoMatch(t *testing.T) {
	reg, err := ParseRegistry([]byte(selectRegistry))
	if err != nil {
		t.Fatal(err)
	}
	sel := reg.SelectCI("mlorentedev/nonexistent")
	if len(sel.Upload) != 0 || len(sel.Skipped) != 0 {
		t.Errorf("no-match repo must select nothing: upload=%v skipped=%v", sel.Upload, sel.Skipped)
	}
}
