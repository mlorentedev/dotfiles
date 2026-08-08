package doctor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Regression guards for the two live defects that motivated the effectiveness
// probes. Both shipped green under checks that asked where an artifact sat.

// probeGit fakes the git seam per repo: `config --get core.hooksPath` answers
// from overrides, and `rev-parse --git-common-dir` answers "<repo>/.git" so the
// unset path resolves the way git resolves it in an ordinary checkout.
type probeGit struct {
	overrides map[string]string // repo -> local core.hooksPath
	global    string
	repoDir   string // answers DOTFILES_REPO_DIR, which resolveRepoDir reads
}

func (p *probeGit) system(vault string) *System {
	return &System{
		Getenv: func(k string) string {
			switch k {
			case "VAULT_PATH":
				return vault
			case "DOTFILES_REPO_DIR":
				return p.repoDir
			}
			return ""
		},
		LookPath: func(n string) (string, error) { return "/usr/bin/" + n, nil },
		CommandOutput: func(name string, args ...string) (string, error) {
			if name != "git" {
				return "", nil
			}
			joined := strings.Join(args, " ")
			// Global probe: `git config --global --get core.hooksPath`.
			if strings.HasPrefix(joined, "config --global") {
				if p.global == "" {
					return "", errors.New("exit status 1")
				}
				return p.global + "\n", nil
			}
			// Per-repo probes are `git -C <repo> ...`.
			if len(args) < 2 || args[0] != "-C" {
				return "", nil
			}
			repo := args[1]
			switch {
			case strings.Contains(joined, "--git-common-dir"):
				return filepath.Join(repo, ".git") + "\n", nil
			case strings.Contains(joined, "core.hooksPath"):
				// `git -C <repo> config --get` reports the EFFECTIVE value: a local
				// entry when one exists, otherwise the global one. Modelling it as
				// "local or nothing" would make the fake disagree with git in the
				// exact direction the check under test cares about.
				if v, ok := p.overrides[repo]; ok {
					return v + "\n", nil
				}
				if p.global != "" {
					return p.global + "\n", nil
				}
				return "", errors.New("exit status 1") // unset everywhere
			}
			return "", nil
		},
		CommandOutputDir: func(dir, name string, args ...string) (string, error) { return "", nil },
	}
}

// fullDispatcher is guardDispatcher plus the pre-push entrypoint the real
// dispatcher ships. The vault gate is a pre-push concern, so a fixture with only
// pre-commit models a dispatcher that cannot answer the question under test.
func fullDispatcher(t *testing.T, dir string) string {
	t.Helper()
	guardDispatcher(t, dir)
	writeExec(t, filepath.Join(dir, "pre-push"))
	return dir
}

// gitRepo makes dir look like an ordinary checkout to the probes.
func gitRepo(t *testing.T, dir string) string {
	t.Helper()
	mkdirAll(t, filepath.Join(dir, ".git", "hooks"))
	return dir
}

