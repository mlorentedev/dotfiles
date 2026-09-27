package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// piRepo lays down a manifest in a temp checkout and a live settings file in a
// temp agent dir, and pins the seams so no test runs pi or reads PATH.
func piRepo(t *testing.T, manifest, live string) (agentDir string, calls *[]string) {
	t.Helper()
	root := makeRepo(t)
	t.Setenv("DOTFILES_REPO_DIR", root)
	t.Setenv(skipEnv, "")
	t.Setenv("HOME", t.TempDir())
	writeFileAt(t, filepath.Join(root, "ai", "pi", "packages.json"), manifest)
	agentDir = t.TempDir()
	writeFileAt(t, filepath.Join(agentDir, "settings.json"), live)

	var got []string
	origRun, origLook := piRun, piLookPath
	t.Cleanup(func() { piRun, piLookPath = origRun, origLook })
	piRun = func(_ string, args ...string) (string, error) {
		got = append(got, strings.Join(args, " "))
		return "", nil
	}
	piLookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	return agentDir, &got
}

func writeFileAt(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

const piManifest = `{"version":1,"packages":[{"source":"npm:keep@1","why":"x"},{"source":"npm:new@1","why":"y"}]}`

func TestPiPackagesCheckExitsNonZeroNamingDrift(t *testing.T) {
	agentDir, _ := piRepo(t, piManifest, `{"packages":["npm:keep@1","npm:gone@2"]}`)
	out, _, err := execute(t, "pi", "packages", "check", "--agent-dir", agentDir)
	if err == nil || !strings.Contains(out, "remove   npm:gone@2") || !strings.Contains(out, "install  npm:new@1") {
		t.Fatalf("drift must exit non-zero and name both directions: err=%v\n%s", err, out)
	}
}

func TestPiPackagesCheckRefusesAnUnreadableManifest(t *testing.T) {
	agentDir, _ := piRepo(t, `{"packages":[]}`, `{"packages":[]}`)
	if _, _, err := execute(t, "pi", "packages", "check", "--agent-dir", agentDir); err == nil {
		t.Fatal("an empty manifest must never read as a clean state")
	}
}

func TestPiPackagesApplyDryRunCallsNothing(t *testing.T) {
	agentDir, calls := piRepo(t, piManifest, `{"packages":["npm:keep@1","npm:gone@2"]}`)
	out, _, err := execute(t, "pi", "packages", "apply", "--dry-run", "--agent-dir", agentDir)
	if err != nil || len(*calls) != 0 || !strings.Contains(out, "1 to remove, 1 to install") {
		t.Fatalf("a dry run plans and calls nothing: err=%v calls=%v\n%s", err, *calls, out)
	}
}

func TestPiPackagesApplyRemovesThenInstalls(t *testing.T) {
	agentDir, calls := piRepo(t, piManifest, `{"packages":["npm:keep@1","npm:gone@2"]}`)
	bin := filepath.Join(t.TempDir(), "pi")
	writeFileAt(t, bin, "")
	out, _, err := execute(t, "pi", "packages", "apply", "--pi", bin, "--agent-dir", agentDir)
	if err != nil || strings.Join(*calls, ";") != "remove npm:gone@2;install npm:new@1" {
		t.Fatalf("want remove then install: err=%v calls=%v\n%s", err, *calls, out)
	}
	if !strings.Contains(out, "changed=2") {
		t.Errorf("the summary says what changed:\n%s", out)
	}
}

func TestPiPackagesApplyFailureExitsNonZero(t *testing.T) {
	agentDir, _ := piRepo(t, piManifest, `{"packages":["npm:keep@1"]}`)
	piRun = func(string, ...string) (string, error) { return "npm ERR!", errors.New("exit status 1") }
	bin := filepath.Join(t.TempDir(), "pi")
	writeFileAt(t, bin, "")
	if out, _, err := execute(t, "pi", "packages", "apply", "--pi", bin, "--agent-dir", agentDir); err == nil {
		t.Fatalf("a failed install must not exit 0:\n%s", out)
	}
}

// AC4: the skip comes first, before any probe, even with no manifest at all.
func TestPiPackagesApplySkipIsFirstAndLoud(t *testing.T) {
	agentDir, calls := piRepo(t, piManifest, `{"packages":[]}`)
	t.Setenv(skipEnv, "1")
	t.Setenv("DOTFILES_REPO_DIR", t.TempDir())
	out, _, err := execute(t, "pi", "packages", "apply", "--agent-dir", agentDir)
	if err != nil || len(*calls) != 0 || !strings.Contains(out, "nothing installed, nothing verified") {
		t.Fatalf("want a loud skip with exit 0: err=%v\n%s", err, out)
	}
}

// AC4: a machine without pi degrades to a warning, never a failed setup.
func TestPiPackagesApplyWithoutPiWarnsAndExitsZero(t *testing.T) {
	agentDir, calls := piRepo(t, piManifest, `{"packages":[]}`)
	piLookPath = func(string) (string, error) { return "", errors.New("not found") }
	out, _, err := execute(t, "pi", "packages", "apply", "--agent-dir", agentDir)
	if err != nil || len(*calls) != 0 || !strings.Contains(out, "pi not installed") {
		t.Fatalf("want a warning and exit 0: err=%v\n%s", err, out)
	}
}

func TestPiPackagesApplyWithoutNpmWarnsAndExitsZero(t *testing.T) {
	agentDir, calls := piRepo(t, piManifest, `{"packages":[]}`)
	piLookPath = func(name string) (string, error) {
		if name == "npm" {
			return "", errors.New("not found")
		}
		return "/usr/bin/pi", nil
	}
	out, _, err := execute(t, "pi", "packages", "apply", "--agent-dir", agentDir)
	if err != nil || len(*calls) != 0 || !strings.Contains(out, "npm not found") {
		t.Fatalf("want a warning and exit 0: err=%v\n%s", err, out)
	}
}

// --repo wins over the cwd: setup passes its own checkout, so a cwd inside a
// different checkout can never pick the manifest that pi is reconciled on.
func TestPiPackagesRepoFlagWinsOverTheCwd(t *testing.T) {
	agentDir, _ := piRepo(t, `{"packages":[{"source":"npm:cwd@1","why":"x"}]}`, `{"packages":[]}`)
	declared := t.TempDir()
	writeFileAt(t, filepath.Join(declared, "ai", "pi", "packages.json"), `{"packages":[{"source":"npm:declared@1","why":"x"}]}`)
	out, _, _ := execute(t, "pi", "packages", "check", "--repo", declared, "--agent-dir", agentDir)
	if !strings.Contains(out, "install  npm:declared@1") || strings.Contains(out, "npm:cwd@1") {
		t.Fatalf("--repo must choose the manifest:\n%s", out)
	}
}
