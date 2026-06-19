package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckCoreTools_MissingFails(t *testing.T) {
	// Everything on PATH except terraform.
	onPath := []string{"git", "zsh", "bash", "curl", "wget", "jq", "eza", "direnv", "node", "npm", "zoxide", "docker", "kubectl"}
	rep := capture(&bytes.Buffer{})
	checkCoreTools(newSys(nil, onPath, nil), nil, rep)
	if rep.Failures() != 1 {
		t.Fatalf("expected exactly 1 failure (terraform), got %d", rep.Failures())
	}
}

func TestCheckCoreTools_SkipsContractCovered(t *testing.T) {
	// git+jq are contract-covered → core-tools must not re-report them. With
	// neither on PATH, the only failures should be the OTHER core tools.
	c := &Contract{RequiredBinaries: []ContractBinary{{Name: "git"}, {Name: "jq"}}}
	var buf bytes.Buffer
	rep := capture(&buf)
	checkCoreTools(newSys(nil, nil, nil), c, rep)
	if strings.Contains(buf.String(), "git ") || strings.Contains(buf.String(), "jq ") {
		t.Error("core-tools must skip contract-covered binaries (git, jq)")
	}
}

func TestCheckRequiredBinaries(t *testing.T) {
	c := &Contract{RequiredBinaries: []ContractBinary{
		{Name: "git", Required: true, MinVersion: "2.30.0", VersionPattern: `git version ([0-9]+\.[0-9]+\.[0-9]+)`},
		{Name: "jq", Required: true, MinVersion: "1.6", VersionPattern: `jq-?([0-9]+\.[0-9]+(\.[0-9]+)?)`},
	}}

	tests := []struct {
		name     string
		onPath   []string
		cmdOut   map[string]string
		wantFail int
	}{
		{"git ok, jq ok", []string{"git", "jq"},
			map[string]string{"git --version": "git version 2.43.0", "jq --version": "jq-1.7.1"}, 0},
		{"git too old", []string{"git", "jq"},
			map[string]string{"git --version": "git version 2.20.0", "jq --version": "jq-1.7"}, 1},
		{"git missing", []string{"jq"},
			map[string]string{"jq --version": "jq-1.7"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep := capture(&bytes.Buffer{})
			checkRequiredBinaries(newSys(nil, tt.onPath, tt.cmdOut), c, rep)
			if rep.Failures() != tt.wantFail {
				t.Errorf("failures = %d, want %d", rep.Failures(), tt.wantFail)
			}
		})
	}
}

func TestCheckRequiredBinaries_UnparseableWarns(t *testing.T) {
	c := &Contract{RequiredBinaries: []ContractBinary{
		{Name: "git", Required: true, MinVersion: "2.30.0", VersionPattern: `git version ([0-9]+\.[0-9]+\.[0-9]+)`},
	}}
	var buf bytes.Buffer
	rep := capture(&buf)
	checkRequiredBinaries(newSys(nil, []string{"git"}, map[string]string{"git --version": "weird output"}), c, rep)
	if rep.Failures() != 0 {
		t.Error("unparseable version must WARN, not FAIL")
	}
	if !strings.Contains(buf.String(), "unparseable") {
		t.Error("expected an unparseable warning")
	}
}

func TestCheckVersionedPaths(t *testing.T) {
	home := t.TempDir()
	javaHome := filepath.Join(home, "jdk")
	writeExec(t, filepath.Join(javaHome, "bin", "java"))
	goHomeNoBin := filepath.Join(home, "go-broken")
	writeFile(t, filepath.Join(goHomeNoBin, "README"), "x") // dir exists, no bin/go

	env := map[string]string{
		"HOME":        home,
		"JAVA_HOME":   javaHome,
		"GO_HOME":     goHomeNoBin,
		"PYTHON_HOME": filepath.Join(home, "does-not-exist"),
		// MAVEN_HOME, MINIKUBE_HOME unset → SKIP
	}
	var buf bytes.Buffer
	rep := capture(&buf)
	checkVersionedPaths(newSys(env, nil, nil), rep)

	// java ok; go dir-but-no-binary fail; python missing fail = 2 failures.
	if rep.Failures() != 2 {
		t.Fatalf("failures = %d, want 2\n%s", rep.Failures(), buf.String())
	}
	if !strings.Contains(buf.String(), "MAVEN_HOME (variable not set)") {
		t.Error("unset MAVEN_HOME should SKIP")
	}
}

