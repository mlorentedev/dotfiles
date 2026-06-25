package secrets

import (
	"strings"
	"testing"
)

// sampleRegistry exercises every expose shape and backend the schema supports.
const sampleRegistry = `
version: 1
secrets:
  - id: nan-api-key
    plane: app
    backend: age
    age: nan.api-key
    expose: { env: NAN_API_KEY }
    consumers: ["agent:opencode"]
    rotate: 90d
  - id: github-token
    plane: app
    backend: age
    age: github.token
    expose: { env: [GITHUB_PERSONAL_ACCESS_TOKEN, RELEASE_TOKEN] }
    consumers: ["ci:release", local]
  - id: x-twitter
    plane: app
    backend: age
    expose:
      env:
        X_API_KEY:      { age: x.api-key }
        X_BEARER_TOKEN: { age: x.bearer-token }
  - id: kubelab-kubeconfig
    plane: infra
    backend: age
    age: kubelab.kubeconfig
    expose: { file: { var: KUBECONFIG, path: "~/.kube/kubelab.config", mode: "0600" } }
  - id: ssh-id-ed25519
    plane: floor
    backend: age-offline
    age: id_ed25519
    expose: { file: { var: SSH_KEY, path: "~/.ssh/id_ed25519", mode: "0600" } }
`

func TestParseRegistry_Shapes(t *testing.T) {
	reg, err := ParseRegistry([]byte(sampleRegistry))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	if reg.Version != 1 {
		t.Fatalf("version = %d, want 1", reg.Version)
	}
	if len(reg.Secrets) != 5 {
		t.Fatalf("got %d secrets, want 5", len(reg.Secrets))
	}
	if s := reg.Secrets[0]; s.ID != "nan-api-key" || s.Plane != "app" || s.Backend != "age" {
		t.Errorf("secret[0] = %+v", s)
	}
}

// entriesByVar indexes entries for set comparison (order-independent).
func entriesByVar(es []Entry) map[string]Entry {
	m := make(map[string]Entry, len(es))
	for _, e := range es {
		m[e.Var] = e
	}
	return m
}

func TestRegistry_Entries_RoundTripWithMapping(t *testing.T) {
	const home = "/home/u"
	const mapping = "" +
		"NAN_API_KEY=nan.api-key\n" +
		"GITHUB_PERSONAL_ACCESS_TOKEN=github.token\n" +
		"RELEASE_TOKEN=github.token\n" +
		"X_API_KEY=x.api-key\n" +
		"X_BEARER_TOKEN=x.bearer-token\n" +
		"@KUBECONFIG=kubelab.kubeconfig>~/.kube/kubelab.config\n" +
		"@SSH_KEY=id_ed25519>~/.ssh/id_ed25519\n"

	want, err := ParseMapping(strings.NewReader(mapping), home)
	if err != nil {
		t.Fatalf("ParseMapping: %v", err)
	}
	reg, err := ParseRegistry([]byte(sampleRegistry))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	got := reg.Entries(home)

	wm, gm := entriesByVar(want), entriesByVar(got)
	if len(wm) != len(gm) {
		t.Fatalf("entry count: registry=%d mapping=%d (%v vs %v)", len(gm), len(wm), gm, wm)
	}
	for v, we := range wm {
		ge, ok := gm[v]
		if !ok {
			t.Errorf("registry missing var %s", v)
			continue
		}
		if ge.File != we.File || ge.IsFile != we.IsFile || ge.Dest != we.Dest {
			t.Errorf("var %s: registry=%+v mapping=%+v", v, ge, we)
		}
	}
}

func TestRegistry_Lookup_IdThenVar(t *testing.T) {
	reg, err := ParseRegistry([]byte(sampleRegistry))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	// by id
	if s := reg.Lookup("github-token"); s == nil || s.ID != "github-token" {
		t.Errorf("Lookup(id) = %v", s)
	}
	// by env var name → the owning secret
	if s := reg.Lookup("RELEASE_TOKEN"); s == nil || s.ID != "github-token" {
		t.Errorf("Lookup(var) = %v", s)
	}
	if s := reg.Lookup("X_BEARER_TOKEN"); s == nil || s.ID != "x-twitter" {
		t.Errorf("Lookup(map var) = %v", s)
	}
	if s := reg.Lookup("nope"); s != nil {
		t.Errorf("Lookup(unknown) = %v, want nil", s)
	}
}

func TestParseRegistry_Validation(t *testing.T) {
	cases := map[string]string{
		"bad version":      "version: 2\nsecrets: []\n",
		"duplicate id":     "version: 1\nsecrets:\n  - {id: a, plane: app, backend: age, age: f, expose: {env: A}}\n  - {id: a, plane: app, backend: age, age: g, expose: {env: B}}\n",
		"unknown backend":  "version: 1\nsecrets:\n  - {id: a, plane: app, backend: vault, age: f, expose: {env: A}}\n",
		"both env and file": "version: 1\nsecrets:\n  - {id: a, plane: app, backend: age, age: f, expose: {env: A, file: {var: V, path: /p}}}\n",
		"missing source":   "version: 1\nsecrets:\n  - {id: a, plane: app, backend: age, expose: {env: A}}\n",
	}
	for name, yml := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseRegistry([]byte(yml)); err == nil {
				t.Errorf("expected error for %q, got nil", name)
			}
		})
	}
}
