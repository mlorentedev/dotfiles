package docpaths

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCurrentRepository(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	policy, err := LoadPolicy(filepath.Join(repo, "harness", "doc-paths.json"))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	files, err := Discover(repo, policy)
	if err != nil {
		t.Fatal(err)
	}
	report, err := CheckFiles(CheckOptions{
		RepoRoot: repo, VaultRoot: os.Getenv("VAULT_PATH"), Policy: policy,
	}, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) > 0 {
		t.Fatalf("current repository findings: %+v", report.Findings)
	}
	if elapsed := time.Since(start); runtime.GOOS == "windows" && elapsed > 2*time.Second {
		t.Fatalf("current repository scan took %s, want <=2s", elapsed)
	}
}

func TestCurrentRepositoryReadOnly(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	before := gitStatus(t, repo)
	policy, err := LoadPolicy(filepath.Join(repo, "harness", "doc-paths.json"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := Discover(repo, policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CheckFiles(CheckOptions{RepoRoot: repo, Policy: policy}, files); err != nil {
		t.Fatal(err)
	}
	if after := gitStatus(t, repo); !reflect.DeepEqual(after, before) {
		t.Fatalf("scan mutated the checkout:\nbefore=%q\nafter=%q", before, after)
	}
}

func gitStatus(t *testing.T, repo string) []string {
	t.Helper()
	cmd := exec.Command("git", "-C", repo, "status", "--porcelain=v1")
	cmd.Env = cleanGitConfigEnv(os.Environ())
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	status := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(status) == 1 && status[0] == "" {
		return nil
	}
	return status
}