// BUG-056. The machine is wired correctly AND the guard is inert, because the
// repo carries a local core.hooksPath. Reading --global cannot tell these apart:
// doctor reported "all ok" both before and after the override was removed, while
// the guard went from not running to running. The check must now fail.
func TestGuardHooks_LocalOverrideDisablesGuard_MustFail(t *testing.T) {
	root := t.TempDir()
	dispatcher := guardDispatcher(t, filepath.Join(root, "deploy", "git-hooks"))

	repo := gitRepo(t, filepath.Join(root, "checkout"))
	writeExec(t, filepath.Join(repo, ".git", "hooks", "pre-commit")) // NOT a dispatcher
	// Local override points at the repo's own hooks — the shape found in the
	// dotfiles checkout, installed to route around BUG-055 and never revisited.
	g := &probeGit{global: dispatcher, repoDir: repo, overrides: map[string]string{
		repo: filepath.Join(repo, ".git", "hooks"),
	}}
	cfg := &Config{DotfilesDir: filepath.Join(root, "deploy")}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkGuardHooks(g.system(filepath.Join(root, "no-vault")), cfg, rep, false)

	if rep.Failures() == 0 {
		t.Fatalf("an inert guard must FAIL even with the global wiring correct; got none:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), repo) {
		t.Errorf("the failure must name the repo whose guard is off, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "--unset core.hooksPath") {
		t.Errorf("the failure must state the remedy, got: %s", buf.String())
	}
}

// The green direction: no override, the dispatcher is what git runs, so the
// guard fires. Without this the test above is satisfied by a check that fails
// unconditionally.
func TestGuardHooks_NoOverrideGuardFires_MustPass(t *testing.T) {
	root := t.TempDir()
	dispatcher := guardDispatcher(t, filepath.Join(root, "deploy", "git-hooks"))

	repo := gitRepo(t, filepath.Join(root, "checkout"))

	g := &probeGit{global: dispatcher, repoDir: repo} // no local override
	cfg := &Config{DotfilesDir: filepath.Join(root, "deploy")}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkGuardHooks(g.system(filepath.Join(root, "no-vault")), cfg, rep, false)

	if rep.Failures() != 0 {
		t.Fatalf("guard runs via the dispatcher; must not FAIL:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "guard fires in "+repo) {
		t.Errorf("want the per-repo effectiveness PASS, got: %s", buf.String())
	}
}

// The vault secret gate's false FAIL. `pre-commit install` REFUSES while
// core.hooksPath is set, so .git/hooks/<stage> is a file the design guarantees
// absent — yet gitleaks runs, via the dispatcher's fallback. A check keyed on
// that file reports INACTIVE on a working gate.
func TestVaultHooks_GateViaDispatcherFallback_MustPass(t *testing.T) {
	root := t.TempDir()
	dispatcher := fullDispatcher(t, filepath.Join(root, "deploy", "git-hooks"))

	vault := gitRepo(t, filepath.Join(root, "vault"))
	writeFile(t, filepath.Join(vault, ".pre-commit-config.yaml"), "repos: []\n")
	// Deliberately NO .git/hooks/pre-commit or pre-push here.

	g := &probeGit{global: dispatcher, overrides: map[string]string{vault: dispatcher}}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkVaultHooks(g.system(vault), rep, false)

	if rep.Failures() != 0 {
		t.Fatalf("the gate reaches pre-commit through the dispatcher; must not FAIL:\n%s", buf.String())
	}
}

// The red direction for the same check: a dispatcher with no config for it to
// act on is a gate in name only, because chaining into pre-commit with no
// .pre-commit-config.yaml is a documented no-op.
func TestVaultHooks_DispatcherWithoutConfig_IsNotAGate(t *testing.T) {
	root := t.TempDir()
	dispatcher := fullDispatcher(t, filepath.Join(root, "deploy", "git-hooks"))
	vault := gitRepo(t, filepath.Join(root, "vault"))
	// No .pre-commit-config.yaml -> checkVaultHooks SKIPs before the probe, which
	// is the correct answer ("not the knowledge vault?"), so assert the probe
	// itself rather than the check.
	g := &probeGit{global: dispatcher, overrides: map[string]string{vault: dispatcher}}

	if stageReachesPreCommit(g.system(vault), vault, "pre-push") {
		t.Error("a dispatcher with no pre-commit config must not count as a live gate")
	}
}

// A non-executable hook is one git silently ignores. Resolution must treat it as
// absent, or the probe re-introduces the file-exists question it replaced.
//
// POSIX-only by nature, not by convenience: Windows has no execute bit, and
// isExecFile says so explicitly (every regular file is executable there). The
// distinction this asserts does not exist on that platform, so skipping is the
// honest answer rather than weakening the assertion for both.
func TestHookForStage_NonExecutableIsNotAHook(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no execute bit on Windows; isExecFile treats every regular file as executable")
	}
	root := t.TempDir()
	repo := gitRepo(t, filepath.Join(root, "repo"))
	p := filepath.Join(repo, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := &probeGit{}

	if got := hookForStage(g.system(""), repo, "pre-commit"); got != "" {
		t.Errorf("a non-executable hook must resolve to \"\", got %q", got)
	}
}

// A System assembled without the command seam must degrade to "unset" rather
// than panic — the shape that took down the whole doctor suite on first run.
func TestEffectiveHooksPath_NilSeamIsUnsetNotPanic(t *testing.T) {
	if got := effectiveHooksPath(&System{}, "/nowhere"); got != "" {
		t.Errorf("a nil command seam must read as unset, got %q", got)
	}
}
