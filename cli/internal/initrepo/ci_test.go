package initrepo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteCIByStack(t *testing.T) {
	cases := map[string]string{
		"go":     "setup-go",
		"python": "setup-python",
		"node":   "setup-node",
		"ts":     "setup-node",
	}
	for stack, marker := range cases {
		root := t.TempDir()
		action, err := WriteCI(root, stack)
		if err != nil {
			t.Fatalf("WriteCI(%s): %v", stack, err)
		}
		if action != "created" {
			t.Errorf("WriteCI(%s) action = %q, want created", stack, action)
		}
		got := readFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"))
		if !strings.Contains(got, marker) {
			t.Errorf("ci.yml for %s should reference %q:\n%s", stack, marker, got)
		}
	}
}

func TestWriteCINoneIsNoOp(t *testing.T) {
	root := t.TempDir()
	action, err := WriteCI(root, "none")
	if err != nil {
		t.Fatalf("WriteCI(none): %v", err)
	}
	if action != "none" {
		t.Errorf("action = %q, want none", action)
	}
	if _, err := os.Stat(filepath.Join(root, ".github")); err == nil {
		t.Error("stack=none should not create a .github directory")
	}
}

func TestWriteCISkipsExisting(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, ".github", "workflows", "ci.yml")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	custom := "name: my-own-ci\n"
	if err := os.WriteFile(dest, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	action, err := WriteCI(root, "go")
	if err != nil {
		t.Fatalf("WriteCI: %v", err)
	}
	if action != "skipped" {
		t.Errorf("action = %q, want skipped", action)
	}
	if got := readFile(t, dest); got != custom {
		t.Errorf("WriteCI clobbered an existing ci.yml:\n%s", got)
	}
}
