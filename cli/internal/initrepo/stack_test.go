package initrepo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStackInitGoWritesModuleAndMakefile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myproj")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := StackInit(root, "go")
	if err != nil {
		t.Fatalf("StackInit(go): %v", err)
	}
	// go is available in CI; go.mod must exist.
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Errorf("expected go.mod: %v (actions=%v skipped=%v)", err, res.Actions, res.Skipped)
	}
	mk := readFile(t, filepath.Join(root, "Makefile"))
	if !strings.Contains(mk, "go build") {
		t.Errorf("Makefile should have a go build target:\n%s", mk)
	}
}

func TestStackInitNoneIsNoOp(t *testing.T) {
	root := t.TempDir()
	res, err := StackInit(root, "none")
	if err != nil {
		t.Fatalf("StackInit(none): %v", err)
	}
	if len(res.Actions) != 0 {
		t.Errorf("stack=none should do nothing, got actions %v", res.Actions)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Errorf("stack=none should not create files, found %d entries", len(entries))
	}
}

func TestStackInitSkipsWhenToolAbsent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir()) // neither uv nor poetry findable
	res, err := StackInit(root, "python")
	if err != nil {
		t.Fatalf("StackInit(python) should not error when tools are absent: %v", err)
	}
	if len(res.Skipped) == 0 {
		t.Errorf("expected a skip note when no python tool is found, got %+v", res)
	}
}
