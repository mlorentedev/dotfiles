package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckHarnessMirrorOrphans drives BUG-058/#843: a harness/{skills,agents}
// record deleted from the repo must not survive forever in the deploy mirror.
func TestCheckHarnessMirrorOrphans(t *testing.T) {
	setup := func(t *testing.T) (repo, mirror string) {
		t.Helper()
		repo = t.TempDir()
		mirror = t.TempDir()
		mkdirAll(t, filepath.Join(repo, "harness", "skills", "kept"))
		mkdirAll(t, filepath.Join(repo, "harness", "agents")) // present, possibly empty -- a real repo counterpart tree
		mkdirAll(t, filepath.Join(mirror, "harness", "skills", "kept"))
		return repo, mirror
	}

	t.Run("orphan record fails and names the path", func(t *testing.T) {
		repo, mirror := setup(t)
		mkdirAll(t, filepath.Join(mirror, "harness", "skills", "retired"))
		cfg := &Config{DotfilesDir: mirror}
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkHarnessMirrorOrphans(sys, cfg, rep, false)

		if rep.Failures() != 1 {
			t.Fatalf("want 1 failure, got %d\n%s", rep.Failures(), buf.String())
		}
		if !strings.Contains(buf.String(), filepath.Join("harness", "skills", "retired")) {
			t.Errorf("expected the orphan path in output\n%s", buf.String())
		}
		if !strings.Contains(buf.String(), "dotf doctor --fix") {
			t.Errorf("expected the --fix remedy in output\n%s", buf.String())
		}
		// non-fix mode must not touch disk
		if !isDir(filepath.Join(mirror, "harness", "skills", "retired")) {
			t.Error("check-only mode must not remove the orphan")
		}
	})

	t.Run("--fix prunes the orphan and is idempotent", func(t *testing.T) {
		repo, mirror := setup(t)
		mkdirAll(t, filepath.Join(mirror, "harness", "agents", "retired"))
		cfg := &Config{DotfilesDir: mirror}
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkHarnessMirrorOrphans(sys, cfg, rep, true)

		if rep.Failures() != 0 {
			t.Fatalf("fix mode should not report a failure\n%s", buf.String())
		}
		if isDir(filepath.Join(mirror, "harness", "agents", "retired")) {
			t.Error("--fix should have removed the orphan directory")
		}
		if !isDir(filepath.Join(mirror, "harness", "skills", "kept")) {
			t.Error("--fix must not touch a record that still has a repo counterpart")
		}

		// second run: nothing left to prune
		buf.Reset()
		rep2 := capture(&buf)
		checkHarnessMirrorOrphans(sys, cfg, rep2, true)
		if rep2.Failures() != 0 || !strings.Contains(buf.String(), "no orphan records") {
			t.Errorf("second --fix run should be a clean pass\n%s", buf.String())
		}
	})

	t.Run("no orphans -> pass", func(t *testing.T) {
		repo, mirror := setup(t)
		cfg := &Config{DotfilesDir: mirror}
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkHarnessMirrorOrphans(sys, cfg, rep, false)

		if rep.Failures() != 0 || !strings.Contains(buf.String(), "no orphan records") {
			t.Errorf("expected a clean pass\n%s", buf.String())
		}
	})

	t.Run("windows has no separate mirror -> no-op even with an orphan present", func(t *testing.T) {
		repo, mirror := setup(t)
		mkdirAll(t, filepath.Join(mirror, "harness", "skills", "retired"))
		cfg := &Config{DotfilesDir: mirror}
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, nil)
		sys.GOOS = "windows"

		var buf bytes.Buffer
		rep := capture(&buf)
		checkHarnessMirrorOrphans(sys, cfg, rep, false)

		if rep.Failures() != 0 || buf.Len() != 0 {
			t.Errorf("windows should be a silent no-op\n%s", buf.String())
		}
	})

	t.Run("mirror IS the repo checkout -> nothing to compare", func(t *testing.T) {
		repo := t.TempDir()
		mkdirAll(t, filepath.Join(repo, "harness", "skills", "kept"))
		cfg := &Config{DotfilesDir: repo}
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkHarnessMirrorOrphans(sys, cfg, rep, false)

		if rep.Failures() != 0 || buf.Len() != 0 {
			t.Errorf("repo==mirror should be a silent no-op\n%s", buf.String())
		}
	})

	t.Run("repo without harness/skills must not prune the mirror", func(t *testing.T) {
		// resolveRepoDir proves only "a git checkout", not "the dotfiles
		// checkout" -- a repo lacking harness/<sub> entirely (wrong repo
		// resolved, e.g. DOTFILES_REPO_DIR unset + doctor run from inside an
		// unrelated project) must not read every mirror entry as orphaned.
		repo := t.TempDir() // no harness/ tree at all
		mirror := t.TempDir()
		mkdirAll(t, filepath.Join(mirror, "harness", "skills", "kept"))
		cfg := &Config{DotfilesDir: mirror}
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, nil)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkHarnessMirrorOrphans(sys, cfg, rep, true)

		if !isDir(filepath.Join(mirror, "harness", "skills", "kept")) {
			t.Errorf("must not prune when the repo has no counterpart tree\n%s", buf.String())
		}
		if rep.Failures() != 0 {
			t.Errorf("a missing counterpart tree is a SKIP, not a FAIL\n%s", buf.String())
		}
	})
}
