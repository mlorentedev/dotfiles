package secrets

import (
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

func TestRegistry_Entries_FlattensExposeShapes(t *testing.T) {
	const home = "/home/u"
	// The expected flattened entries for sampleRegistry — one per env var, file
	// secrets carrying their ~-expanded Dest. (Previously cross-checked against
	// env-mapping.conf via ParseMapping; that twin was retired with #587.)
	want := []Entry{
		{Var: "NAN_API_KEY", File: "nan.api-key"},
		{Var: "GITHUB_PERSONAL_ACCESS_TOKEN", File: "github.token"},
		{Var: "RELEASE_TOKEN", File: "github.token"},
		{Var: "X_API_KEY", File: "x.api-key"},
		{Var: "X_BEARER_TOKEN", File: "x.bearer-token"},
		{Var: "KUBECONFIG", File: "kubelab.kubeconfig", IsFile: true, Dest: home + "/.kube/kubelab.config"},
		{Var: "SSH_KEY", File: "id_ed25519", IsFile: true, Dest: home + "/.ssh/id_ed25519"},
	}
	reg, err := ParseRegistry([]byte(sampleRegistry))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	got := reg.Entries(home)

	wm, gm := entriesByVar(want), entriesByVar(got)
	if len(wm) != len(gm) {
		t.Fatalf("entry count: registry=%d expected=%d (%v vs %v)", len(gm), len(wm), gm, wm)
	}
	for v, we := range wm {
		ge, ok := gm[v]
		if !ok {
			t.Errorf("registry missing var %s", v)
			continue
		}
		if ge.File != we.File || ge.IsFile != we.IsFile || ge.Dest != we.Dest {
			t.Errorf("var %s: registry=%+v expected=%+v", v, ge, we)
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

func TestRegistry_Entries_IncludesBwBackend(t *testing.T) {
	const yml = "version: 1\nsecrets:\n" +
		"  - {id: age-one, plane: app, backend: age, age: a.key, expose: {env: A_KEY}}\n" +
		"  - {id: bw-one, plane: app, backend: bw, bw: {item: bw-item, field: password}, expose: {env: B_KEY}}\n"
	reg, err := ParseRegistry([]byte(yml))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	got := entriesByVar(reg.Entries("/h"))
	if len(got) != 2 {
		t.Fatalf("Entries = %+v, want both age + bw entries", got)
	}
	if a := got["A_KEY"]; a.Backend != "age" || a.File != "a.key" {
		t.Errorf("A_KEY = %+v, want backend=age file=a.key", a)
	}
	if b := got["B_KEY"]; b.Backend != "bw" || b.Item != "bw-item" || b.Field != "password" {
		t.Errorf("B_KEY = %+v, want backend=bw item=bw-item field=password", b)
	}
}

func TestParseRegistry_BwShapes(t *testing.T) {
	const yml = `
version: 1
secrets:
  - id: single
    plane: app
    backend: bw
    bw: { item: single-item, field: password }
    expose: { env: SINGLE }
  - id: multi
    plane: app
    backend: bw
    bw: { item: multi-item }
    expose:
      env:
        A_KEY: { field: a-field }
        B_KEY: { field: b-field }
  - id: bwfile
    plane: infra
    backend: bw
    bw: { item: kube-item, field: notes }
    expose: { file: { var: KUBECONFIG, path: "~/.kube/c", mode: "0600" } }
`
	reg, err := ParseRegistry([]byte(yml))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	got := entriesByVar(reg.Entries("/home/u"))
	if e := got["SINGLE"]; e.Backend != "bw" || e.Item != "single-item" || e.Field != "password" {
		t.Errorf("SINGLE = %+v (single var inherits top-level field)", e)
	}
	if e := got["A_KEY"]; e.Item != "multi-item" || e.Field != "a-field" {
		t.Errorf("A_KEY = %+v (shared item, per-var field)", e)
	}
	if e := got["B_KEY"]; e.Item != "multi-item" || e.Field != "b-field" {
		t.Errorf("B_KEY = %+v", e)
	}
	if e := got["KUBECONFIG"]; !e.IsFile || e.Item != "kube-item" || e.Field != "notes" || e.Dest != "/home/u/.kube/c" {
		t.Errorf("KUBECONFIG = %+v (bw file secret, ~-expanded dest)", e)
	}
}

func TestParseRegistry_BwValidation(t *testing.T) {
	cases := map[string]string{
		"bw missing bw block":   "version: 1\nsecrets:\n  - {id: a, plane: app, backend: bw, expose: {env: A}}\n",
		"bw missing item":       "version: 1\nsecrets:\n  - {id: a, plane: app, backend: bw, bw: {field: password}, expose: {env: A}}\n",
		"bw env missing field":  "version: 1\nsecrets:\n  - {id: a, plane: app, backend: bw, bw: {item: it}, expose: {env: A}}\n",
		"bw file missing field": "version: 1\nsecrets:\n  - {id: a, plane: app, backend: bw, bw: {item: it}, expose: {file: {var: V, path: /p}}}\n",
	}
	for name, yml := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseRegistry([]byte(yml)); err == nil {
				t.Errorf("expected error for %q, got nil", name)
			}
		})
	}
}

// TestBWSource_Folder proves the registry parses bw.folder (OPS-028: ADR-028 ratified
// a folder taxonomy the schema couldn't express) and rejects anything outside the
// ratified set (Dotfiles/apps, Dotfiles/infra) — floor and personal deliberately have
// no folder yet (#586).
func TestBWSource_Folder(t *testing.T) {
	const yml = "version: 1\nsecrets:\n" +
		"  - {id: a, plane: app, backend: bw, bw: {item: it, field: password, folder: Dotfiles/apps}, expose: {env: A}}\n"
	reg, err := ParseRegistry([]byte(yml))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	s := reg.Lookup("a")
	if s == nil || s.BW.Folder != "Dotfiles/apps" {
		t.Fatalf("secret = %+v, want BW.Folder = Dotfiles/apps", s)
	}
}

func TestBWSource_Folder_NoneDeclared(t *testing.T) {
	const yml = "version: 1\nsecrets:\n" +
		"  - {id: a, plane: app, backend: bw, bw: {item: it, field: password}, expose: {env: A}}\n"
	reg, err := ParseRegistry([]byte(yml))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	if s := reg.Lookup("a"); s == nil || s.BW.Folder != "" {
		t.Fatalf("secret = %+v, want empty Folder (unfoldered, today's default)", s)
	}
}

func TestParseRegistry_BwFolder_RejectsUnratified(t *testing.T) {
	cases := map[string]string{
		"lowercase (casing must match the ADR, not #951's own transcription typo)": "dotfiles/apps",
		"unratified folder":   "Dotfiles/personal",
		"arbitrary free text": "My Random Folder",
	}
	for name, folder := range cases {
		t.Run(name, func(t *testing.T) {
			yml := "version: 1\nsecrets:\n" +
				"  - {id: a, plane: app, backend: bw, bw: {item: it, field: password, folder: \"" + folder + "\"}, expose: {env: A}}\n"
			if _, err := ParseRegistry([]byte(yml)); err == nil {
				t.Errorf("folder %q: expected a validation error, got nil", folder)
			}
		})
	}
}

func TestParseRegistry_Validation(t *testing.T) {
	cases := map[string]string{
		"bad version":        "version: 2\nsecrets: []\n",
		"duplicate id":       "version: 1\nsecrets:\n  - {id: a, plane: app, backend: age, age: f, expose: {env: A}}\n  - {id: a, plane: app, backend: age, age: g, expose: {env: B}}\n",
		"unknown backend":    "version: 1\nsecrets:\n  - {id: a, plane: app, backend: vault, age: f, expose: {env: A}}\n",
		"both env and file":  "version: 1\nsecrets:\n  - {id: a, plane: app, backend: age, age: f, expose: {env: A, file: {var: V, path: /p}}}\n",
		"missing source":     "version: 1\nsecrets:\n  - {id: a, plane: app, backend: age, expose: {env: A}}\n",
		"invalid file mode":  "version: 1\nsecrets:\n  - {id: a, plane: app, backend: age, age: f, expose: {file: {var: V, path: /p, mode: \"x\"}}}\n",
		"non-octal digit":    "version: 1\nsecrets:\n  - {id: a, plane: app, backend: age, age: f, expose: {file: {var: V, path: /p, mode: \"99\"}}}\n",
		"age path traversal": "version: 1\nsecrets:\n  - {id: a, plane: app, backend: age, age: ../../etc/passwd, expose: {env: A}}\n",
		"age source slash":   "version: 1\nsecrets:\n  - {id: a, plane: app, backend: age, age: sub/dir, expose: {env: A}}\n",
		"invalid var name":   "version: 1\nsecrets:\n  - {id: a, plane: app, backend: age, age: f, expose: {env: 1BAD}}\n",
	}
	for name, yml := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseRegistry([]byte(yml)); err == nil {
				t.Errorf("expected error for %q, got nil", name)
			}
		})
	}
}
