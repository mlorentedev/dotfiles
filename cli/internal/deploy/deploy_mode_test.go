package deploy

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

// placeInSync writes dst with c's source content at the given mode, as a file
// deployed earlier and then chmod-ed would be.
func placeInSync(t *testing.T, dst, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dst, mode); err != nil {
		t.Fatal(err)
	}
}

func permOf(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

// #1664 (1): doctor runs PlanConfig and deploy runs ensureMode, so a mode-only
// drift must look the same from both. PlanConfig said "in sync" while
// `dotf deploy --dry-run` said `would fix mode`.
func TestPlanConfig_ReportsTheModeDriftDeployWouldFix(t *testing.T) {
	root := repoWithTrust(t, "ai/secret.json", `{"k":"v"}`)
	home := t.TempDir()
	c := Config{Name: "secret", Src: "ai/secret.json", Dst: "{HOME}/.tool/secret.json", Mode: "0600"}
	dst := filepath.Join(home, ".tool", "secret.json")
	placeInSync(t, dst, `{"k":"v"}`, 0o644) // on Windows: an inherited DACL

	p, err := PlanConfig(c, root, home, noResolve)
	if err != nil {
		t.Fatal(err)
	}
	dry, err := Deploy(c, root, home, noResolve, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if p.Changed || !p.ModeDrift || !dry.ModeFixed {
		t.Fatalf("plan and deploy must agree on a mode-only drift: plan %+v, dry run %+v", p, dry)
	}

	if _, err := Deploy(c, root, home, noResolve, nil, false); err != nil {
		t.Fatal(err)
	}
	if p, err = PlanConfig(c, root, home, noResolve); err != nil || p.ModeDrift || p.Changed {
		t.Fatalf("after the deploy the plan must be in sync: %+v, %v", p, err)
	}
}

// #1664 (5): the in-sync convergence narrows, never widens. An operator who
// tightened a deployed 0644 file to owner-only keeps it, and neither the deploy
// nor the plan doctor reads calls it drift. It runs on Windows too, where the
// tightening is the owner-only DACL fsmode writes: Needs never reads the DACL
// for a mode that grants group or other, so a 0644 entry cannot widen it.
func TestDeploy_InSyncKeepsAFileAnOperatorTightened(t *testing.T) {
	root := repoWithTrust(t, "ai/rc.json", `{"k":"v"}`)
	home := t.TempDir()
	c := Config{Name: "rc", Src: "ai/rc.json", Dst: "{HOME}/.tool/rc.json", Mode: "0644"}
	dst := filepath.Join(home, ".tool", "rc.json")
	placeInSync(t, dst, `{"k":"v"}`, 0o644)
	if err := fsmode.Apply(dst, 0o600); err != nil { // as the operator would, DACL included
		t.Fatal(err)
	}

	p, err := PlanConfig(c, root, home, noResolve)
	if err != nil {
		t.Fatal(err)
	}
	if p.Changed || p.ModeDrift {
		t.Fatalf("a tightened file is not drift: %+v", p)
	}
	res, err := Deploy(c, root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed || res.ModeFixed {
		t.Fatalf("a tightened file must be left alone: %+v", res)
	}
	if needs, err := fsmode.Needs(dst, 0o600); err != nil || needs {
		t.Fatalf("the deploy widened a tightened file: needs=%v err=%v", needs, err)
	}
}

// The convergence still removes what the declaration does not grant, and
// restores an owner bit: those are the fixes CLI-055 added it for.
func TestDeploy_InSyncConvergesTowardTheDeclaredMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful here")
	}
	for _, tc := range []struct {
		name           string
		declared, have os.FileMode
		want           os.FileMode
	}{
		{"a bit not declared is removed", 0o644, 0o666, 0o644},
		{"an execute bit not declared is removed", 0o600, 0o700, 0o600},
		{"a missing owner bit is restored", 0o644, 0o444, 0o644},
		{"a missing owner bit is restored, tightening kept", 0o644, 0o400, 0o600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := repoWithTrust(t, "ai/rc.json", `{"k":"v"}`)
			home := t.TempDir()
			c := Config{Name: "rc", Src: "ai/rc.json", Dst: "{HOME}/.tool/rc.json", Mode: octal(tc.declared)}
			dst := filepath.Join(home, ".tool", "rc.json")
			placeInSync(t, dst, `{"k":"v"}`, tc.have)

			res, err := Deploy(c, root, home, noResolve, nil, false)
			if err != nil {
				t.Fatal(err)
			}
			if !res.ModeFixed {
				t.Fatalf("want a mode fix, got %+v", res)
			}
			if got := permOf(t, dst); got != tc.want {
				t.Fatalf("got %04o, want %04o", got, tc.want)
			}
		})
	}
}

