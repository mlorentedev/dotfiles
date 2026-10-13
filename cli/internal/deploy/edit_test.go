package deploy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEditJSON_WritesOnlyWhenTheEditChangesSomething(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(dst, []byte(`{"keep": 1, "drop": true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c := Config{Name: "x", Mode: "0644"}

	changed, err := EditJSON(c, dst, func(map[string]any) bool { return false })
	if err != nil || changed {
		t.Fatalf("a no-op edit: changed=%v err=%v", changed, err)
	}
	if _, err := os.Stat(dst + BackupSuffix); !os.IsNotExist(err) {
		t.Fatal("a no-op edit left a backup")
	}

	changed, err = EditJSON(c, dst, func(doc map[string]any) bool { delete(doc, "drop"); return true })
	if err != nil || !changed {
		t.Fatalf("an edit: changed=%v err=%v", changed, err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "drop") || !strings.Contains(string(got), `"keep": 1`) {
		t.Errorf("edited file = %s", got)
	}
	if _, err := os.Stat(dst + BackupSuffix); err != nil {
		t.Errorf("the first rewrite keeps a backup, as deploy does: %v", err)
	}
	// installMode keeps an operator's tighter mode, as deploy does.
	if info, err := os.Stat(dst); err == nil && runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %04o, want the existing 0600 kept", info.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Dir(dst))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".deploy-") {
			t.Errorf("a staged file was left behind: %s", e.Name())
		}
	}
}

// A repair never creates a file deploy has not, and never rewrites a file it
// cannot read as an object.
func TestEditJSON_RefusesAnAbsentOrNonObjectDestination(t *testing.T) {
	dir := t.TempDir()
	c := Config{Name: "x", Mode: "0644"}
	edit := func(map[string]any) bool { return true }
	if _, err := EditJSON(c, filepath.Join(dir, "absent.json"), edit); err == nil {
		t.Error("an absent destination was not an error")
	}
	if _, err := os.Stat(filepath.Join(dir, "absent.json")); !os.IsNotExist(err) {
		t.Error("an absent destination was created")
	}
	arr := filepath.Join(dir, "arr.json")
	if err := os.WriteFile(arr, []byte(`[1]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := EditJSON(c, arr, edit); err == nil {
		t.Error("a non-object destination was not an error")
	}
}
