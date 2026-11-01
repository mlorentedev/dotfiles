package doctor

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeRecord writes a committed record whose provenance names src (relative to
// the vault) at the given sha.
func writeRecord(t *testing.T, repo, rel, src, sha string) {
	t.Helper()
	p := filepath.Join(repo, filepath.FromSlash(rel))
	mkdirAll(t, filepath.Dir(p))
	body := "---\ngenerated: true\ngenerated_from: " + src + "\ngenerated_sha: " + sha + "\nid: x\n---\n\nbody\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// recordSource is a vault source whose stamped render is exactly the body
// writeRecord writes, so a record written from its sha is also its render.
const recordSource = "---\nid: x\n---\n\nbody\n"

// writeSource writes a vault source and returns the 16-hex sha compile-harness.sh records for it.
func writeSource(t *testing.T, vault, rel, content string) string {
	t.Helper()
	p := filepath.Join(vault, filepath.FromSlash(rel))
	mkdirAll(t, filepath.Dir(p))
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])[:16]
}

func TestCheckSkillRecordsAgainstVault(t *testing.T) {
	const skillSrc = "00_meta/skills/teardown/SKILL.md"
	const agentSrc = "00_meta/agents/definitions/builder/AGENT.md"

	run := func(t *testing.T, env map[string]string) (string, *Report) {
		t.Helper()
		var buf bytes.Buffer
		rep := capture(&buf)
		checkSkillRecordsAgainstVault(newSys(env, nil, nil), rep)
		return buf.String(), rep
	}

	t.Run("records refreshed from the current vault pass", func(t *testing.T) {
		repo, vault := t.TempDir(), t.TempDir()
		writeRecord(t, repo, "harness/skills/teardown/SKILL.md", skillSrc, writeSource(t, vault, skillSrc, recordSource))
		writeRecord(t, repo, "harness/agents/builder/AGENT.md", agentSrc, writeSource(t, vault, agentSrc, recordSource))
		out, rep := run(t, map[string]string{"DOTFILES_REPO_DIR": repo, "VAULT_PATH": vault})
		if rep.Warnings() != 0 || rep.Failures() != 0 {
			t.Fatalf("want a clean pass\n%s", out)
		}
		if !strings.Contains(out, "2 committed records match their vault source") {
			t.Errorf("want the pass to count the records\n%s", out)
		}
	})

	t.Run("a vault edit the records were not refreshed from warns and names the record", func(t *testing.T) {
		repo, vault := t.TempDir(), t.TempDir()
		old := writeSource(t, vault, skillSrc, "v1\n")
		writeRecord(t, repo, "harness/skills/teardown/SKILL.md", skillSrc, old)
		writeSource(t, vault, skillSrc, "v2\n")
		out, rep := run(t, map[string]string{"DOTFILES_REPO_DIR": repo, "VAULT_PATH": vault})
		if rep.Warnings() != 1 {
			t.Fatalf("want 1 warning, got %d\n%s", rep.Warnings(), out)
		}
		for _, want := range []string{"skills/teardown", "dotf harness refresh"} {
			if !strings.Contains(out, want) {
				t.Errorf("want %q in the warning\n%s", want, out)
			}
		}
	})

	t.Run("a record edited without its vault source warns and names the record", func(t *testing.T) {
		repo, vault := t.TempDir(), t.TempDir()
		writeRecord(t, repo, "harness/skills/teardown/SKILL.md", skillSrc, writeSource(t, vault, skillSrc, recordSource+"an edit made only in the vault\n"))
		out, rep := run(t, map[string]string{"DOTFILES_REPO_DIR": repo, "VAULT_PATH": vault})
		if rep.Warnings() != 1 || strings.Contains(out, "[ OK ]") {
			t.Fatalf("want one warning and no pass\n%s", out)
		}
		for _, want := range []string{"skills/teardown", "carry it to the vault source"} {
			if !strings.Contains(out, want) {
				t.Errorf("want %q in the warning\n%s", want, out)
			}
		}
	})

	t.Run("a record whose vault source is gone warns as such", func(t *testing.T) {
		repo, vault := t.TempDir(), t.TempDir()
		writeRecord(t, repo, "harness/agents/builder/AGENT.md", agentSrc, "0123456789abcdef")
		out, rep := run(t, map[string]string{"DOTFILES_REPO_DIR": repo, "VAULT_PATH": vault})
		if rep.Warnings() != 1 || !strings.Contains(out, "agents/builder (source gone)") {
			t.Fatalf("want one warning naming the missing source\n%s", out)
		}
	})

	t.Run("a record without provenance warns instead of being skipped", func(t *testing.T) {
		repo, vault := t.TempDir(), t.TempDir()
		writeRecord(t, repo, "harness/skills/teardown/SKILL.md", skillSrc, writeSource(t, vault, skillSrc, recordSource))
		p := filepath.Join(repo, "harness", "skills", "bare", "SKILL.md")
		mkdirAll(t, filepath.Dir(p))
		if err := os.WriteFile(p, []byte("---\nid: bare\n---\n\nbody\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, rep := run(t, map[string]string{"DOTFILES_REPO_DIR": repo, "VAULT_PATH": vault})
		if rep.Warnings() != 1 || !strings.Contains(out, "skills/bare") || strings.Contains(out, "[ OK ]") {
			t.Fatalf("want one warning naming the unstamped record and no pass\n%s", out)
		}
	})

	t.Run("no vault on this machine skips", func(t *testing.T) {
		repo := t.TempDir()
		writeRecord(t, repo, "harness/skills/teardown/SKILL.md", skillSrc, "0123456789abcdef")
		out, rep := run(t, map[string]string{"DOTFILES_REPO_DIR": repo, "VAULT_PATH": filepath.Join(repo, "absent")})
		if rep.Warnings() != 0 || !strings.Contains(out, "SKIP") {
			t.Fatalf("want a skip without a vault\n%s", out)
		}
	})
}

// TestCheckSkillRecordsNamesAnUnreadableRecordsDir: filepath.Glob read an
// unreadable harness/skills/ as holding no records, so the check passed over
// it in silence (#2797).
func TestCheckSkillRecordsNamesAnUnreadableRecordsDir(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("directory permissions do not deny reads here")
	}
	repo, vault := t.TempDir(), t.TempDir()
	const src = "00_meta/agents/definitions/builder/AGENT.md"
	writeRecord(t, repo, "harness/agents/builder/AGENT.md", src, writeSource(t, vault, src, recordSource))
	writeRecord(t, repo, "harness/skills/teardown/SKILL.md", "00_meta/skills/teardown/SKILL.md", "0123456789abcdef")
	locked := filepath.Join(repo, "harness", "skills")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) }) //nolint:gosec // restores a test directory so TempDir can remove it

	var buf bytes.Buffer
	rep := capture(&buf)
	checkSkillRecordsAgainstVault(newSys(map[string]string{"DOTFILES_REPO_DIR": repo, "VAULT_PATH": vault}, nil, nil), rep)
	out := buf.String()
	if rep.Warnings() != 1 || !strings.Contains(out, "cannot read harness/skills") {
		t.Errorf("want one warning naming the unreadable records dir, got %d\n%s", rep.Warnings(), out)
	}
	if strings.Contains(out, "match their vault source") {
		t.Errorf("a partial read must not report a clean pass\n%s", out)
	}
}
