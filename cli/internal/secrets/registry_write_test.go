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
	// The target is DISCOVERED, not named: any age-backed entry with a bw: block that
	// SetBackendBW accepts. A hardcoded id pinned GITHUB_PERSONAL_ACCESS_TOKEN "until
	// C9", and the day that entry migrated (CLI-080, by reconcile rather than
	// `migrate --split`) the test broke for a reason that had nothing to do with what
	// it asserts. Every future migration would have done the same.
	id := migratableAgeEntry(t, in)
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
	// The age line and the bw: line are read from the INPUT block, so the assertion
	// follows whichever entry was discovered.
	for _, ln := range inLines[start:end] {
		switch trimmed := strings.TrimSpace(ln); {
		case strings.HasPrefix(trimmed, "age:"):
			if strings.Contains(block, ln) {
				t.Errorf("age source not dropped (%q):\n%s", trimmed, block)
			}
		case strings.HasPrefix(trimmed, "bw:"):
			if !strings.Contains(block, ln) {
				t.Errorf("declared bw line not preserved in place (%q):\n%s", trimmed, block)
			}
		}
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

// TestSetBackendBW_FileSecret proves the line surgery is shape-agnostic: a file secret
// flips exactly like an env secret does — backend bw, age: dropped, declared bw: kept.
func TestSetBackendBW_FileSecret(t *testing.T) {
	const file = `version: 1
secrets:
  - id: kube
    plane: infra
    backend: age
    age: kube.cfg
    bw: { item: kube, field: notes }
    expose: { file: { var: KUBECONFIG, path: "~/.kube/c" } }
`
	out, err := SetBackendBW([]byte(file), "kube")
	if err != nil {
		t.Fatalf("SetBackendBW(file secret): %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "backend: bw") {
		t.Errorf("backend not flipped:\n%s", got)
	}
	if strings.Contains(got, "age: kube.cfg") {
		t.Errorf("age source not dropped:\n%s", got)
	}
	if !strings.Contains(got, "bw: { item: kube, field: notes }") {
		t.Errorf("declared bw not preserved in place:\n%s", got)
	}
	if !strings.Contains(got, "expose: { file: { var: KUBECONFIG, path: \"~/.kube/c\" } }") {
		t.Errorf("expose: block not preserved verbatim:\n%s", got)
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

// migratableAgeEntry returns the first age-backed registry entry SetBackendBW accepts,
// skipping the test when none remains — the day every entry has migrated.
func migratableAgeEntry(t *testing.T, registry []byte) string {
	t.Helper()
	reg, err := ParseRegistry(registry)
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	for _, s := range reg.Secrets {
		if s.Backend == BackendAge && s.BW != nil && assertMigratable(registry, s.ID) == nil {
			return s.ID
		}
	}
	t.Skip("no age-backed entry with a bw: target is left to activate")
	return ""
}
