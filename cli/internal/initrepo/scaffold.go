package initrepo

import (
	"os"
	"path/filepath"
	"strings"
)

// scaffoldDirs is the knowledge-placement repo structure dotf init lays down
// (ADR-022 / pattern-knowledge-placement): build/operate docs live in docs/,
// specs in specs/, per-agent config in .claude/.
var scaffoldDirs = []string{
	"src", "tests", "scripts", "specs",
	"docs",
	filepath.Join("docs", "adr"),
	filepath.Join("docs", "lessons"),
	filepath.Join("docs", "runbooks"),
	filepath.Join("docs", "troubleshooting"),
	".claude",
}

// staticFiles maps an embedded template to its repo-relative destination. Source
// template names are chosen so the file is actually committed AND embedded: they
// carry no leading dot (go:embed skips dot/underscore files, so gitignore ->
// .gitignore), and they dodge the repo's own .gitignore (claude-md -> CLAUDE.md,
// else the root `CLAUDE.md` ignore rule drops the template from a fresh checkout
// and embed open fails at runtime). Each write is skip-if-present.
var staticFiles = []struct{ template, dest string }{
	{"gitignore", ".gitignore"},
	{"pre-commit-config.yaml", ".pre-commit-config.yaml"},
	{"claude-md", "CLAUDE.md"},
	{"env-contract.json", "env-contract.json"},
	{"lessons-index.md", filepath.Join("docs", "lessons", "_index.md")},
}

// ScaffoldResult reports what Scaffold wrote, in repo-relative paths.
type ScaffoldResult struct {
	Created []string // files written this run
	Skipped []string // files left untouched because they already existed
}

// Scaffold lays down the placement-model directory structure and the static
// practice-stack files under root (which should be an absolute path),
// substituting {{repo}} with the repo basename. Directories are created
// idempotently (mkdir -p); files are skip-if-present so a re-run never clobbers
// project content. AGENTS.md/SDD wiring, stack init, git, and the vault entry are
// layered on top by the orchestrator.
func Scaffold(root string) (ScaffoldResult, error) {
	var res ScaffoldResult

	for _, d := range scaffoldDirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return res, err
		}
	}

	repo := filepath.Base(root)
	for _, f := range staticFiles {
		dest := filepath.Join(root, f.dest)
		if _, err := os.Stat(dest); err == nil {
			res.Skipped = append(res.Skipped, f.dest)
			continue
		}
		raw, err := ReadTemplate(f.template)
		if err != nil {
			return res, err
		}
		content := strings.ReplaceAll(string(raw), "{{repo}}", repo)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return res, err
		}
		if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			return res, err
		}
		res.Created = append(res.Created, f.dest)
	}
	return res, nil
}
