package secrets

import "testing"

func targetRegistry(t *testing.T) *Registry {
	t.Helper()
	reg, err := ParseRegistry([]byte(`
version: 1
secrets:
  - {id: tok, plane: app, backend: bw, bw: {item: it, field: password}, expose: {env: TOK}}
  - {id: multi, plane: app, backend: bw, bw: {item: mi}, expose: {env: {A: {field: fa}, B: {field: fb}}}}
  - {id: file-sec, plane: infra, backend: bw, bw: {item: fi, field: notes}, expose: {file: {var: KC, path: "~/.k"}}}
  - {id: age-only, plane: app, backend: age, age: a, expose: {env: AGEV}}
`))
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestBWTarget(t *testing.T) {
	reg := targetRegistry(t)

	if i, f, isFile, err := reg.Lookup("tok").BWTarget(""); err != nil || i != "it" || f != "password" || isFile {
		t.Errorf("single env, no arg = (%q,%q,%v,%v), want (it,password,false,nil)", i, f, isFile, err)
	}
	if i, f, _, err := reg.Lookup("multi").BWTarget("B"); err != nil || i != "mi" || f != "fb" {
		t.Errorf("multi env, var B = (%q,%q,%v), want (mi,fb,nil)", i, f, err)
	}
	if _, _, _, err := reg.Lookup("multi").BWTarget(""); err == nil {
		t.Error("multi-var secret with no var must error (forces disambiguation)")
	}
	if i, f, isFile, err := reg.Lookup("file-sec").BWTarget(""); err != nil || i != "fi" || f != "notes" || !isFile {
		t.Errorf("file secret = (%q,%q,%v,%v), want (fi,notes,true,nil)", i, f, isFile, err)
	}
	if _, _, _, err := reg.Lookup("age-only").BWTarget(""); err == nil {
		t.Error("age-only secret must error (no bw source to write to)")
	}
	if _, _, _, err := reg.Lookup("multi").BWTarget("NOPE"); err == nil {
		t.Error("unknown var must error")
	}
}
