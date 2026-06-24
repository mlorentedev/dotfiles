package mem

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestClaudeAdapterGolden is PR2b-2b's gate: the Go adapter (ClaudeContext +
// ClaudeEnvelope) must be byte-for-byte identical to the live claude-session-start.sh
// for the same inputs. This is what lets the shell hook cluster be deleted at PR3.
//
// POSIX-only by design (same reason as the session-brief gate: absolute paths render
// differently under Windows, and the hook is bash). Hermetic: the hook + core are
// copied into a scriptsDir with NO vault-health.sh (health pins to "not found"), NO
// deployed env-contract (doctor-drift gated off), NO ensure-memory-symlink helper and
// no vault memory source (auto-memory is a no-op) — so every run is deterministic.
func TestClaudeAdapterGolden(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("claude-session-start.sh is POSIX; Windows parity is a separate target")
	}
	for _, bin := range []string{"bash", "jq"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s required to run the shell hook", bin)
		}
	}

	repo := repoRoot(t)
	scriptsDir := t.TempDir()
	copyFile(t, filepath.Join(repo, "scripts", "claude-session-start.sh"), filepath.Join(scriptsDir, "claude-session-start.sh"))
	copyFile(t, filepath.Join(repo, "scripts", "session-brief.sh"), filepath.Join(scriptsDir, "session-brief.sh"))
	hook := filepath.Join(scriptsDir, "claude-session-start.sh")

	now := time.Now()
	vault := t.TempDir()

	scenarios := map[string]string{
		"bare-outside-vault":  t.TempDir(),
		"git-repo-hive-specs": newGitHiveCWD(t, vault, now.Add(-100*24*time.Hour)),
		"inside-vault":        newVaultCWD(t, now.Add(-100*24*time.Hour)),
	}

	for name, cwd := range scenarios {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			dotfilesDir := t.TempDir() // no env-contract.json -> doctor-drift gated off

			cmd := exec.Command("bash", hook)
			cmd.Stdin = strings.NewReader(fmt.Sprintf(`{"cwd":%q,"session_id":"golden"}`, cwd))
			cmd.Env = append(os.Environ(),
				"HOME="+home,
				"VAULT_PATH="+vault,
				"DOTFILES_DIR="+dotfilesDir,
			)
			shOut, err := cmd.Output()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					t.Fatalf("hook failed: %v\nstderr:\n%s", err, ee.Stderr)
				}
				t.Fatalf("hook failed: %v", err)
			}

			ctx := ClaudeContext(ClaudeContextInput{
				Cwd: cwd, Vault: vault, ScriptsDir: scriptsDir, Home: home,
				ContractPath: filepath.Join(dotfilesDir, "env-contract.json"),
				ClaudeJSON:   filepath.Join(home, ".claude", ".claude.json"),
				ConfigPath:   filepath.Join(scriptsDir, "..", "session-start-config.json"),
				Now:          now,
				DoctorQuick:  nil,
			})
			goOut, err := ClaudeEnvelope(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if string(shOut) != goOut {
				t.Errorf("golden mismatch:\n--- shell ---\n%s\n--- go ---\n%s", shOut, goOut)
			}
		})
	}
}

// newGitHiveCWD builds a git repo CWD that triggers the .git block: a hive vault
// project match (vault/10_projects/<repo>), 1 active + 1 archived spec, and a stale
// docs/lessons.md.
func newGitHiveCWD(t *testing.T, vault string, lessonsTime time.Time) string {
	cwd := filepath.Join(t.TempDir(), "myrepo")
	mustMkdirAll(t, filepath.Join(cwd, ".git"))
	mustMkdirAll(t, filepath.Join(cwd, "specs", "SPEC-1"))
	mustMkdirAll(t, filepath.Join(cwd, "specs", "archive", "OLD-1"))
	mustMkdirAll(t, filepath.Join(vault, "10_projects", "myrepo")) // hive match
	lessons := filepath.Join(cwd, "docs", "lessons.md")
	mustWrite(t, lessons, "lessons\n")
	if err := os.Chtimes(lessons, lessonsTime, lessonsTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	return cwd
}
