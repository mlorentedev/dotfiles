package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckDeployedSkillSymlinks drives the BUG-100 narrowing: a symlink at a
// name this repo's harness does NOT manage (e.g. Orca's ~/.agents/skills
// mechanism, or pi's sibling-skill links) must not fail doctor. Only a symlink
// shadowing a harness/skills/<name> record is a real BUG-100 regression.
func TestCheckDeployedSkillSymlinks(t *testing.T) {
	newEnv := func(t *testing.T) (home, mirror string, cfg *Config, sys *System) {
		t.Helper()
		home = t.TempDir()
		mirror = t.TempDir()
		mkdirAll(t, filepath.Join(mirror, "harness", "skills", "orca-cli"))
		cfg = &Config{DotfilesDir: mirror}
		sys = newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": mirror}, nil, nil)
		return
	}

	t.Run("symlinked dir at an unmanaged name is silent", func(t *testing.T) {
		home, _, cfg, sys := newEnv(t)
		foreignTarget := t.TempDir()
		mustSymlink(t, foreignTarget, filepath.Join(home, ".claude", "skills", "computer-use"))

		var buf bytes.Buffer
		rep := capture(&buf)
		checkDeployedSkillSymlinks(sys, cfg, rep)

		if rep.Failures() != 0 {
			t.Fatalf("unmanaged-name symlink must not fail\n%s", buf.String())
		}
		if !strings.Contains(buf.String(), "no symlinks at managed skill names") {
			t.Errorf("expected the narrowed pass line\n%s", buf.String())
		}
	})

	t.Run("symlinked dir at a managed name fails and names the path", func(t *testing.T) {
		home, _, cfg, sys := newEnv(t)
		foreignTarget := t.TempDir()
		managedPath := filepath.Join(home, ".claude", "skills", "orca-cli")
		mustSymlink(t, foreignTarget, managedPath)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkDeployedSkillSymlinks(sys, cfg, rep)

		if rep.Failures() == 0 {
			t.Fatalf("managed-name symlink must fail\n%s", buf.String())
		}
		if !strings.Contains(buf.String(), managedPath) {
			t.Errorf("expected the flagged path in output\n%s", buf.String())
		}
	})

	t.Run("symlinked SKILL.md one level inside a managed dir fails", func(t *testing.T) {
		home, _, cfg, sys := newEnv(t)
		realDir := filepath.Join(home, ".claude", "skills", "orca-cli")
		mkdirAll(t, realDir)
		foreignFile := filepath.Join(t.TempDir(), "SKILL.md")
		if err := os.WriteFile(foreignFile, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		mustSymlink(t, foreignFile, filepath.Join(realDir, "SKILL.md"))

		var buf bytes.Buffer
		rep := capture(&buf)
		checkDeployedSkillSymlinks(sys, cfg, rep)

		if rep.Failures() == 0 {
			t.Fatalf("symlinked SKILL.md at a managed name must fail\n%s", buf.String())
		}
	})

	t.Run("symlinked command file at an unmanaged name is silent, managed one fails", func(t *testing.T) {
		home, _, cfg, sys := newEnv(t)
		foreignFile := filepath.Join(t.TempDir(), "src.md")
		mustSymlink(t, foreignFile, filepath.Join(home, ".config", "opencode", "commands", "find-skills.md"))
		mustSymlink(t, foreignFile, filepath.Join(home, ".config", "opencode", "commands", "orca-cli.md"))

		var buf bytes.Buffer
		rep := capture(&buf)
		checkDeployedSkillSymlinks(sys, cfg, rep)

		// header line + the one flagged path (findSymlinks' header counts as a
		// Fail too, same as the pre-existing behavior this test guards).
		if rep.Failures() != 2 {
			t.Fatalf("expected 2 failures (header + orca-cli.md only), got %d\n%s", rep.Failures(), buf.String())
		}
		if strings.Contains(buf.String(), "find-skills.md") {
			t.Errorf("unmanaged command symlink must not appear in output\n%s", buf.String())
		}
		if !strings.Contains(buf.String(), "orca-cli.md") {
			t.Errorf("expected managed command symlink in output\n%s", buf.String())
		}
	})

	t.Run("no deployed skill paths -> skip", func(t *testing.T) {
		home := t.TempDir()
		mirror := t.TempDir()
		cfg := &Config{DotfilesDir: mirror}
		sys := newSys(map[string]string{"HOME": home}, nil, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkDeployedSkillSymlinks(sys, cfg, rep)

		if rep.Failures() != 0 || !strings.Contains(buf.String(), "no deployed skill paths found") {
			t.Errorf("expected a skip\n%s", buf.String())
		}
	})
}
