package cmd

import (
	"strings"
	"testing"
)

func TestEnvRoleAddListRemove(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	steps := []struct {
		args []string
		want string
	}{
		{[]string{"env", "role"}, ""},
		{[]string{"env", "role", "add", "ssh-target"}, "added role ssh-target"},
		{[]string{"env", "role", "add", "ssh-target"}, "unchanged role ssh-target"},
		{[]string{"env", "role"}, "ssh-target\n"},
		{[]string{"env", "role", "rm", "ssh-target"}, "removed role ssh-target"},
		{[]string{"env", "role"}, ""},
	}
	for _, s := range steps {
		out, _, err := execute(t, s.args...)
		if err != nil {
			t.Fatalf("%v: %v", s.args, err)
		}
		if (s.want == "" && out != "") || !strings.Contains(out, s.want) {
			t.Fatalf("%v printed %q, want %q", s.args, out, s.want)
		}
	}
}

func TestEnvRoleRejectsAnUnknownRole(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, _, err := execute(t, "env", "role", "add", "ssh-targt"); err == nil {
		t.Fatal("env role add ssh-targt = nil error, want one")
	}
}
