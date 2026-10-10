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
			name:       "managed file absent in deploy → not drift",
			repoFiles:  []file{{"scripts/x.sh", "echo hi"}},
			lsFiles:    []string{"scripts/x.sh"},
			wantSubstr: "agree",
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
			checkDeployDrift(sys, cfg, rep, false)

			if rep.Failures() != tc.wantFailures {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFailures, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}

// TestCheckDeployDirLeftovers drives the half of the drift check that looks at
// files only the deploy dir has (#2266). git is faked through the same seam:
// the shallow probe and the deleted-paths log, keyed on the temp repo.
func TestCheckDeployDirLeftovers(t *testing.T) {
	logKey := func(repo string) string {
		return "git -C " + repo + " log --all --no-renames --diff-filter=D --name-only --pretty=format: -- .zsh ssh scripts"
	}
	setup := func(t *testing.T, gitFile bool) (repo, deploy string, cmdOut map[string]string) {
		repo, deploy = t.TempDir(), filepath.Join(t.TempDir(), "deploy")
		if gitFile {
			// A linked worktree: .git is a file naming the gitdir.
			writeFile(t, filepath.Join(repo, ".git"), "gitdir: /elsewhere\n")
		} else {
			mkdirAll(t, filepath.Join(repo, ".git"))
		}
		writeFile(t, filepath.Join(repo, "scripts", "live.sh"), "live")
		writeFile(t, filepath.Join(deploy, "scripts", "live.sh"), "live")
		writeFile(t, filepath.Join(deploy, "scripts", "old.sh"), "retired")
		cmdOut = map[string]string{
			"git -C " + repo + " ls-files":                          "scripts/live.sh",
			"git -C " + repo + " rev-parse --is-shallow-repository": "false",
			logKey(repo): "scripts/old.sh",
		}
		return repo, deploy, cmdOut
	}
	run := func(repo, deploy string, cmdOut map[string]string, fix bool) (*Report, string) {
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, cmdOut)
		var buf bytes.Buffer
		rep := capture(&buf)
		checkDeployDrift(sys, &Config{DotfilesDir: deploy}, rep, fix)
		return rep, buf.String()
	}

	t.Run("a leftover fails and names the remedy", func(t *testing.T) {
		repo, deploy, cmdOut := setup(t, false)
		rep, out := run(repo, deploy, cmdOut, false)
		if rep.Failures() != 1 || !strings.Contains(out, "leftover: scripts/old.sh") || !strings.Contains(out, "dotf doctor --fix") {
			t.Fatalf("want one failure naming scripts/old.sh and the remedy:\n%s", out)
		}
		if !pathExists(filepath.Join(deploy, "scripts", "old.sh")) {
			t.Error("a report without --fix removed the file")
		}
	})

	t.Run("--fix prunes it and a re-run passes", func(t *testing.T) {
		repo, deploy, cmdOut := setup(t, false)
		rep, out := run(repo, deploy, cmdOut, true)
		if rep.Failures() != 0 || !strings.Contains(out, "pruned deploy-dir leftover: scripts/old.sh") {
			t.Fatalf("want the prune reported and no failure:\n%s", out)
		}
		if pathExists(filepath.Join(deploy, "scripts", "old.sh")) {
			t.Error("--fix left the leftover")
		}
		_, out = run(repo, deploy, cmdOut, false)
		if !strings.Contains(out, "no leftovers") {
			t.Errorf("re-run should pass:\n%s", out)
		}
	})

	t.Run("an orphan git never tracked is only named", func(t *testing.T) {
		repo, deploy, cmdOut := setup(t, false)
		cmdOut[logKey(repo)] = ""
		rep, out := run(repo, deploy, cmdOut, true)
		if rep.Failures() != 0 || !strings.Contains(out, "scripts/old.sh is in the deploy dir but not in the checkout") {
			t.Fatalf("want a warning, no failure:\n%s", out)
		}
		if !pathExists(filepath.Join(deploy, "scripts", "old.sh")) {
			t.Error("--fix removed a file git never tracked")
		}
	})

	t.Run("a worktree checkout is checked, not skipped", func(t *testing.T) {
		repo, deploy, cmdOut := setup(t, true)
		_, out := run(repo, deploy, cmdOut, false)
		if strings.Contains(out, "not a git repo") || !strings.Contains(out, "leftover: scripts/old.sh") {
			t.Fatalf("a .git file is a worktree, not a missing repo:\n%s", out)
		}
	})
}
