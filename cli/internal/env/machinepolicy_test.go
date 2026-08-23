package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedMachineFile(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "machine.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("seed machine.json: %v", err)
	}
	return path
}

// The default is the security-relevant one. ADR-032 §8: a machine whose
// identity cannot be established denies every non-local pool, so the unknown
// case degrades to "no cross-pool dispatch" rather than to "all pools allowed".
//
// The failure this prevents is specific and silent: a rebuilt corporate machine
// that has not restored machine.json probes successfully for personal pools and
// would otherwise default to allowing them.
func TestLoadMachinePolicy_UnidentifiedMachineIsNotAPermissiveOne(t *testing.T) {
	tests := []struct {
		name string
		body string
		// path empty means "the file does not exist at all"
		missing bool
	}{
		{name: "no file at all", missing: true},
		{name: "a file with only paths", body: `{"paths": {"VAULT_PATH": "/v"}}`},
		{name: "a machine block with no id", body: `{"machine": {}}`},
		{name: "an id that is only whitespace", body: `{"machine": {"id": "   "}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "machine.json")
			if !tc.missing {
				path = seedMachineFile(t, tc.body)
			}
			p, err := LoadMachinePolicy(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if p.Identified {
				t.Fatal("identity reported as established; an undeclared machine must not be a trusted one")
			}
			// Every pool is denied, including ones no deny list mentions.
			for _, pool := range []string{"nan", "claude", "anything"} {
				if !p.Denies(pool) {
					t.Errorf("pool %q allowed on an unidentified machine", pool)
				}
			}
		})
	}
}

func TestLoadMachinePolicy_DeclaredIdentity(t *testing.T) {
	path := seedMachineFile(t, `{
      "paths": {"VAULT_PATH": "/v"},
      "machine": {"id": "msi"},
      "pools": {"deny": ["claude"]}
    }`)

	p, err := LoadMachinePolicy(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !p.Identified {
		t.Fatal("a declared id did not establish identity")
	}
	if p.ID != "msi" {
		t.Errorf("id = %q, want msi", p.ID)
	}
	if !p.Denies("claude") {
		t.Error("claude is on the deny list and was allowed")
	}
	if p.Denies("nan") {
		t.Error("nan is not on the deny list and was denied")
	}
	if len(p.Deny) != 1 || p.Deny[0] != "claude" {
		t.Errorf("Deny = %v, want [claude]", p.Deny)
	}
}

// An identified machine with no deny list denies nothing. The fail-closed
// default is about UNKNOWN identity, not about silence on denial: conflating
// them would make declaring an identity useless.
func TestLoadMachinePolicy_IdentifiedWithNoDenyListAllowsEverything(t *testing.T) {
	p, err := LoadMachinePolicy(seedMachineFile(t, `{"machine": {"id": "msi"}}`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !p.Identified {
		t.Fatal("identity not established")
	}
	for _, pool := range []string{"nan", "claude", "copilot"} {
		if p.Denies(pool) {
			t.Errorf("pool %q denied with no deny list present", pool)
		}
	}
}

// A malformed machine.json must not read as an absent one. Absent means "not
// declared" and denies; malformed means the file is there and cannot be
// trusted, which has to be loud — silently denying would send an operator
// hunting for a policy that is actually a syntax error.
func TestLoadMachinePolicy_MalformedIsAnErrorNotAnAbsence(t *testing.T) {
	_, err := LoadMachinePolicy(seedMachineFile(t, `{"machine": {"id": "msi"`))
	if err == nil {
		t.Fatal("a truncated machine.json parsed successfully")
	}
	if !strings.Contains(err.Error(), "machine.json") {
		t.Errorf("error does not name the file: %v", err)
	}
}

// A deny entry naming a pool that does not exist is almost always a typo, and
// the typo fails in the dangerous direction: `claud` in the list leaves
// `claude` allowed on a machine that meant to forbid it.
func TestMachinePolicy_ValidateDenyNames(t *testing.T) {
	declared := map[string]bool{"nan": true, "claude": true, "copilot": true}

	t.Run("every name resolves", func(t *testing.T) {
		p := MachinePolicy{Identified: true, Deny: []string{"claude", "nan"}}
		if err := p.ValidateDeny(declared); err != nil {
			t.Errorf("valid deny list rejected: %v", err)
		}
	})

	t.Run("a typo is rejected and named", func(t *testing.T) {
		p := MachinePolicy{Identified: true, Deny: []string{"claud"}}
		err := p.ValidateDeny(declared)
		if err == nil {
			t.Fatal("a deny entry naming no declared pool was accepted; the pool it meant stays allowed")
		}
		if !strings.Contains(err.Error(), "claud") {
			t.Errorf("error does not name the offending entry: %v", err)
		}
	})

	t.Run("an empty deny list is not an error", func(t *testing.T) {
		p := MachinePolicy{Identified: true}
		if err := p.ValidateDeny(declared); err != nil {
			t.Errorf("empty deny list rejected: %v", err)
		}
	})
}
