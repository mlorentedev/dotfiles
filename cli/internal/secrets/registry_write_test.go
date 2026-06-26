package secrets

import (
	"os"
	"strings"
	"testing"
)

const writeFixture = `version: 1
secrets:
  # ── apps ──────────────────────────────────
  - id: alpha                    # aligned trailing comment
    plane: app
    backend: age
    age: alpha.key
    expose: { env: ALPHA }
    rotate: 90d

  - id: beta
    plane: app
    backend: age
    age: beta.key
    expose: { env: BETA }
`

func TestSetBackendBW_FlipsOnlyTargetBlock(t *testing.T) {
	out, err := SetBackendBW([]byte(writeFixture), "alpha", "alpha-item", "password")
	if err != nil {
		t.Fatalf("SetBackendBW: %v", err)
	}
	got := string(out)

	// The alpha block flipped to bw, age dropped, bw inserted.
	if !strings.Contains(got, "backend: bw") {
		t.Errorf("alpha backend not flipped:\n%s", got)
	}
	if !strings.Contains(got, "bw: { item: alpha-item, field: password }") {
		t.Errorf("bw source not inserted:\n%s", got)
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
		"    backend: age\n    age: beta.key\n    expose: { env: BETA }\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("preservation: missing %q in:\n%s", want, got)
		}
	}
}

// The transformation is net-zero lines (backend value in place, -age, +bw), so every
// line outside the target block must keep its exact index and content.
func TestSetBackendBW_RealRegistry_OnlyTargetChanges(t *testing.T) {
	in, err := os.ReadFile("../../../secrets/registry.yaml")
	if err != nil {
		t.Skipf("registry.yaml not found: %v", err)
	}
	const id = "github-bitacora-pat"
	out, err := SetBackendBW(in, id, "github-bitacora", "password")
	if err != nil {
		t.Fatalf("SetBackendBW(%s): %v", id, err)
	}
	inLines := strings.Split(string(in), "\n")
	outLines := strings.Split(string(out), "\n")
	if len(inLines) != len(outLines) {
		t.Fatalf("line count changed: in=%d out=%d (transformation must be net-zero lines)", len(inLines), len(outLines))
	}
	start, end, _, err := secretBlock(inLines, id)
	if err != nil {
		t.Fatal(err)
	}
	for i := range inLines {
		if i >= start && i < end {
			continue // the only region allowed to change
		}
		if inLines[i] != outLines[i] {
			t.Errorf("line %d changed OUTSIDE the %s block:\n  in : %q\n  out: %q", i+1, id, inLines[i], outLines[i])
		}
	}
	// And the target block did flip.
	block := strings.Join(outLines[start:end], "\n")
	if !strings.Contains(block, "backend: bw") || !strings.Contains(block, "bw: { item: github-bitacora, field: password }") || strings.Contains(block, "age: github.bitacora") {
		t.Errorf("target block not correctly flipped:\n%s", block)
	}
}

func TestSetBackendBW_Idempotent(t *testing.T) {
	once, err := SetBackendBW([]byte(writeFixture), "alpha", "alpha-item", "password")
	if err != nil {
		t.Fatal(err)
	}
	twice, err := SetBackendBW(once, "alpha", "alpha-item", "password")
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
    expose: { file: { var: KUBECONFIG, path: "~/.kube/c" } }
`
	cases := map[string]struct {
		data, id, item, field string
	}{
		"unknown id":  {writeFixture, "nope", "i", "f"},
		"multi-var":   {multi, "multi", "i", "f"},
		"file secret": {file, "kube", "i", "f"},
		"empty item":  {writeFixture, "alpha", "", "f"},
		"empty field": {writeFixture, "alpha", "i", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := SetBackendBW([]byte(c.data), c.id, c.item, c.field); err == nil {
				t.Errorf("expected an error for %q", name)
			}
		})
	}
}