func TestCheckVersionMatch(t *testing.T) {
	home := t.TempDir()
	apps := filepath.Join(home, "Applications")
	mkdirAll(t, filepath.Join(apps, "go-1.26.0"))
	// Java dir intentionally missing.
	env := map[string]string{"HOME": home, "APPS_HOME": apps}
	cfg := &Config{Versions: map[string]string{
		"GO_VERSION":   "1.26.0",
		"JAVA_VERSION": "21.0.4",
		"YARN_VERSION": "1.22.22",
	}}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkVersionMatch(newSys(env, []string{"yarn"}, map[string]string{"yarn --version": "1.0.0"}), cfg, rep)

	// Go dir present (pass); Java dir missing (fail); yarn drift (warn, not fail).
	if rep.Failures() != 1 {
		t.Fatalf("failures = %d, want 1 (java)\n%s", rep.Failures(), buf.String())
	}
	if !strings.Contains(buf.String(), "yarn version drift") {
		t.Error("yarn drift should WARN")
	}
}

// TestOSGatingSkipsLinuxOnlyChecksOnWindows exercises the GOOS seam: on Windows
// the POSIX-only checks SKIP instead of emitting false failures.
func TestOSGatingSkipsLinuxOnlyChecksOnWindows(t *testing.T) {
	winSys := func() *System {
		s := newSys(map[string]string{}, []string{"git"}, nil)
		s.GOOS = "windows"
		return s
	}

	t.Run("tmux skips", func(t *testing.T) {
		var b bytes.Buffer
		rep := capture(&b)
		checkTmux(winSys(), &Config{}, rep)
		if rep.Failures() != 0 || !strings.Contains(b.String(), "Linux-only") {
			t.Errorf("tmux must SKIP on Windows\n%s", b.String())
		}
	})

	t.Run("version match skips versioned dirs", func(t *testing.T) {
		var b bytes.Buffer
		rep := capture(&b)
		checkVersionMatch(winSys(), &Config{Versions: map[string]string{"JAVA_VERSION": "21.0.4"}}, rep)
		if rep.Failures() != 0 {
			t.Errorf("versioned-dir checks must SKIP on Windows\n%s", b.String())
		}
	})

	t.Run("core tools skip posix-only", func(t *testing.T) {
		var b bytes.Buffer
		rep := capture(&b)
		checkCoreTools(winSys(), &Contract{}, rep)
		if !strings.Contains(b.String(), "zsh (POSIX-only") || !strings.Contains(b.String(), "direnv (POSIX-only") {
			t.Errorf("zsh/direnv must SKIP as POSIX-only on Windows\n%s", b.String())
		}
	})

	t.Run("tool-home vars skip when unset", func(t *testing.T) {
		var b bytes.Buffer
		rep := capture(&b)
		checkToolHomeEnvVars(winSys(), rep)
		if rep.Failures() != 0 {
			t.Errorf("tool-home vars must SKIP when unset on Windows\n%s", b.String())
		}
	})
}

func TestCheckSymlinks(t *testing.T) {
	home := t.TempDir()
	// Valid symlink: ~/.zshrc -> a real target.
	target := filepath.Join(home, "repo", ".zshrc")
	writeFile(t, target, "# zshrc")
	mustSymlink(t, target, filepath.Join(home, ".zshrc"))
	// Broken symlink: ~/.bashrc -> nonexistent.
	mustSymlink(t, filepath.Join(home, "gone"), filepath.Join(home, ".bashrc"))
	// Real file (not a symlink): ~/.dotfiles dir.
	mkdirAll(t, filepath.Join(home, ".dotfiles"))
	// The rest (.zsh/aliases.zsh, functions.zsh, .ssh/config) are missing.

	env := map[string]string{"HOME": home}
	var buf bytes.Buffer
	rep := capture(&buf)
	checkSymlinks(newSys(env, nil, nil), rep)

	if !strings.Contains(buf.String(), ".zshrc symlink valid") {
		t.Error("valid symlink should PASS")
	}
	if !strings.Contains(buf.String(), ".bashrc symlink broken") {
		t.Error("dangling symlink should FAIL")
	}
	if !strings.Contains(buf.String(), ".dotfiles exists (not a symlink)") {
		t.Error("real file should PASS as exists")
	}
	// 3 missing (aliases, functions, ssh/config) + 1 broken (.bashrc) = 4 fails,
	// plus the two claude-mem SKIPs (no installed_plugins.json).
	if rep.Failures() != 4 {
		t.Errorf("failures = %d, want 4\n%s", rep.Failures(), buf.String())
	}
}

