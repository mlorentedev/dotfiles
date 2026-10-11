package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #2013 D10: every check in the sweep is classified machine or identity. None
// is left at the zero value, an identity check names the section it reports
// under and how to restore what it needs, and a section is claimed once.
func TestSweep_EveryCheckIsClassified(t *testing.T) {
	entries := sweep(nil, nil, nil, nil, Options{}, "")
	identity := 0
	seen := map[string]bool{}
	for i, c := range entries {
		switch c.kind {
		case kindMachine:
		case kindIdentity:
			identity++
			if c.section == "" || c.restore == "" {
				t.Errorf("entry %d: an identity check needs a section and a restore, got %q / %q", i, c.section, c.restore)
			}
			if seen[c.section] {
				t.Errorf("entry %d: section %q is classified twice", i, c.section)
			}
			seen[c.section] = true
		default:
			t.Errorf("entry %d has no kind: classify it machine or identity", i)
		}
		if c.run == nil {
			t.Errorf("entry %d runs nothing", i)
		}
	}
	if identity == 0 || identity == len(entries) {
		t.Errorf("want both kinds in the sweep, got %d identity of %d", identity, len(entries))
	}
}

// The SKIP printed for an identity check carries its section title, which is a
// literal in the check's own code. A renamed section would leave the SKIP under
// a heading the default run never prints.
func TestSweep_IdentitySectionsAreTheTitlesTheChecksPrint(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var src strings.Builder
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src.Write(b)
	}
	for _, c := range sweep(nil, nil, nil, nil, Options{}, "") {
		if c.kind != kindIdentity {
			continue
		}
		if want := `rep.Section("` + c.section + `")`; !strings.Contains(src.String(), want) {
			t.Errorf("no check prints %s; the identity entry's section does not match its check", want)
		}
	}
}

func TestParseScope(t *testing.T) {
	for in, want := range map[string]Scope{"": ScopeAll, "all": ScopeAll, "machine": ScopeMachine} {
		if got, err := ParseScope(in); err != nil || got != want {
			t.Errorf("ParseScope(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseScope("identity"); err == nil {
		t.Error("an unknown scope must be refused")
	}
}

// End to end on a machine with no identity: the default scope reports the
// missing vault, the machine scope runs no identity check at all and prints one
// SKIP per identity entry instead, and a skip never fails the run.
func TestRun_MachineScopeSkipsEveryIdentityCheck(t *testing.T) {
	home := t.TempDir()
	dotfiles := filepath.Join(home, ".dotfiles")
	writeFile(t, filepath.Join(dotfiles, "env-contract.json"),
		`{"env_vars":[],"required_path_entries":{"linux":[]},"required_binaries":[],"optional_binaries":[]}`)
	writeFile(t, filepath.Join(dotfiles, "versions.conf"), "GO_VERSION=1.26.0\n")
	env := map[string]string{"HOME": home, "DOTFILES_DIR": dotfiles}
	run := func(scope Scope) (string, int) {
		t.Helper()
		var buf bytes.Buffer
		code, err := Run(Options{Out: &buf, System: newSys(env, nil, nil), StartDir: home, Scope: scope})
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		return buf.String(), code
	}

	all, _ := run(ScopeAll)
	if !strings.Contains(all, "vault directory missing") {
		t.Fatalf("the default scope must run the vault check\n%s", all)
	}
	if strings.Contains(all, "not run under --scope machine") {
		t.Errorf("the default scope skipped a check\n%s", all)
	}

	machine, _ := run(ScopeMachine)
	if strings.Contains(machine, "vault directory missing") {
		t.Errorf("the machine scope ran the vault check\n%s", machine)
	}
	identity := 0
	for _, c := range sweep(nil, nil, nil, nil, Options{}, "") {
		if c.kind == kindIdentity {
			identity++
		}
	}
	if got := strings.Count(machine, "an identity check, not run under --scope machine"); got != identity {
		t.Errorf("want one SKIP per identity check (%d), got %d\n%s", identity, got, machine)
	}
	if !strings.Contains(machine, "[check, machine scope]") {
		t.Errorf("the header must say the scope\n%s", machine)
	}
}
