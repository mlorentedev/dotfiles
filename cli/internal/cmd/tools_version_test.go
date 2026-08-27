package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// The two banners that broke the setup scripts' token parse, plus the absent
// case: a shell caller tests stdout and the exit status, both must be right.
func TestToolsVersionCmd(t *testing.T) {
	orig := toolsVersionRunner
	t.Cleanup(func() { toolsVersionRunner = orig })
	toolsVersionRunner = func(name string, _ ...string) ([]byte, error) {
		switch name {
		case "hive":
			return []byte("hive-vault 3.0.0\n"), nil
		case "opencode":
			return []byte("OpenCode locked.\n1.16.2\n"), nil
		}
		return nil, errors.New("not found")
	}
	run := func(name string) (string, error) {
		c := newToolsVersionCmd()
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&bytes.Buffer{})
		c.SetArgs([]string{name})
		err := c.Execute()
		return strings.TrimSpace(out.String()), err
	}

	if got, err := run("hive"); err != nil || got != "3.0.0" {
		t.Errorf("hive: want 3.0.0, got %q (%v)", got, err)
	}
	if got, err := run("opencode"); err != nil || got != "1.16.2" {
		t.Errorf("opencode: the banner line must not be taken as the version, got %q (%v)", got, err)
	}
	if got, err := run("nope"); err == nil || got != "" {
		t.Errorf("absent tool: want exit error and empty stdout, got %q (%v)", got, err)
	}
}
