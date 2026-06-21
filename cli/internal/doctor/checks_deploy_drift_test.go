package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckDeployDrift drives the repo↔deploy-dir byte-drift port of diff-check:
// one row per branch (agree, managed drift, unmanaged ignored, half-present,
// missing pieces). `git ls-files` is faked via the CommandOutput seam keyed on
// the temp repo path, so the test stays git-free.
func TestCheckDeployDrift(t *testing.T) {
	type file struct{ path, content string }
	cases := []struct {
		name         string
		repoFiles    []file
		deployFiles  []file
		lsFiles      []string // what `git ls-files` reports
		noGit        bool     // omit repo/.git → not-a-git-repo
		noDeploy     bool     // omit the deploy dir entirely
		gitErr       bool     // no ls-files entry in cmdOut → CommandOutput errors
		wantFailures int
		wantSubstr   string
	}{
		{
			name:        "all managed equal → pass",
			repoFiles:   []file{{"versions.conf", "A=1"}, {"scripts/x.sh", "echo hi"}},
			deployFiles: []file{{"versions.conf", "A=1"}, {"scripts/x.sh", "echo hi"}},
			lsFiles:     []string{"versions.conf", "scripts/x.sh"},
			wantSubstr:  "agree",
		},
		{
			name:         "managed drift → fail naming the path",
			repoFiles:    []file{{"versions.conf", "A=2"}},
			deployFiles:  []file{{"versions.conf", "A=1"}},
			lsFiles:      []string{"versions.conf"},
			wantFailures: 1,
			wantSubstr:   "versions.conf",
		},
		{
			name:        "unmanaged tracked file drift → ignored",
			repoFiles:   []file{{"README.md", "new"}},
			deployFiles: []file{{"README.md", "old"}},
			lsFiles:     []string{"README.md"},
			wantSubstr:  "agree",
		},
		{
			name:        "managed file absent in deploy → not drift",
			repoFiles:   []file{{"scripts/x.sh", "echo hi"}},
			lsFiles:     []string{"scripts/x.sh"},
			wantSubstr:  "agree",
		},
		{
			name:       "no deploy-dir → skip",
			repoFiles:  []file{{"versions.conf", "A=1"}},
			lsFiles:    []string{"versions.conf"},
			noDeploy:   true,
			wantSubstr: "deploy-dir",
		},
		{
			name:        "not a git repo → skip",
			repoFiles:   []file{{"versions.conf", "A=1"}},
			deployFiles: []file{{"versions.conf", "A=1"}},
			lsFiles:     []string{"versions.conf"},
			noGit:       true,
			wantSubstr:  "git repo",
		},
		{
			name:        "git ls-files fails → warn, no crash",
			repoFiles:   []file{{"versions.conf", "A=1"}},
			deployFiles: []file{{"versions.conf", "A=1"}},
			gitErr:      true,
			wantSubstr:  "git ls-files",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			deploy := filepath.Join(t.TempDir(), "deploy")
			if !tc.noDeploy {
				mkdirAll(t, deploy)
			}
			if !tc.noGit {
				mkdirAll(t, filepath.Join(repo, ".git"))
			}
			for _, f := range tc.repoFiles {
				writeFile(t, filepath.Join(repo, filepath.FromSlash(f.path)), f.content)
			}
			for _, f := range tc.deployFiles {
				writeFile(t, filepath.Join(deploy, filepath.FromSlash(f.path)), f.content)
			}

			cmdOut := map[string]string{}
			if !tc.gitErr {
				cmdOut["git -C "+repo+" ls-files"] = strings.Join(tc.lsFiles, "\n")
			}
			sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, cmdOut)
			cfg := &Config{DotfilesDir: deploy}

			var buf bytes.Buffer
			rep := capture(&buf)
			checkDeployDrift(sys, cfg, rep)

			if rep.Failures() != tc.wantFailures {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFailures, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}

// TestIsManagedDeployPath pins the allowlist that MUST mirror setup's copy block.
func TestIsManagedDeployPath(t *testing.T) {
	managed := []string{
		"versions.conf", ".zshrc", ".bashrc", ".profile", ".gitconfig", "tmux.conf",
		".zsh/aliases.zsh", "ssh/config", "scripts/utils.sh", "sensitive/env-mapping.conf",
	}
	unmanaged := []string{"README.md", "go.mod", "cli/main.go", "docs/lessons.md", ".github/workflows/ci.yml"}
	for _, p := range managed {
		if !isManagedDeployPath(p) {
			t.Errorf("isManagedDeployPath(%q) = false, want true", p)
		}
	}
	for _, p := range unmanaged {
		if isManagedDeployPath(p) {
			t.Errorf("isManagedDeployPath(%q) = true, want false", p)
		}
	}
}
