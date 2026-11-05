package initrepo

import (
	"os"
	"path/filepath"
	"testing"
)

func isDir(root, rel string) bool {
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil && info.IsDir()
}

// Each stack's code directories, one row per stack. An unknown stack keeps the
// full set, as every stack had before the table existed.
func TestScaffoldCodeDirectoriesByStack(t *testing.T) {
	all := []string{"src", "tests", "scripts"}
	for stack, want := range map[string][]string{
		"go":        all,
		"python":    all,
		"node":      all,
		"ts":        all,
		"none":      all,
		"":          all,
		"cobol":     all,
		"workflows": {"tests"},
	} {
		t.Run("stack="+stack, func(t *testing.T) {
			root := t.TempDir()
			if _, err := ScaffoldOpts(root, stack, false); err != nil {
				t.Fatalf("ScaffoldOpts: %v", err)
			}
			wanted := map[string]bool{}
			for _, d := range want {
				wanted[d] = true
			}
			for _, d := range all {
				if got := isDir(root, d); got != wanted[d] {
					t.Errorf("%s/ exists = %v, want %v", d, got, wanted[d])
				}
			}
			for _, d := range []string{"specs", "docs/adr", "docs/lessons", ".claude"} {
				if !isDir(root, d) {
					t.Errorf("shared directory %s/ missing", d)
				}
			}
		})
	}
}

func TestInitWorkflowsStackScaffoldsNoCodeAndNoCI(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workflows-repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(InitOptions{Root: root, Stack: "workflows", Date: "2026-11-04", SkipGithub: true}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	for _, d := range []string{"tests", "specs", "docs"} {
		if !isDir(root, d) {
			t.Errorf("expected %s/", d)
		}
	}
	for _, d := range []string{"src", "scripts"} {
		if isDir(root, d) {
			t.Errorf("a workflows repo got %s/", d)
		}
	}
	wf := filepath.Join(root, ".github", "workflows")
	if _, err := os.Stat(filepath.Join(wf, "lint.yml")); err != nil {
		t.Errorf("expected lint.yml: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wf, "ci.yml")); err == nil {
		t.Error("a workflows repo got ci.yml")
	}
}

func TestStackInitWorkflowsIsANoOp(t *testing.T) {
	res, err := StackInit(t.TempDir(), "workflows")
	if err != nil || len(res.Actions) != 0 || len(res.Skipped) != 0 {
		t.Errorf("got %+v, %v; want nothing done and nothing skipped", res, err)
	}
}