func TestCheckContractEnvVars(t *testing.T) {
	home := t.TempDir()
	existing := filepath.Join(home, "exists")
	mkdirAll(t, existing)

	contract := &Contract{EnvVars: []ContractEnvVar{
		{Name: "SET_OK", Validation: "path_exists"}, // set → existing dir
		{Name: "UNSET_DEFAULT_MISSING", Default: map[string]string{"linux": "$HOME/nope"}, Validation: "path_exists", Required: true},
		{Name: "UNSET_NO_DEFAULT_REQ", Required: true},                          // unset, no default, required → FAIL
		{Name: "USERPROFILE", RequiredOn: "windows", Validation: "path_exists"}, // windows-scoped → skipped on linux
	}}
	env := map[string]string{"HOME": home, "SET_OK": existing}

	// Check mode.
	var buf bytes.Buffer
	rep := capture(&buf)
	checkContractEnvVars(newSys(env, nil, nil), contract, rep, false)
	// UNSET_DEFAULT_MISSING: warn + fail (default dir missing). UNSET_NO_DEFAULT_REQ: fail.
	if rep.Failures() != 2 {
		t.Fatalf("check-mode failures = %d, want 2\n%s", rep.Failures(), buf.String())
	}
	if !strings.Contains(buf.String(), "windows-scoped, skipped") {
		t.Error("windows-scoped var must be skipped on Linux")
	}

	// Fix mode: the unset-with-default becomes a FIX line naming the profile export.
	var fixBuf bytes.Buffer
	repFix := capture(&fixBuf)
	checkContractEnvVars(newSys(env, nil, nil), contract, repFix, true)
	if !strings.Contains(fixBuf.String(), "export UNSET_DEFAULT_MISSING=") {
		t.Errorf("--fix must print the profile export line\n%s", fixBuf.String())
	}
}

func TestCheckSecrets(t *testing.T) {
	dotfiles := t.TempDir()
	secrets := filepath.Join(dotfiles, "sensitive")
	writeFile(t, filepath.Join(secrets, "OPENAI.secret.age"), "x")
	writeFile(t, filepath.Join(secrets, "KUBECONFIG.secret.age"), "x")
	writeFile(t, filepath.Join(secrets, "orphan.secret.age"), "x") // no mapping → orphan
	// mapping references OPENAI (env) + KUBECONFIG (file) + MISSING (no .age file).
	writeFile(t, filepath.Join(secrets, "env-mapping.conf"),
		"# comment\nOPENAI=OPENAI\n@KUBECONFIG=KUBECONFIG>~/.kube/config\nMISSING=MISSING\n")

	cfg := &Config{DotfilesDir: dotfiles}
	var buf bytes.Buffer
	rep := capture(&buf)
	checkSecrets(newSys(map[string]string{"HOME": dotfiles}, nil, nil), cfg, rep)

	// MISSING.secret.age absent (fail) + orphan.secret.age unmapped (fail) = 2.
	if rep.Failures() != 2 {
		t.Fatalf("failures = %d, want 2\n%s", rep.Failures(), buf.String())
	}
	if !strings.Contains(buf.String(), "orphan: orphan.secret.age") {
		t.Error("unmapped .age file should be reported as orphan")
	}
	if !strings.Contains(buf.String(), "KUBECONFIG [file]") {
		t.Error("@-prefixed file secret should be labelled [file]")
	}
}

