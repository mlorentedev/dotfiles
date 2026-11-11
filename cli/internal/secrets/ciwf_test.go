package secrets

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestWorkflowReads(t *testing.T) {
	wf := []byte(`on: push
jobs:
  a:
    runs-on: ubuntu-latest
    env:
      A: ${{ secrets.ALPHA }}
      B: ${{ secrets['BRAVO'] }}
      C: ${{ secrets["CHARLIE"] }}
      T: ${{ secrets.GITHUB_TOKEN }}
      D: ${{ secrets.ALPHA }}
      E: ${{ inputs.secrets }}
  call:
    uses: owner/ci/.github/workflows/w.yml@abc
    secrets:
      THEIR_NAME: ${{ secrets.MINE }}
`)
	names, unknown := workflowReads(wf)
	want := []string{"ALPHA", "BRAVO", "CHARLIE", "MINE"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	if len(unknown) != 0 {
		t.Fatalf("unknown = %v, want none", unknown)
	}
}

func TestWorkflowReadsUnknown(t *testing.T) {
	wf := []byte(`jobs:
  a:
    env:
      ALL: ${{ toJSON(secrets) }}
      ONE: ${{ secrets[matrix.key] }}
      OK: ${{ secrets.FINE }}
`)
	names, unknown := workflowReads(wf)
	if !reflect.DeepEqual(names, []string{"FINE"}) {
		t.Fatalf("names = %v, want [FINE]", names)
	}
	want := []string{"secrets[matrix.key]", "toJSON(secrets)"}
	if !reflect.DeepEqual(unknown, want) {
		t.Fatalf("unknown = %v, want %v", unknown, want)
	}
}

func TestInheritedCalls(t *testing.T) {
	wf := []byte(`on: pull_request
jobs:
  review:
    uses: owner/ci-workflows/.github/workflows/pr-agent.yml@0123abc
    secrets: inherit
  mapped:
    uses: owner/ci-workflows/.github/workflows/release.yml@v1
    secrets:
      RELEASE_TOKEN: ${{ secrets.MINE }}
  local:
    uses: ./.github/workflows/local.yml
    secrets: inherit
  plain:
    runs-on: ubuntu-latest
`)
	got, err := inheritedCalls(wf)
	if err != nil {
		t.Fatal(err)
	}
	want := []workflowRef{{Repo: "owner/ci-workflows", Path: ".github/workflows/pr-agent.yml", Ref: "0123abc"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %+v, want %+v", got, want)
	}
}

func TestCalleeSecrets(t *testing.T) {
	// A bare `on:` is the YAML 1.1 boolean true; the parser must still reach
	// on.workflow_call.secrets, or every inherited secret disappears silently.
	wf := []byte(`name: pr-agent
on:
  workflow_call:
    secrets:
      NAN_API_KEY:
        required: true
      PR_AGENT_ANTHROPIC_API_KEY:
        required: false
      NO_FLAG: {}
`)
	got, err := calleeSecrets(wf)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"NAN_API_KEY": true, "PR_AGENT_ANTHROPIC_API_KEY": false, "NO_FLAG": false}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("callee secrets = %v, want %v", got, want)
	}
}

type fakeWorkflows struct {
	files map[string]map[string][]byte // repo -> path -> content, at the default ref
	refs  map[string][]byte            // "repo/path@ref" -> content
	reads int
}

