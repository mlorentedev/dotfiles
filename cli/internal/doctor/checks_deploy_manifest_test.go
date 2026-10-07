package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Four entries, one per shape the check must tell apart: a merge entry, a
// replace entry, one gated on a command, one rendered.
const deployManifestFixture = `{"version":3,"configs":[
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
	checkDeployManifest(sys, rep, false)
	return buf.String()
}

func assertNoDir(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the check created %s — a diagnostic must not deploy", path)
	}
}

// An entry declared for other OSes is not compared: on Windows a POSIX rc file
// that is absent is not drift (#1843 B1).
func TestCheckDeployManifest_AnEntryForAnotherOSIsNotCompared(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "ai", "deploy.json"), `{"version":4,"configs":[
  {"name":"r","src":"ai/r.json","dst":"{HOME}/.r/config.json"},
  {"name":"posix","src":"ai/p.json","dst":"{HOME}/.p/rc","platforms":["linux","darwin"]}
]}`)
	writeFile(t, filepath.Join(repo, "ai", "r.json"), `{"r":true}`)
	writeFile(t, filepath.Join(repo, "ai", "p.json"), `x`)
	writeFile(t, filepath.Join(home, ".r", "config.json"), `{"r":true}`)

	sys := newSys(map[string]string{"HOME": home, "USERPROFILE": home, "DOTFILES_REPO_DIR": repo}, nil, nil)
	sys.GOOS = "windows"
	var buf bytes.Buffer
	checkDeployManifest(sys, NewReport(&buf, true), false)
	out := buf.String()
	if strings.Contains(out, "dotf deploy posix") || !strings.Contains(out, "1 not compared") {
		t.Errorf("the POSIX-only entry must not be compared on windows:\n%s", out)
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
		writeFile(t, filepath.Join(home, ".p", "models.json"), `{"k":"rendered"}`)

		out := runCheckDeployManifest(t, repo, home, nil)
		if got := statusOfLine(out, "in sync"); got != StatusPass {
			t.Errorf("want PASS, got %v\n%s", got, out)
		}
		if !strings.Contains(out, "2 deployed config(s) in sync") || !strings.Contains(out, "2 not compared") {
			t.Errorf("the PASS line must count compared and not-compared entries:\n%s", out)
		}
		assertNoDir(t, filepath.Join(home, ".g"))
	})

	// #2100: a rendered entry's content is not comparable, its absence is.
	t.Run("rendered entry that was never deployed → WARN naming the remedy", func(t *testing.T) {
		repo, home := deployManifestRepo(t), t.TempDir()
		writeFile(t, filepath.Join(home, ".m", "settings.json"), `{"model":"m","autoUpdate":false}`)
		writeFile(t, filepath.Join(home, ".r", "config.json"), `{"r":true}`)

		out := runCheckDeployManifest(t, repo, home, nil)
		if got := statusOfLine(out, "dotf deploy p"); got != StatusWarn {
			t.Errorf("want WARN for the undeployed rendered p, got %v\n%s", got, out)
		}
		if !strings.Contains(out, "models.json not deployed") {
			t.Errorf("the WARN must say the file is not deployed:\n%s", out)
		}
		if statusOfLine(out, "in sync") == StatusPass {
			t.Errorf("a missing rendered config must not PASS:\n%s", out)
		}
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

// checkTmux no longer compares ~/.tmux.conf: the `tmux` deploy entry owns it
// and this check is the only thing that would report its drift. So the guard is
// on the shipped manifest, end to end: a ~/.tmux.conf that differs from the
// repo's tmux.conf is a drift WARN naming `dotf deploy tmux`. Without the entry
// (or if the check were gated behind something that skips it) doctor would
// silently stop reporting tmux.conf drift on every OS.
func TestCheckDeployManifest_ReportsTmuxConfDriftFromTheShippedManifest(t *testing.T) {
	repo, home := repoRootForDoctorTest(t), t.TempDir()
	writeFile(t, filepath.Join(home, ".tmux.conf"), "DIFFERENT\n")

	out := runCheckDeployManifest(t, repo, home, []string{"tmux"})
	if got := statusOfLine(out, "drift: tmux "); got != StatusWarn {
		t.Errorf("a differing ~/.tmux.conf must be a drift WARN from the shipped manifest: got %v\n%s", got, out)
	}
}

// A directory the manifest deploys a private file into, left open from before
// the deploy created it 0700, is reported; --fix tightens it (#2053).
func TestCheckDeployManifest_ReportsAndFixesAnOpenPrivateDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful here")
	}
	repo, home := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "ai", "deploy.json"), `{"version":4,"configs":[
  {"name":"ssh-config","src":"ai/c","dst":"{HOME}/.ssh/config","mode":"0600"},
  {"name":"ssh-pubkey","src":"ai/k","dst":"{HOME}/.ssh/id.pub","mode":"0644"},
  {"name":"rc","src":"ai/k","dst":"{HOME}/.tool/rc","mode":"0644"}
]}`)
	writeFile(t, filepath.Join(repo, "ai", "c"), "c\n")
	writeFile(t, filepath.Join(repo, "ai", "k"), "k\n")
	ssh, tool := filepath.Join(home, ".ssh"), filepath.Join(home, ".tool")
	for _, d := range []string{ssh, tool} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(d, 0o755); err != nil { // umask-independent
			t.Fatal(err)
		}
	}
	// Deployed, from before the directory mode was derived: the private file
	// sits in an open directory.
	writeFile(t, filepath.Join(ssh, "config"), "c\n")
	sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil)

	var buf bytes.Buffer
	checkDeployManifest(sys, NewReport(&buf, true), false)
	if !strings.Contains(buf.String(), ssh+" is 0755 but holds a private deployed file") {
		t.Errorf("an open ~/.ssh must be reported\n%s", buf.String())
	}
	if strings.Contains(buf.String(), tool+" is") {
		t.Errorf("a directory with only public files must not be reported\n%s", buf.String())
	}
	if info, _ := os.Stat(ssh); info.Mode().Perm() != 0o755 {
		t.Errorf("without --fix the check must not change the mode, got %o", info.Mode().Perm())
	}

	buf.Reset()
	checkDeployManifest(sys, NewReport(&buf, true), true)
	if info, _ := os.Stat(ssh); info.Mode().Perm() != 0o700 {
		t.Errorf("--fix must tighten ~/.ssh to 0700, got %o\n%s", info.Mode().Perm(), buf.String())
	}
	if info, _ := os.Stat(tool); info.Mode().Perm() != 0o755 {
		t.Errorf("--fix must leave a public directory alone, got %o", info.Mode().Perm())
	}

	buf.Reset()
	checkDeployManifest(sys, NewReport(&buf, true), true)
	if strings.Contains(buf.String(), "tightened") {
		t.Errorf("a second --fix must find nothing to do\n%s", buf.String())
	}
}

// The target-OS seam: a Windows target has no POSIX mode bits to judge.
func TestCheckDeployManifest_SkipsDirectoryModesForAWindowsTarget(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "ai", "deploy.json"), `{"version":4,"configs":[
  {"name":"ssh-config","src":"ai/c","dst":"{HOME}/.ssh/config","mode":"0600"}
]}`)
	writeFile(t, filepath.Join(repo, "ai", "c"), "c\n")
	if err := os.Mkdir(filepath.Join(home, ".ssh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(home, ".ssh"), 0o755); err != nil {
		t.Fatal(err)
	}
	sys := newSys(map[string]string{"HOME": home, "USERPROFILE": home, "DOTFILES_REPO_DIR": repo}, nil, nil)
	sys.GOOS = "windows"
	var buf bytes.Buffer
	checkDeployManifest(sys, NewReport(&buf, true), false)
	if strings.Contains(buf.String(), "holds a private deployed file") {
		t.Errorf("a Windows target must not be judged by POSIX mode bits\n%s", buf.String())
	}
}

// A private entry that does not apply here deployed nothing, so its directory
// is not reported however open it is (#2098 review).
func TestCheckDeployManifest_IgnoresAPrivateEntryThatDoesNotApplyHere(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful here")
	}
	repo, home := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "ai", "deploy.json"), `{"version":4,"configs":[
  {"name":"other-os","src":"ai/c","dst":"{HOME}/.app/config","mode":"0600","platforms":["windows"]},
  {"name":"gated","src":"ai/c","dst":"{HOME}/.gated/config","mode":"0600","requires":"absenttool"}
]}`)
	writeFile(t, filepath.Join(repo, "ai", "c"), "c\n")
	for _, f := range []string{".app/config", ".gated/config"} {
		writeFile(t, filepath.Join(home, filepath.FromSlash(f)), "c\n")
		if err := os.Chmod(filepath.Dir(filepath.Join(home, filepath.FromSlash(f))), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil)
	sys.GOOS = "darwin"
	var buf bytes.Buffer
	checkDeployManifest(sys, NewReport(&buf, true), true)
	if strings.Contains(buf.String(), "private deployed file") || strings.Contains(buf.String(), "tightened") {
		t.Errorf("an entry that does not apply here must not mark its directory\n%s", buf.String())
	}
}