func TestCheckTmux(t *testing.T) {
	home := t.TempDir()
	dotfiles := filepath.Join(home, ".dotfiles")
	writeFile(t, filepath.Join(dotfiles, "tmux.conf"), "set -g mouse on\n")
	cfg := &Config{DotfilesDir: dotfiles}
	cmd := map[string]string{"tmux -V": "tmux 3.4"}

	t.Run("deployed matches", func(t *testing.T) {
		writeFile(t, filepath.Join(home, ".tmux.conf"), "set -g mouse on\n")
		var buf bytes.Buffer
		rep := capture(&buf)
		checkTmux(newSys(map[string]string{"HOME": home}, []string{"tmux"}, cmd), cfg, rep)
		if rep.Failures() != 0 {
			t.Fatalf("matching deploy should pass\n%s", buf.String())
		}
	})

	t.Run("drifted", func(t *testing.T) {
		writeFile(t, filepath.Join(home, ".tmux.conf"), "DIFFERENT\n")
		var buf bytes.Buffer
		rep := capture(&buf)
		checkTmux(newSys(map[string]string{"HOME": home}, []string{"tmux"}, cmd), cfg, rep)
		if rep.Failures() != 1 || !strings.Contains(buf.String(), "drifted") {
			t.Fatalf("drift should fail\n%s", buf.String())
		}
	})

	t.Run("not installed", func(t *testing.T) {
		var buf bytes.Buffer
		rep := capture(&buf)
		checkTmux(newSys(map[string]string{"HOME": home}, nil, nil), cfg, rep)
		if rep.Failures() != 1 {
			t.Fatalf("missing tmux should fail\n%s", buf.String())
		}
	})
}

func TestCheckOptionalTools_DotfDrift(t *testing.T) {
	cfg := &Config{Versions: map[string]string{"DOTF_VERSION": "0.2.0"}}
	var buf bytes.Buffer
	rep := capture(&buf)
	checkOptionalTools(
		newSys(nil, []string{"dotf", "gh"}, map[string]string{"dotf version": "dotf version 0.1.0"}),
		cfg, nil, rep)
	if rep.Failures() != 0 {
		t.Error("dotf version drift must WARN, not FAIL")
	}
	if !strings.Contains(buf.String(), "dotf version drift") {
		t.Errorf("expected dotf drift warning\n%s", buf.String())
	}
}

