package docpaths

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCheckFilesPathOutcomes(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "scripts/utils.sh", "ok\n")
	writeRepoFile(t, repo, "harness/skills/real/SKILL.md", "ok\n")
	if err := os.MkdirAll(filepath.Join(repo, "ai", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repo, "AGENTS.md", strings.Join([]string{
		"Existing: `scripts/utils.sh`.",
		"Existing glob: `harness/skills/*/SKILL.md`.",
		"Existing glob with trailing slash: `scripts/*.sh/`.",
		"Missing: `scripts/definitely-not-here.sh`.",
		"Empty glob: `ai/skills/*/SKILL.md`.",
		"Uppercase file: `harness/skills/missing/SKILL.md`.",
		"Escape: `scripts/../../outside/README.md`.",
		"Ignore unrooted escape: `not-a-real-toplevel/../other/thing.md`.",
		"Ignore Git internals: `.git/info/exclude`.",
		"Ignore model: `opencode-go/qwen3.6-plus`.",
		"Ignore command: `&>/dev/null`.",
		"Ignore placeholder: `ai/<agent>/`.",
		"Ignore secret placeholder: `sensitive/KEYNAME.secret.age`.",
		"Ignore home: `~/.claude/skills/`.",
		"Ignore env: `$SOME_VAR/00_meta/patterns/x.md`.",
		"Ignore URL: `https://example.com/a/b.md`.",
		"Ignore bare file: `review.md`.",
	}, "\n"))
	policy := Policy{
		Version:           1,
		GovernedBasenames: []string{"AGENTS.md", "README.md"},
		KnownExtensions:   []string{".sh", ".ps1", ".zsh", ".bats", ".conf", ".md", ".json", ".jsonc", ".yaml", ".yml", ".toml"},
		SourceExtensions:  []string{".conf", ".sh", ".zsh"},
	}

	report, err := CheckFiles(CheckOptions{RepoRoot: repo, Policy: policy}, []string{"AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	got := findingKeys(report.Findings)
	for _, want := range []string{
		"empty-glob:ai/skills/*/SKILL.md",
		"escape:scripts/../../outside/README.md",
		"missing:harness/skills/missing/SKILL.md",
		"missing:scripts/definitely-not-here.sh",
	} {
		if !got[want] {
			t.Errorf("missing finding %q: %+v", want, report.Findings)
		}
	}

	if len(report.Findings) != 4 {
		t.Errorf("findings = %+v, want exactly four", report.Findings)
	}
}

func TestCheckFilesTreatsNotDirectoryGlobAsEmpty(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "scripts", "not a directory\n")
	writeRepoFile(t, repo, "AGENTS.md", "Broken glob: `scripts/*.sh`.\n")
	policy := Policy{
		Version:           1,
		GovernedBasenames: []string{"AGENTS.md"},
		KnownExtensions:   []string{".sh"},
	}

	report, err := CheckFiles(CheckOptions{RepoRoot: repo, Policy: policy}, []string{"AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	if got := findingKeys(report.Findings); len(got) != 1 || !got["empty-glob:scripts/*.sh"] {
		t.Fatalf("findings = %+v", report.Findings)
	}
}

func TestCheckFilesTreatsMalformedGlobAsEmpty(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "scripts/exists.sh", "ok\n")
	writeRepoFile(t, repo, "AGENTS.md", "Broken glob: `scripts/[.sh`.\n")
	policy := Policy{
		Version:           1,
		GovernedBasenames: []string{"AGENTS.md"},
		KnownExtensions:   []string{".sh"},
	}

	report, err := CheckFiles(CheckOptions{RepoRoot: repo, Policy: policy}, []string{"AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	if got := findingKeys(report.Findings); len(got) != 1 || !got["empty-glob:scripts/[.sh"] {
		t.Fatalf("findings = %+v", report.Findings)
	}
}

func TestCheckFilesGlobFollowsSymlinkedDirectories(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "fixtures/skill/SKILL.md", "ok\n")
	if err := os.MkdirAll(filepath.Join(repo, "ai", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(repo, "fixtures", "skill"), filepath.Join(repo, "ai", "skills", "linked")); err != nil {
		t.Skipf("directory symlinks unavailable: %v", err)
	}
	writeRepoFile(t, repo, "AGENTS.md", "Existing glob: `ai/skills/*/SKILL.md`.\n")
	policy := Policy{
		Version:           1,
		GovernedBasenames: []string{"AGENTS.md"},
		KnownExtensions:   []string{".md"},
	}

	report, err := CheckFiles(CheckOptions{RepoRoot: repo, Policy: policy}, []string{"AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings = %+v", report.Findings)
	}
}

func TestCheckFilesVaultPaths(t *testing.T) {
	repo, vault := t.TempDir(), t.TempDir()
	writeRepoFile(t, repo, "AGENTS.md", strings.Join([]string{
		"Existing: `$VAULT_PATH/00_meta/real.md`.",
		"Missing: `${VAULT_PATH}/00_meta/missing.md`.",
	}, "\n"))
	writeRepoFile(t, vault, "00_meta/real.md", "ok\n")
	policy := Policy{Version: 1, GovernedBasenames: []string{"AGENTS.md"}}

	report, err := CheckFiles(CheckOptions{RepoRoot: repo, VaultRoot: vault, Policy: policy}, []string{"AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	if got := findingKeys(report.Findings); len(got) != 1 || !got["missing-vault:${VAULT_PATH}/00_meta/missing.md"] {
		t.Fatalf("vault findings = %+v", report.Findings)
	}

	report, err = CheckFiles(CheckOptions{RepoRoot: repo, Policy: policy}, []string{"AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("an unavailable optional vault must skip vault checks: %+v", report.Findings)
	}
}

func TestCheckFilesRejectsBareSourceCommand(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "AGENTS.md", strings.Join([]string{
		"Broken:",
		"```bash",
		". versions.conf",
		"```",
		"Working: `. ./versions.conf`.",
		"# Warning example: `. versions.conf` is the broken form.",
	}, "\n"))
	policy := Policy{
		Version:           1,
		GovernedBasenames: []string{"AGENTS.md"},
		SourceExtensions:  []string{".conf", ".sh", ".zsh"},
	}

	report, err := CheckFiles(CheckOptions{RepoRoot: repo, Policy: policy}, []string{"AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	got := findingKeys(report.Findings)
	if len(got) != 1 || !got["bare-source:versions.conf"] {
		t.Fatalf("bare source findings = %+v", report.Findings)
	}
}

func TestCheckFilesDeterministicOutput(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "scripts/a.sh", "ok\n")
	writeRepoFile(t, repo, "AGENTS.md", "Second `scripts/z.sh`, first `scripts/b.sh`, duplicate `scripts/z.sh`.\n")
	policy := Policy{Version: 1, GovernedBasenames: []string{"AGENTS.md"}, KnownExtensions: []string{".sh"}}

	first, err := CheckFiles(CheckOptions{RepoRoot: repo, Policy: policy}, []string{"AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := CheckFiles(CheckOptions{RepoRoot: repo, Policy: policy}, []string{"AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("reports differ:\nfirst=%+v\nsecond=%+v", first, second)
	}
	got := findingKeys(first.Findings)
	if len(got) != 2 || !got["missing:scripts/b.sh"] || !got["missing:scripts/z.sh"] {
		t.Fatalf("findings = %+v", first.Findings)
	}
}

func TestCheckFilesMissingExplicitFileIsAnError(t *testing.T) {
	repo := t.TempDir()
	if _, err := CheckFiles(CheckOptions{RepoRoot: repo, Policy: Policy{Version: 1}}, []string{"AGENTS.md"}); err == nil {
		t.Fatal("CheckFiles() succeeded for a missing explicit file")
	}
}

func findingKeys(findings []Finding) map[string]bool {
	out := make(map[string]bool, len(findings))
	for _, finding := range findings {
		out[finding.Kind+":"+finding.Token] = true
	}
	return out
}

func writeRepoFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	assertInside(t, root, path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertInside(t *testing.T, root, path string) {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") || filepath.IsAbs(rel) {
		t.Fatalf("%s is outside %s", path, root)
	}
}