func (f *fakeWorkflows) List(repo string) ([]string, error) {
	var out []string
	for p := range f.files[repo] {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeWorkflows) Read(repo, path, ref string) ([]byte, error) {
	f.reads++
	if ref == "" {
		return f.files[repo][path], nil
	}
	b, ok := f.refs[repo+"/"+path+"@"+ref]
	if !ok {
		return nil, errors.New("not found")
	}
	return b, nil
}

func TestCheckRepo(t *testing.T) {
	callee := []byte(`on:
  workflow_call:
    secrets:
      NAN_API_KEY:
        required: true
      PR_AGENT_ANTHROPIC_API_KEY:
        required: false
`)
	caller := []byte(`jobs:
  review:
    uses: o/ci/.github/workflows/pr-agent.yml@abc
    secrets: inherit
`)
	fw := &fakeWorkflows{
		files: map[string]map[string][]byte{"o/app": {
			".github/workflows/review.yml": caller,
			".github/workflows/again.yml":  caller,
			".github/workflows/deploy.yml": []byte("jobs:\n  d:\n    env:\n      K: ${{ secrets.DEPLOY_KEY }}\n      J: ${{ toJSON(secrets) }}\n"),
		}},
		refs: map[string][]byte{"o/ci/.github/workflows/pr-agent.yml@abc": callee},
	}
	reg := &Registry{Secrets: []Secret{
		{ID: "nan", Consumers: []string{"ci:o/app"}, Expose: Expose{Env: EnvExpose{Vars: []EnvVar{{Name: "NAN_API_KEY"}}}}},
		{ID: "stale", Consumers: []string{"ci:o/app"}, Expose: Expose{Env: EnvExpose{Vars: []EnvVar{{Name: "OLD_TOKEN"}}}}},
	}}
	c := NewCIChecker(fw, reg)
	got, err := c.CheckRepo("o/app")
	if err != nil {
		t.Fatal(err)
	}
	wantMissing := []CIMissing{
		{Name: "DEPLOY_KEY"},
		{Name: "PR_AGENT_ANTHROPIC_API_KEY", OptionalIn: "o/ci/.github/workflows/pr-agent.yml@abc"},
	}
	if !reflect.DeepEqual(got.Missing, wantMissing) {
		t.Errorf("missing = %+v, want %+v", got.Missing, wantMissing)
	}
	if !reflect.DeepEqual(got.Excess, []string{"OLD_TOKEN"}) {
		t.Errorf("excess = %v, want [OLD_TOKEN]", got.Excess)
	}
	if len(got.Unknown) != 1 || got.Unknown[0].File != ".github/workflows/deploy.yml" || got.Unknown[0].Expr != "toJSON(secrets)" {
		t.Errorf("unknown = %+v", got.Unknown)
	}
	// Three listed files plus one callee: the callee is read once, not per caller.
	if fw.reads != 4 {
		t.Errorf("reads = %d, want 4 (callee cached)", fw.reads)
	}
	if !got.Failing() {
		t.Error("a repository with missing or excess names must fail")
	}
}

func TestCheckRepoCleanAndUnknownOnly(t *testing.T) {
	fw := &fakeWorkflows{files: map[string]map[string][]byte{"o/a": {
		".github/workflows/x.yml": []byte("jobs:\n  a:\n    env:\n      K: ${{ secrets.K }}\n      A: ${{ toJSON(secrets) }}\n"),
	}}}
	reg := &Registry{Secrets: []Secret{{ID: "k", Consumers: []string{"ci:o/a"}, Expose: Expose{Env: EnvExpose{Vars: []EnvVar{{Name: "K"}}}}}}}
	got, err := NewCIChecker(fw, reg).CheckRepo("o/a")
	if err != nil {
		t.Fatal(err)
	}
	if got.Failing() {
		t.Errorf("unknown alone is reported, not failed: %+v", got)
	}
	if len(got.Unknown) != 1 {
		t.Errorf("unknown = %+v, want one", got.Unknown)
	}
}

func TestCheckRepoCalleeUnreadable(t *testing.T) {
	fw := &fakeWorkflows{files: map[string]map[string][]byte{"o/a": {
		".github/workflows/x.yml": []byte("jobs:\n  r:\n    uses: o/ci/.github/workflows/gone.yml@v9\n    secrets: inherit\n"),
	}}}
	_, err := NewCIChecker(fw, &Registry{}).CheckRepo("o/a")
	if err == nil || !strings.Contains(err.Error(), "gone.yml@v9") {
		t.Fatalf("an unreadable callee must fail loudly, naming it; got %v", err)
	}
}
