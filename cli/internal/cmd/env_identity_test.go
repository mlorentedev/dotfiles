package cmd

import (
	"strings"
	"testing"
)

// The loop #1236 asks for: an undeclared machine says so, declares an id
// without a hand edit, and declaring it again changes nothing.
func TestEnvIdentityDeclareShowAndRepeat(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if _, _, err := execute(t, "env", "identity"); err == nil || !strings.Contains(err.Error(), "dotf env identity set") {
		t.Fatalf("env identity on an undeclared machine = %v, want an error naming the verb", err)
	}
	steps := []struct {
		args []string
		want string
	}{
		{[]string{"env", "identity", "set", "macmini"}, "declared identity macmini"},
		{[]string{"env", "identity", "set", "macmini"}, "unchanged identity macmini"},
		{[]string{"env", "identity"}, "macmini\n"},
	}
	for _, s := range steps {
		out, _, err := execute(t, s.args...)
		if err != nil {
			t.Fatalf("%v: %v", s.args, err)
		}
		if !strings.Contains(out, s.want) {
			t.Fatalf("%v printed %q, want %q", s.args, out, s.want)
		}
	}
}

// A different id is refused without --replace and taken with it.
func TestEnvIdentityRenameNeedsReplace(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, _, err := execute(t, "env", "identity", "set", "msi"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := execute(t, "env", "identity", "set", "other"); err == nil || !strings.Contains(err.Error(), "--replace") {
		t.Fatalf("a rename without --replace = %v, want a refusal naming --replace", err)
	}
	if _, _, err := execute(t, "env", "identity", "set", "other", "--replace"); err != nil {
		t.Fatalf("a rename with --replace: %v", err)
	}
	if out, _, _ := execute(t, "env", "identity"); !strings.Contains(out, "other") {
		t.Fatalf("env identity printed %q after the rename, want other", out)
	}
}