func TestCheckHarnessDrift(t *testing.T) {
	home := t.TempDir()
	dotfiles := filepath.Join(home, ".dotfiles")
	writeExec(t, filepath.Join(dotfiles, "scripts", "compile-harness.sh"))
	cfg := &Config{DotfilesDir: dotfiles}

	// compile-harness --check "passes" (bash returns the script's output, nil err).
	cmd := map[string]string{"bash " + filepath.Join(dotfiles, "scripts", "compile-harness.sh") + " --check": "ok"}
	var buf bytes.Buffer
	rep := capture(&buf)
	checkHarnessDrift(newSys(map[string]string{"HOME": home}, nil, cmd), cfg, rep)
	if rep.Failures() != 0 {
		t.Fatalf("compile-harness --check passing should not fail\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "no drift") {
		t.Error("expected the no-drift pass line")
	}
}

func TestRun_ExitContract(t *testing.T) {
	home := t.TempDir()
	dotfiles := filepath.Join(home, ".dotfiles")
	// Minimal contract + versions so loadConfig + loadContract succeed.
	writeFile(t, filepath.Join(dotfiles, "env-contract.json"),
		`{"env_vars":[],"required_path_entries":{"linux":[]},"required_binaries":[{"name":"git","required":true}],"optional_binaries":[]}`)
	writeFile(t, filepath.Join(dotfiles, "versions.conf"), "GO_VERSION=1.26.0\n")

	env := map[string]string{"HOME": home, "DOTFILES_DIR": dotfiles}
	// git missing on PATH → at least one FAIL → exit 1.
	sys := newSys(env, nil, nil)

	var buf bytes.Buffer
	code, err := Run(Options{Out: &buf, System: sys, StartDir: home, Verbose: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 1 {
		t.Fatalf("expected exit 1 with a failing check, got %d\n%s", code, buf.String())
	}
	if !strings.Contains(buf.String(), "Results:") {
		t.Error("summary line missing")
	}
	if !strings.Contains(buf.String(), "env-contract.json loaded") {
		t.Error("contract should load from DOTFILES_DIR")
	}
}

func TestRun_QuickSkipsHeavySections(t *testing.T) {
	home := t.TempDir()
	dotfiles := filepath.Join(home, ".dotfiles")
	// Contract that passes trivially (no required vars/paths/binaries).
	writeFile(t, filepath.Join(dotfiles, "env-contract.json"),
		`{"env_vars":[],"required_path_entries":{"linux":[]},"required_binaries":[],"optional_binaries":[]}`)
	writeFile(t, filepath.Join(dotfiles, "versions.conf"), "GO_VERSION=1.26.0\n")
	env := map[string]string{"HOME": home, "DOTFILES_DIR": dotfiles}
	// Nothing on PATH, no vault/secrets/tmux — the full sweep's healthcheck
	// sections all FAIL; the contract sweep (empty) passes.
	sys := newSys(env, nil, nil)

	// Full mode: heavy sections run and fail → exit 1.
	var full bytes.Buffer
	fullCode, _ := Run(Options{Out: &full, System: sys, StartDir: home, Verbose: true})
	if fullCode != 1 {
		t.Fatalf("full mode should fail (missing tools/vault/...), got %d", fullCode)
	}
	if !strings.Contains(full.String(), "Core tools in PATH") {
		t.Error("full mode must run the core-tools section")
	}

	// Quick mode: contract-only → exit 0, heavy section headers absent.
	var quick bytes.Buffer
	quickCode, _ := Run(Options{Out: &quick, System: sys, StartDir: home, Verbose: true, Quick: true})
	if quickCode != 0 {
		t.Fatalf("quick mode should pass (contract-only), got %d\n%s", quickCode, quick.String())
	}
	for _, heavy := range []string{"Core tools in PATH", "Harness + skill drift", "Antigravity CLI health", "Knowledge vault"} {
		if strings.Contains(quick.String(), heavy) {
			t.Errorf("quick mode must skip the %q section", heavy)
		}
	}
	if !strings.Contains(quick.String(), "[quick]") {
		t.Error("quick mode header should announce [quick]")
	}
}

// TestCheckOpenCode_piPathResilience guards the Orca / GUI per-node-version PATH
// trap: pi installed under an nvm node version not on the current PATH. The
// doctor must FAIL loudly (not SKIP "pi not installed") when pi is configured
// or present at the ~/.local launcher yet unreachable, and PASS/SKIP correctly
// otherwise. Pairs with the setup-linux.sh `--prefix ~/.local` install (#426).
func TestCheckOpenCode_piPathResilience(t *testing.T) {
	cfg := &Config{Versions: map[string]string{"PI_VERSION": "0.79.1", "OPENCODE_VERSION": "1.0.0"}}

	t.Run("configured but not on PATH fails loud", func(t *testing.T) {
		home := t.TempDir()
		writeFile(t, filepath.Join(home, ".pi", "agent", "models.json"), `{"models":{}}`)
		var buf bytes.Buffer
		checkOpenCode(newSys(map[string]string{"HOME": home}, nil, nil), cfg, capture(&buf))
		out := buf.String()
		if !strings.Contains(out, "pi configured (~/.pi present) but not on PATH") {
			t.Errorf("configured-but-unreachable pi must FAIL with the root cause\n%s", out)
		}
		if strings.Contains(out, "pi not installed") {
			t.Errorf("must not emit the misleading 'pi not installed' SKIP when pi is configured\n%s", out)
		}
	})

	t.Run("present at ~/.local launcher but not on PATH", func(t *testing.T) {
		home := t.TempDir()
		writeExec(t, filepath.Join(home, ".local", "bin", "pi"))
		var buf bytes.Buffer
		checkOpenCode(newSys(map[string]string{"HOME": home}, nil, nil), cfg, capture(&buf))
		if out := buf.String(); !strings.Contains(out, "pi exists at") || !strings.Contains(out, "but not in PATH") {
			t.Errorf("pi at ~/.local but off PATH must FAIL (reload shell)\n%s", out)
		}
	})

	t.Run("on PATH passes", func(t *testing.T) {
		home := t.TempDir()
		var buf bytes.Buffer
		checkOpenCode(newSys(map[string]string{"HOME": home}, []string{"pi"},
			map[string]string{"pi --version": "pi 0.79.1"}), cfg, capture(&buf))
		if out := buf.String(); !strings.Contains(out, "pi in PATH:") {
			t.Errorf("pi on PATH should PASS\n%s", out)
		}
	})

	t.Run("truly absent skips", func(t *testing.T) {
		home := t.TempDir()
		var buf bytes.Buffer
		checkOpenCode(newSys(map[string]string{"HOME": home}, nil, nil), cfg, capture(&buf))
		if out := buf.String(); !strings.Contains(out, "pi not installed") {
			t.Errorf("truly-absent pi should SKIP with install hint\n%s", out)
		}
	})
}