// #1664 (7): the rendered path has its own in-sync branch, after the render.
// A file whose rendered content matches but whose mode is off gets the mode
// fix there too, without a rewrite.
func TestDeploy_RenderedInSyncContentStillGetsItsDeclaredMode(t *testing.T) {
	root := repoWithTrust(t, "ai/secret.json", `{"k":"v"}`)
	home := t.TempDir()
	c := Config{Name: "secret", Src: "ai/secret.json", Dst: "{HOME}/.tool/secret.json", Mode: "0600", Render: true}
	dst := filepath.Join(home, ".tool", "secret.json")
	placeInSync(t, dst, `{"k":"v"}`, 0o644) // looser than declared; on Windows, an inherited DACL
	noop := func(string) error { return nil }

	res, err := Deploy(c, root, home, noResolve, noop, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed || !res.ModeFixed {
		t.Fatalf("want a mode fix on the rendered in-sync path, got %+v", res)
	}
	if needs, err := fsmode.Needs(dst, 0o600); err != nil || needs {
		t.Fatalf("the declared mode must be on the file, got needs=%v err=%v", needs, err)
	}
	if _, err := os.Stat(dst + BackupSuffix); !os.IsNotExist(err) {
		t.Fatal("a mode fix is not a rewrite: no backup may be taken")
	}
}

// #1664 (2): the installed mode is the declared one. The rendered case pins
// commit()'s fsmode.Apply: it alone undoes the renderer's own 0600
// (secrets.Render writes the staged copy anew through AtomicWrite). The fresh
// install is end to end only: stage() and commit() each set the mode there, so
// removing either one alone stays green. stage()'s call is pinned by
// TestDeploy_TheStagedCopyHasItsDeclaredModeBeforeTheRender.
func TestDeploy_TheInstalledModeIsTheDeclaredOne(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful here")
	}
	for _, tc := range []struct {
		name   string
		render Renderer
	}{
		{"a fresh install lands with the declared mode", nil},
		{"a rendered install is not left at the renderer's 0600", func(path string) error { return os.Chmod(path, 0o600) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := repoWithTrust(t, "ai/rc.json", `{"k":"v"}`)
			home := t.TempDir()
			c := Config{Name: "rc", Src: "ai/rc.json", Dst: "{HOME}/.tool/rc.json", Mode: "0644", Render: tc.render != nil}
			res, err := Deploy(c, root, home, noResolve, tc.render, false)
			if err != nil {
				t.Fatal(err)
			}
			if got := permOf(t, res.Dst); got != 0o644 {
				t.Fatalf("installed %04o, declared 0644", got)
			}
		})
	}
}

func octal(m os.FileMode) string {
	return "0" + strconv.FormatUint(uint64(m.Perm()), 8)
}

// #1664 (2): stage() sets the declared mode before anything reads the staged
// copy: the renderer writes credentials into it. On POSIX that turns
// os.CreateTemp's 0600 into a declared 0644; on Windows it replaces the DACL the
// temp file inherited with the owner-only one a 0600 declares, which is the
// half that matters, since commit() only applies it after the rename.
func TestDeploy_TheStagedCopyHasItsDeclaredModeBeforeTheRender(t *testing.T) {
	declared := os.FileMode(0o644)
	if runtime.GOOS == "windows" {
		declared = 0o600
	}
	root := repoWithTrust(t, "ai/rc.json", `{"k":"v"}`)
	home := t.TempDir()
	c := Config{Name: "rc", Src: "ai/rc.json", Dst: "{HOME}/.tool/rc.json", Mode: octal(declared), Render: true}
	var called, needs bool
	var needsErr error
	render := func(staged string) error {
		called = true
		needs, needsErr = fsmode.Needs(staged, declared)
		return nil
	}
	if _, err := Deploy(c, root, home, noResolve, render, false); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("the renderer never ran, so the staged copy's mode was never observed")
	}
	if needsErr != nil || needs {
		t.Fatalf("the renderer saw a staged copy without its declared %04o: needs=%v err=%v", declared, needs, needsErr)
	}
}

// The rewrite path keeps the tightening too: new content arriving over a
// file an operator narrowed to 0600 must not land 0644 (#1664 (5)).
func TestDeploy_ARewriteKeepsAFileAnOperatorTightened(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful here")
	}
	for _, render := range []Renderer{nil, func(string) error { return nil }} {
		root := repoWithTrust(t, "ai/rc.json", `{"k":"new"}`)
		home := t.TempDir()
		c := Config{Name: "rc", Src: "ai/rc.json", Dst: "{HOME}/.tool/rc.json", Mode: "0644", Render: render != nil}
		dst := filepath.Join(home, ".tool", "rc.json")
		placeInSync(t, dst, `{"k":"old"}`, 0o600)

		res, err := Deploy(c, root, home, noResolve, render, false)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Changed || res.ModeFixed {
			t.Fatalf("render=%v: want a content rewrite, got %+v", c.Render, res)
		}
		if got := permOf(t, dst); got != 0o600 {
			t.Fatalf("render=%v: the rewrite widened a tightened file to %04o", c.Render, got)
		}
	}
}
