package deploy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/fsmode"
)

// CLI-055 (#1302), found on the box: a 0600 config whose content was already
// in sync came back "in sync" and kept the ACL its directory handed it,
// because the in-sync path compared bytes and nothing else. The mode is part
// of what "deployed" means, so the in-sync path now asks fsmode.Needs and
// applies the declared mode without rewriting the content — reported as a
// mode fix, never as a deploy, and a dry run reports it without touching.
func TestDeploy_InSyncContentStillGetsItsDeclaredMode(t *testing.T) {
	root := repoWithTrust(t, "ai/secret.json", `{"k":"v"}`)
	home := t.TempDir()
	c := Config{Name: "secret", Src: "ai/secret.json", Dst: "{HOME}/.tool/secret.json", Mode: "0600"}
	dst := filepath.Join(home, ".tool", "secret.json")

	// The file as an older binary would have left it: same bytes, whatever
	// mode the directory and os.WriteFile give it.
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(`{"k":"v"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if needs, err := fsmode.Needs(dst, 0o600); err != nil || !needs {
		t.Fatalf("precondition: the file must be missing its mode, got needs=%v err=%v", needs, err)
	}

	dry, err := Deploy(c, root, home, noResolve, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if !dry.Changed || !dry.ModeFixed {
		t.Fatalf("dry run must report the mode fix it would make: %+v", dry)
	}
	if needs, _ := fsmode.Needs(dst, 0o600); !needs {
		t.Fatal("a dry run must not touch the file")
	}

	res, err := Deploy(c, root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed || !res.ModeFixed {
		t.Fatalf("the mode fix must be reported as one: %+v", res)
	}
	if needs, err := fsmode.Needs(dst, 0o600); err != nil || needs {
		t.Fatalf("after the run nothing must be left to fix, got needs=%v err=%v", needs, err)
	}
	if got, _ := os.ReadFile(dst); string(got) != `{"k":"v"}` {
		t.Fatalf("content must be untouched by a mode fix, got %q", got)
	}

	again, err := Deploy(c, root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed || again.ModeFixed {
		t.Fatalf("second run must be in sync, got %+v", again)
	}
}
