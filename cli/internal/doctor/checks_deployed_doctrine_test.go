package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckDeployedDoctrine(t *testing.T) {
	setup := func(t *testing.T) (repo, home string) {
		t.Helper()
		repo = t.TempDir()
		home = t.TempDir()
		return repo, home
	}

	validDoctrineContent := `<!-- BEGIN HARNESS GENERATED (sha256:123) -->
## Non-negotiable rules
- No AI attribution in git history
- English only in git/GitHub artifacts
- No internal phase/milestone references
- Auto-merge is forbidden in every repository
- Working code is not a finished change
- What binds is the disposition, not the waiting (PR Stewardship)
- Atomic PRs, ~300 LOC hard cap
<!-- END HARNESS GENERATED -->
`

	t.Run("all deployed surfaces present and contain enforced regions -> pass", func(t *testing.T) {
		repo, home := setup(t)
		writeFile(t, filepath.Join(home, ".gemini", "GEMINI.md"), validDoctrineContent)
		writeFile(t, filepath.Join(home, ".codex", "AGENTS.md"), validDoctrineContent)
		writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), validDoctrineContent)

		sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil)
		cfg := &Config{DotfilesDir: repo}

		var buf bytes.Buffer
		rep := NewReport(&buf, true)

		checkDeployedDoctrine(sys, cfg, rep)

		out := buf.String()
		if strings.Contains(out, "FAIL") {
			t.Errorf("expected pass, got failures:\n%s", out)
		}
		if !strings.Contains(out, "deployed doctrine payloads contain all enforced regions") {
			t.Errorf("expected pass message, got:\n%s", out)
		}
	})

	t.Run("missing enforced region in deployed surface -> fail naming region and surface", func(t *testing.T) {
		repo, home := setup(t)
		// Missing "No AI attribution" and "Auto-merge is forbidden"
		badContent := `<!-- BEGIN HARNESS GENERATED (sha256:123) -->
- English only in git/GitHub artifacts
- No internal phase/milestone references
- Working code is not a finished change
- What binds is the disposition, not the waiting (PR Stewardship)
<!-- END HARNESS GENERATED -->
`
		writeFile(t, filepath.Join(home, ".gemini", "GEMINI.md"), badContent)

		sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil)
		cfg := &Config{DotfilesDir: repo}

		var buf bytes.Buffer
		rep := NewReport(&buf, true)

		checkDeployedDoctrine(sys, cfg, rep)

		out := buf.String()
		if !strings.Contains(out, "FAIL") {
			t.Errorf("expected failure when enforced region is missing, got:\n%s", out)
		}
		if !strings.Contains(out, "no-attribution") || !strings.Contains(out, ".gemini/GEMINI.md") {
			t.Errorf("expected error to name missing region 'no-attribution' and surface '.gemini/GEMINI.md', got:\n%s", out)
		}
	})
}

func TestCheckDeployedDoctrine_MissingRegion(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()

	// Incomplete deployed file
	writeFile(t, filepath.Join(home, ".codex", "AGENTS.md"), "<!-- BEGIN HARNESS GENERATED -->\nDefinition of Done\n<!-- END -->\n")

	sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil)
	cfg := &Config{DotfilesDir: repo}

	var buf bytes.Buffer
	rep := NewReport(&buf, true)

	checkDeployedDoctrine(sys, cfg, rep)

	out := buf.String()
	if !strings.Contains(out, "FAIL") {
		t.Fatalf("expected FAIL on missing doctrine regions, got:\n%s", out)
	}
	if !strings.Contains(out, "no-attribution") || !strings.Contains(out, ".codex/AGENTS.md") {
		t.Errorf("expected FAIL to cite missing 'no-attribution' in '.codex/AGENTS.md', got:\n%s", out)
	}
}
