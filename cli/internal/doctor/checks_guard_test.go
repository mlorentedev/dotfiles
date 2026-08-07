package doctor

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// recordingGit is a System whose CommandOutput records every invocation and
// answers `git config --global --get core.hooksPath` from a fixed value, so the
// AC5 idempotence + no-clobber contract can be asserted on the exact commands
// issued (not just the report text).
type recordingGit struct {
	getValue string // value returned for the --get probe
	unset    bool   // true → probe exits non-zero (key absent)
	calls    []string
}

func (g *recordingGit) system() *System {
	return &System{
		Getenv:   func(string) string { return "" },
		LookPath: func(n string) (string, error) { return "/usr/bin/" + n, nil },
		CommandOutput: func(name string, args ...string) (string, error) {
			full := name + " " + strings.Join(args, " ")
			g.calls = append(g.calls, full)
			if name == "git" && len(args) >= 4 && args[2] == "--get" {
				if g.unset {
					return "", errors.New("exit status 1")
				}
				return g.getValue + "\n", nil
			}
			return "", nil // the write succeeds
		},
	}
}

func (g *recordingGit) issued(cmd string) bool {
	for _, c := range g.calls {
		if c == cmd {
			return true
		}
	}
	return false
}

func guardCfg(t *testing.T, withDispatcher bool) (*Config, string) {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "git-hooks")
	if withDispatcher {
		writeExec(t, filepath.Join(target, "pre-commit"))
	}
	return &Config{DotfilesDir: dir}, target
}

func TestCheckGuardHooks_MissingDispatcherFails(t *testing.T) {
	cfg, _ := guardCfg(t, false) // dispatcher NOT deployed
	g := &recordingGit{unset: true}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkGuardHooks(g.system(), cfg, rep, true)

	if rep.Failures() != 1 {
		t.Fatalf("missing dispatcher must FAIL once, got %d", rep.Failures())
	}
	if !strings.Contains(buf.String(), "dispatcher not found") {
		t.Errorf("want 'dispatcher not found', got: %s", buf.String())
	}
	if g.issued("git config --global core.hooksPath " + filepath.Join(cfg.DotfilesDir, "git-hooks")) {
		t.Error("must not wire hooksPath when the dispatcher is absent")
	}
}

func TestCheckGuardHooks_UnsetFailsWithoutFix(t *testing.T) {
	cfg, target := guardCfg(t, true)
	g := &recordingGit{unset: true}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkGuardHooks(g.system(), cfg, rep, false)

	if rep.Failures() != 1 {
		t.Fatalf("unset hooksPath must FAIL in check mode, got %d", rep.Failures())
	}
	if g.issued("git config --global core.hooksPath " + target) {
		t.Error("check mode must never write; --fix is required to wire")
	}
}

func TestCheckGuardHooks_UnsetWiredOnFix(t *testing.T) {
	cfg, target := guardCfg(t, true)
	g := &recordingGit{unset: true}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkGuardHooks(g.system(), cfg, rep, true)

	if rep.Failures() != 0 {
		t.Fatalf("--fix on an unset hooksPath must not FAIL, got %d", rep.Failures())
	}
	if !g.issued("git config --global core.hooksPath " + target) {
		t.Errorf("--fix must wire core.hooksPath → %s; calls: %v", target, g.calls)
	}
	if !strings.Contains(buf.String(), "wired core.hooksPath") {
		t.Errorf("want a FIX line, got: %s", buf.String())
	}
}

func TestCheckGuardHooks_AlreadyWiredIsIdempotent(t *testing.T) {
	cfg, target := guardCfg(t, true)
	g := &recordingGit{getValue: target}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkGuardHooks(g.system(), cfg, rep, true) // fix=true, but already wired

	if rep.Failures() != 0 {
		t.Fatalf("already-wired must PASS, got %d", rep.Failures())
	}
	if g.issued("git config --global core.hooksPath " + target) {
		t.Error("idempotent: must NOT re-write an already-correct hooksPath")
	}
	if !strings.Contains(buf.String(), "wired to the GUARD dispatcher") {
		t.Errorf("want the wired PASS, got: %s", buf.String())
	}
}

