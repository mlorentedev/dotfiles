package secrets

import (
	"os"
	"strings"
	"testing"
)

// Fixtures mirror the env-var-per-entry model: every migratable entry PRE-DECLARES its
// `bw:` target (dormant while backend is age). SetBackendBW only activates it.
const writeFixture = `version: 1
secrets:
  # ── apps ──────────────────────────────────
  - id: alpha                    # aligned trailing comment
    plane: app
    backend: age
    age: alpha.key
    bw: { item: alpha-item, field: password }
    expose: { env: ALPHA }
    rotate: 90d

  - id: beta
    plane: app
    backend: age
    age: beta.key
    bw: { item: beta-item, field: api-key }
    expose: { env: BETA }
`

func TestSetBackendBW_FlipsOnlyTargetBlock(t *testing.T) {
	out, err := SetBackendBW([]byte(writeFixture), "alpha")
	if err != nil {
		t.Fatalf("SetBackendBW: %v", err)
	}
	got := string(out)

	// The alpha block flipped to bw, age dropped, the declared bw kept verbatim in place.
	if !strings.Contains(got, "backend: bw") {
		t.Errorf("alpha backend not flipped:\n%s", got)
	}
	if !strings.Contains(got, "bw: { item: alpha-item, field: password }") {
		t.Errorf("declared bw not preserved in place:\n%s", got)
	}
	if strings.Contains(got, "age: alpha.key") {
		t.Errorf("alpha age source not dropped:\n%s", got)
	}
	// Everything else is byte-for-byte: the aligned comment, the blank line, and the
	// whole beta block survive verbatim.
	for _, want := range []string{
		"  # ── apps ──────────────────────────────────",
		"  - id: alpha                    # aligned trailing comment",
		"\n\n  - id: beta\n",
		"    backend: age\n    age: beta.key\n    bw: { item: beta-item, field: api-key }\n    expose: { env: BETA }\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("preservation: missing %q in:\n%s", want, got)
		}
	}
}

// The activation drops exactly one line (the dead age: source) and flips backend in
// place, so the registry is one line shorter and every line OUTSIDE the target block is
// byte-identical — content-preserved, just shifted past the dropped line. This asserts
// that content invariant against the real registry (golden), not stable indices.
func TestSetBackendBW_RealRegistry_OnlyTargetChanges(t *testing.T) {
	in, err := os.ReadFile("../../../secrets/registry.yaml")
	if err != nil {
		t.Skipf("registry.yaml not found: %v", err)
	}
	const id = "BITACORA_PAT"
	out, err := SetBackendBW(in, id)
	if err != nil {
		t.Fatalf("SetBackendBW(%s): %v", id, err)
	}
	inLines := strings.Split(string(in), "\n")
	outLines := strings.Split(string(out), "\n")
	if len(outLines) != len(inLines)-1 {
		t.Fatalf("expected exactly one line dropped (the age: source): in=%d out=%d", len(inLines), len(outLines))
	}
	start, end, _, err := secretBlock(inLines, id)
	if err != nil {
		t.Fatal(err)
	}
	// Everything BEFORE the block keeps its exact index and content.
	for i := 0; i < start; i++ {
		if inLines[i] != outLines[i] {
			t.Errorf("line %d changed BEFORE the %s block:\n  in : %q\n  out: %q", i+1, id, inLines[i], outLines[i])
		}
	}
	// Everything AFTER the block is byte-identical, shifted up by the one dropped line.
	for i := end; i < len(inLines); i++ {
		if inLines[i] != outLines[i-1] {
			t.Errorf("line %d changed AFTER the %s block:\n  in : %q\n  out: %q", i+1, id, inLines[i], outLines[i-1])
		}
	}
	// And the target block flipped: backend bw, age gone, declared bw kept verbatim.
	block := strings.Join(outLines[start:end-1], "\n")
	if !strings.Contains(block, "backend: bw") {
		t.Errorf("backend not flipped:\n%s", block)
	}
	if strings.Contains(block, "age: github.bitacora") {
		t.Errorf("age source not dropped:\n%s", block)
	}
	if !strings.Contains(block, "bw: { item: github-bitacora-pat, field: api-token }") {
		t.Errorf("declared bw block not preserved in place:\n%s", block)
	}
}

func TestSetBackendBW_Idempotent(t *testing.T) {
	once, err := SetBackendBW([]byte(writeFixture), "alpha")
	if err != nil {
		t.Fatal(err)
	}
	twice, err := SetBackendBW(once, "alpha")
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if string(once) != string(twice) {
		t.Errorf("not idempotent:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

func TestSetBackendBW_Guards(t *testing.T) {
	multi := `version: 1
secrets:
  - id: multi
    plane: app
    backend: age
    expose:
      env:
        A: { age: a.k }
        B: { age: b.k }
`
	file := `version: 1
secrets:
  - id: kube
    plane: infra
    backend: age
    age: kube.cfg
    bw: { item: kube, field: notes }
    expose: { file: { var: KUBECONFIG, path: "~/.kube/c" } }
`
	// Single scalar env, but no bw: target declared — nothing to activate.
	nobw := `version: 1
secrets:
  - id: orphan
    plane: app
    backend: age
    age: orphan.key
    expose: { env: ORPHAN }
`
	cases := map[string]struct {
		data, id string
	}{
		"unknown id":   {writeFixture, "nope"},
		"multi-var":    {multi, "multi"},
		"file secret":  {file, "kube"},
		"no bw target": {nobw, "orphan"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := SetBackendBW([]byte(c.data), c.id); err == nil {
				t.Errorf("expected an error for %q", name)
			}
		})
	}
}
