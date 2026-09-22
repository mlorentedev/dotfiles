package secrets

import (
	"strings"
	"testing"
)

// A `from:` names where reconcile copies a value FROM when the declared item or
// field is absent. It is the only place reconcile ever gets a value, so the
// registry refuses every shape that would make that copy ambiguous or pointless.
func TestRegistryAcceptsAWellFormedFrom(t *testing.T) {
	const yml = "version: 1\nsecrets:\n" +
		"  - {id: PAT, plane: app, backend: bw, bw: {item: github-cli-pat, field: PAT, folder: Dotfiles/apps, from: {item: GitHub, field: \"Personal Access Token\"}}, expose: {env: PAT}}\n"
	reg, err := ParseRegistry([]byte(yml))
	if err != nil {
		t.Fatalf("a well-formed from: must parse: %v", err)
	}
	got := reg.Secrets[0].BW.From
	if got == nil || got.Item != "GitHub" || got.Field != "Personal Access Token" {
		t.Fatalf("from: not carried through: %+v", got)
	}
}

func TestRegistryRejectsAMalformedFrom(t *testing.T) {
	cases := map[string]struct{ bw, expose, want string }{
		"empty item": {
			`{item: dest, field: F, from: {field: src}}`, `{env: V}`, "from.item",
		},
		"empty field": {
			`{item: dest, field: F, from: {item: src}}`, `{env: V}`, "from.field",
		},
		"source is the destination": {
			`{item: dest, field: F, from: {item: dest, field: F}}`, `{env: V}`, "is its own destination",
		},
		// A multi-var secret has one item and a field PER VAR, so a single from:
		// cannot say which field it fills. Refused rather than guessed.
		"multi-var secret": {
			`{item: dest, from: {item: src, field: s}}`, `{env: {A: {field: a}, B: {field: b}}}`, "multi-var",
		},
		"control character": {
			"{item: dest, field: F, from: {item: \"sr\\tc\", field: s}}", `{env: V}`, "from",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			yml := "version: 1\nsecrets:\n  - {id: S, plane: personal, backend: bw, bw: " + tc.bw + ", expose: " + tc.expose + "}\n"
			_, err := ParseRegistry([]byte(yml))
			if err == nil {
				t.Fatal("want a rejection, got none")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error must name the problem (%q): %v", tc.want, err)
			}
		})
	}
}

// A dormant bw block (age-backed secret) is the migration target, and a from: on
// it is exactly the split this exists for — the GitHub PATs were age-backed with a
// revoked blob. It must validate there too, not only on bw-backed entries.
func TestRegistryValidatesFromOnADormantBlock(t *testing.T) {
	const yml = "version: 1\nsecrets:\n" +
		"  - {id: S, plane: app, backend: age, age: some.blob, bw: {item: dest, field: F, folder: Dotfiles/apps, from: {item: dest, field: F}}, expose: {env: S}}\n"
	if _, err := ParseRegistry([]byte(yml)); err == nil {
		t.Fatal("a malformed from: on a dormant block must be rejected too")
	}
}

// A file-exposed secret carries its from: into its declaration exactly as an env
// secret does — KUBECONFIG and the recovery codes are file-exposed, and a split
// that forgot them would be the file-expose defect of #1600 over again.
func TestBWDeclarationsCarryFromForFileExposedSecrets(t *testing.T) {
	const yml = "version: 1\nsecrets:\n" +
		"  - {id: K, plane: infra, backend: bw, bw: {item: kube, field: notes, folder: Dotfiles/infra, from: {item: old-kube, field: notes}}, expose: {file: {var: KUBECONFIG, path: \"~/.kube/x\"}}}\n" +
		"  - {id: E, plane: app, backend: bw, bw: {item: api, field: KEY, folder: Dotfiles/apps, from: {item: legacy, field: key}}, expose: {env: KEY}}\n"
	reg, err := ParseRegistry([]byte(yml))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	for _, d := range reg.BWDeclarations() {
		if d.From == nil {
			t.Errorf("declaration %s lost its from:", d.Secret)
		}
	}
}