// guardDispatcher writes a full GUARD dispatcher (entrypoint + the memory-sink
// guard it delegates to) at dir, so it is recognizable as an *equivalent*
// dispatcher rather than merely a directory holding some pre-commit file.
func guardDispatcher(t *testing.T, dir string) string {
	t.Helper()
	writeExec(t, filepath.Join(dir, "pre-commit"))
	writeExec(t, filepath.Join(dir, "lib", "memory-sink-guard.sh"))
	return dir
}

// AC1: hooksPath at a DIFFERENT directory that is nonetheless a GUARD dispatcher
// means the guard IS running. Reporting it INACTIVE is the #766 false negative:
// the check tested path identity instead of gate effectiveness.
func TestCheckGuardHooks_EquivalentDispatcherElsewherePasses(t *testing.T) {
	cfg, target := guardCfg(t, true)
	elsewhere := guardDispatcher(t, filepath.Join(t.TempDir(), "checkout", "git-hooks"))
	g := &recordingGit{getValue: elsewhere}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkGuardHooks(g.system(), cfg, rep, true)

	if rep.Failures() != 0 {
		t.Fatalf("an equivalent dispatcher must not FAIL, got %d", rep.Failures())
	}
	if strings.Contains(buf.String(), "GUARD inactive") {
		t.Errorf("guard IS active via %s; must not report it inactive: %s", elsewhere, buf.String())
	}
	if !strings.Contains(buf.String(), elsewhere) {
		t.Errorf("the active path must be named so the divergence stays visible, got: %s", buf.String())
	}
	if g.issued("git config --global core.hooksPath " + target) {
		t.Error("must NOT repoint hooksPath; no-clobber stands")
	}
}

// AC2: a directory with a pre-commit but WITHOUT the memory-sink guard is not a
// GUARD dispatcher — the guard genuinely cannot run, so the WARN must survive.
func TestCheckGuardHooks_ForeignPreCommitStillWarns(t *testing.T) {
	cfg, _ := guardCfg(t, true)
	foreign := filepath.Join(t.TempDir(), "other", "git-hooks")
	writeExec(t, filepath.Join(foreign, "pre-commit")) // entrypoint only, no lib/
	g := &recordingGit{getValue: foreign}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkGuardHooks(g.system(), cfg, rep, true)

	if !strings.Contains(buf.String(), "preserving it") {
		t.Errorf("a pre-commit without the memory-sink guard must still WARN, got: %s", buf.String())
	}
}

// AC3: git reports core.hooksPath with forward slashes even on Windows, so a
// byte compare against a filepath.Join target is a false negative there.
func TestCheckGuardHooks_SeparatorAndTrailingSlashNormalize(t *testing.T) {
	cfg, target := guardCfg(t, true)
	g := &recordingGit{getValue: filepath.ToSlash(target) + "/"}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkGuardHooks(g.system(), cfg, rep, true)

	if rep.Failures() != 0 {
		t.Fatalf("a separator-variant of the target must not FAIL, got %d", rep.Failures())
	}
	if !strings.Contains(buf.String(), "wired to the GUARD dispatcher") {
		t.Errorf("want the tier-1 wired PASS, got: %s", buf.String())
	}
	if g.issued("git config --global core.hooksPath " + target) {
		t.Error("idempotent: a normalized-equal hooksPath must not be re-written")
	}
}

func TestCheckGuardHooks_UnrelatedIsPreservedNotClobbered(t *testing.T) {
	cfg, target := guardCfg(t, true)
	g := &recordingGit{getValue: "/home/u/.my-hooks"}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkGuardHooks(g.system(), cfg, rep, true) // fix=true, but a foreign hooksPath exists

	if rep.Failures() != 0 {
		t.Fatalf("a foreign hooksPath is advisory (WARN), not a FAIL, got %d", rep.Failures())
	}
	if g.issued("git config --global core.hooksPath " + target) {
		t.Error("must NOT clobber an unrelated pre-existing core.hooksPath")
	}
	if !strings.Contains(buf.String(), "preserving it") {
		t.Errorf("want a preserve WARN, got: %s", buf.String())
	}
}
