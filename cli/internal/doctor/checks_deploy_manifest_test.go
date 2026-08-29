package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Four entries, one per shape the check must tell apart: a merge entry, a
// replace entry, one gated on a command, one rendered.
const deployManifestFixture = `{"version":2,"configs":[
  {"name":"m","src":"ai/m.json","dst":"{HOME}/.m/settings.json","render":false,"mode":"0644","strategy":"merge"},
  {"name":"r","src":"ai/r.json","dst":"{HOME}/.r/config.json","render":false,"mode":"0644"},
  {"name":"g","src":"ai/g.json","dst":"{HOME}/.g/x.json","render":false,"mode":"0644","requires":"gatedtool"},
  {"name":"p","src":"ai/p.json","dst":"{HOME}/.p/models.json","render":true,"mode":"0600"}
]}`

func deployManifestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "ai", "deploy.json"), deployManifestFixture)
	writeFile(t, filepath.Join(repo, "ai", "m.json"), `{"model":"m","autoUpdate":false}`)
	writeFile(t, filepath.Join(repo, "ai", "r.json"), `{"r":true}`)
	writeFile(t, filepath.Join(repo, "ai", "g.json"), `{"g":true}`)
	writeFile(t, filepath.Join(repo, "ai", "p.json"), `{"k":"${X}"}`)
	return repo
}

func runCheckDeployManifest(t *testing.T, repo, home string, onPath []string) string {
	t.Helper()
	env := map[string]string{"HOME": home, "USERPROFILE": home}
	if repo != "" {
		env["DOTFILES_REPO_DIR"] = repo
	}
	sys := newSys(env, onPath, nil)
	var buf bytes.Buffer
	rep := NewReport(&buf, true) // verbose: PASS lines are printed, so status can be asserted
	checkDeployManifest(sys, rep)
	return buf.String()
}

func assertNoDir(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the check created %s — a diagnostic must not deploy", path)
	}
}

// AC4 (AI-039, #1322): the manifest check reports by status — PASS when every
// compared entry is in sync (and says how many it did not compare), WARN naming
// the drifted entry and `dotf deploy <name>`, SKIP without a repo — and never
// creates a destination directory while asking.
func TestCheckDeployManifest_ByStatus(t *testing.T) {
	t.Run("all compared entries in sync → PASS; gated and rendered not compared", func(t *testing.T) {
		repo, home := deployManifestRepo(t), t.TempDir()
		writeFile(t, filepath.Join(home, ".m", "settings.json"), `{"effortLevel":"max","model":"m","autoUpdate":false}`)
		writeFile(t, filepath.Join(home, ".r", "config.json"), `{"r":true}`)

		out := runCheckDeployManifest(t, repo, home, nil)
		if got := statusOfLine(out, "in sync"); got != StatusPass {
			t.Errorf("want PASS, got %v\n%s", got, out)
		}
		if !strings.Contains(out, "2 deployed config(s) in sync") || !strings.Contains(out, "2 not compared") {
			t.Errorf("the PASS line must count compared and not-compared entries:\n%s", out)
		}
		assertNoDir(t, filepath.Join(home, ".g"))
		assertNoDir(t, filepath.Join(home, ".p"))
	})

	t.Run("merge entry with a differing managed key → WARN naming it and the remedy", func(t *testing.T) {
		repo, home := deployManifestRepo(t), t.TempDir()
		writeFile(t, filepath.Join(home, ".m", "settings.json"), `{"effortLevel":"max","model":"other","autoUpdate":false}`)
		writeFile(t, filepath.Join(home, ".r", "config.json"), `{"r":true}`)

		out := runCheckDeployManifest(t, repo, home, nil)
		if got := statusOfLine(out, "dotf deploy m"); got != StatusWarn {
			t.Errorf("want WARN naming the remedy for m, got %v\n%s", got, out)
		}
		if strings.Contains(out, "dotf deploy r") {
			t.Errorf("r is in sync and must not be named:\n%s", out)
		}
		if statusOfLine(out, "in sync") == StatusPass {
			t.Errorf("a drifted manifest must not PASS:\n%s", out)
		}
	})

	t.Run("replace entry that differs → WARN naming it", func(t *testing.T) {
		repo, home := deployManifestRepo(t), t.TempDir()
		writeFile(t, filepath.Join(home, ".m", "settings.json"), `{"model":"m","autoUpdate":false}`)
		writeFile(t, filepath.Join(home, ".r", "config.json"), `{"r":false}`)

		out := runCheckDeployManifest(t, repo, home, nil)
		if got := statusOfLine(out, "dotf deploy r"); got != StatusWarn {
			t.Errorf("want WARN for r, got %v\n%s", got, out)
		}
	})

	t.Run("absent destinations → WARN, and no directory is created by asking", func(t *testing.T) {
		repo, home := deployManifestRepo(t), t.TempDir()

		out := runCheckDeployManifest(t, repo, home, nil)
		for _, name := range []string{"m", "r"} {
			if got := statusOfLine(out, "dotf deploy "+name); got != StatusWarn {
				t.Errorf("want WARN for absent %s, got %v\n%s", name, got, out)
			}
		}
		assertNoDir(t, filepath.Join(home, ".m"))
		assertNoDir(t, filepath.Join(home, ".r"))
	})

	t.Run("gated entry is compared once its command is on PATH", func(t *testing.T) {
		repo, home := deployManifestRepo(t), t.TempDir()
		writeFile(t, filepath.Join(home, ".m", "settings.json"), `{"model":"m","autoUpdate":false}`)
		writeFile(t, filepath.Join(home, ".r", "config.json"), `{"r":true}`)

		out := runCheckDeployManifest(t, repo, home, []string{"gatedtool"})
		if got := statusOfLine(out, "dotf deploy g"); got != StatusWarn {
			t.Errorf("with gatedtool present, absent g is drift: got %v\n%s", got, out)
		}
		assertNoDir(t, filepath.Join(home, ".g"))
	})

	t.Run("no repo → SKIP", func(t *testing.T) {
		t.Chdir(t.TempDir()) // the cwd walk-up must not find the real checkout
		out := runCheckDeployManifest(t, "", t.TempDir(), nil)
		if got := statusOfLine(out, "repo not found"); got != StatusSkip {
			t.Errorf("want SKIP, got %v\n%s", got, out)
		}
	})
}
