package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckHomeDeployDrift drives the deploy-dir↔$HOME leg — the one no doctor
// section covered before OPS-043, and the one setup's check_deployed was the
// only assertion on. Each row is a branch of the contract: content drift on a
// checked entry FAILs, the same drift on an exempt entry does not, a symlink
// FAILs even when it resolves to identical bytes, and an unprovisioned source
// SKIPs rather than failing (which is how setup's `[ -f "$DOTFILES_DIR/..." ]`
// conditionals behave).
func TestCheckHomeDeployDrift(t *testing.T) {
	type file struct{ path, content string }
	cases := []struct {
		name         string
		deployFiles  []file
		homeFiles    []file
		symlinks     map[string]string // $HOME rel path → deploy-dir rel path it points at
		wantFailures int
		wantSubstr   string
	}{
		{
			name:        "checked entries agree → pass",
			deployFiles: []file{{".zsh/functions.sh", "f()"}, {".zsh/aliases.zsh", "a=1"}},
			homeFiles:   []file{{".zsh/functions.sh", "f()"}, {".zsh/aliases.zsh", "a=1"}},
			wantSubstr:  "matches",
		},
		{
			// The file that lives in NO doctor list today: checkSymlinks covers
			// aliases.zsh and functions.zsh only.
			name:         "functions.sh drift → fail naming both paths",
			deployFiles:  []file{{".zsh/functions.sh", "new"}},
			homeFiles:    []file{{".zsh/functions.sh", "old"}},
			wantFailures: 1,
			wantSubstr:   ".zsh/functions.sh",
		},
		{
			name:         "tmux.conf drift → fail, and the two legs use different relative paths",
			deployFiles:  []file{{"tmux.conf", "set -g mouse on"}},
			homeFiles:    []file{{".tmux.conf", "set -g mouse off"}},
			wantFailures: 1,
			wantSubstr:   ".tmux.conf",
		},
		{
			// Measured on msi 2026-09-02: the only file of the eleven observed
			// drifting. Every `git config --global` rewrites it.
			name:        "gitconfig drift → exempt, pass",
			deployFiles: []file{{".gitconfig", "[user]\n\tname = repo"}},
			homeFiles:   []file{{".gitconfig", "[user]\n\tname = local"}},
			wantSubstr:  "exists",
		},
		{
			// Installers (opencode, bun, NVM, ggshield) append PATH/init lines.
			name:        "rc files drift → exempt, pass",
			deployFiles: []file{{".zshrc", "base"}, {".bashrc", "base"}, {".profile", "base"}},
			homeFiles:   []file{{".zshrc", "base\nexport PATH=x"}, {".bashrc", "base\n. bun"}, {".profile", "base\nnvm"}},
			wantSubstr:  "exists",
		},
		{
			// ADR-012 moved deployment to copy, so a symlink here is a
			// pre-ADR-012 leftover. cmp follows it and checkSymlinks PASSes it,
			// so this FAIL exists nowhere else in doctor.
			name:         "symlink at $HOME → fail even when content resolves equal",
			deployFiles:  []file{{".zsh/functions.sh", "f()"}},
			symlinks:     map[string]string{".zsh/functions.sh": ".zsh/functions.sh"},
			wantFailures: 1,
			wantSubstr:   "symlink",
		},
		{
			name:         "deployed source present but $HOME copy missing → fail",
			deployFiles:  []file{{".inputrc", "set completion-ignore-case on"}},
			wantFailures: 1,
			wantSubstr:   "missing",
		},
		{
			// setup guards .profile/.gitconfig/.bashrc on the SOURCE existing,
			// so "not provisioned" is a skip, never a failure (R4).
			name:        "source absent from deploy dir → skip, not fail",
			deployFiles: []file{{".zsh/functions.sh", "f()"}},
			homeFiles:   []file{{".zsh/functions.sh", "f()"}},
			wantSubstr:  "not provisioned",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			deploy := filepath.Join(t.TempDir(), "deploy")
			mkdirAll(t, deploy)
			for _, f := range tc.deployFiles {
				writeFile(t, filepath.Join(deploy, filepath.FromSlash(f.path)), f.content)
			}
			for _, f := range tc.homeFiles {
				writeFile(t, filepath.Join(home, filepath.FromSlash(f.path)), f.content)
			}
			for dst, src := range tc.symlinks {
				p := filepath.Join(home, filepath.FromSlash(dst))
				mkdirAll(t, filepath.Dir(p))
				if err := os.Symlink(filepath.Join(deploy, filepath.FromSlash(src)), p); err != nil {
					t.Fatalf("symlink: %v", err)
				}
			}

			sys := newSys(map[string]string{"HOME": home}, nil, nil)
			cfg := &Config{DotfilesDir: deploy}

			var buf bytes.Buffer
			rep := capture(&buf)
			checkHomeDeployDrift(sys, cfg, rep)

			if rep.Failures() != tc.wantFailures {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFailures, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}

// TestHomeDeployMapCoversSetup is the join guard (AC5). Nothing in the repo
// spans setup-linux.sh and this map, which is the gap #1337 itself came through:
// a claim about coverage that no test could refute. It reads the script from the
// repo root the way env_test.go and prtriage_test.go already do.
func TestHomeDeployMapCoversSetup(t *testing.T) {
	script := filepath.Join("..", "..", "..", "setup-linux.sh")
	b, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("read %s: %v", script, err)
	}

	pairs := setupHomeDeployPairs(string(b))
	if len(pairs) == 0 {
		t.Fatal("no `deploy_file \"$DOTFILES_DIR/...\" \"$HOME/...\"` call sites found — the parser or the script changed shape")
	}

	covered := map[string]string{}
	for _, e := range homeDeployMap {
		covered[e.src] = e.dst
	}
	for src, dst := range pairs {
		got, ok := covered[src]
		if !ok {
			t.Errorf("setup-linux.sh deploys %q to $HOME/%s but homeDeployMap has no entry — add one, or doctor stops covering it", src, dst)
			continue
		}
		if got != dst {
			t.Errorf("homeDeployMap[%q] targets $HOME/%s, setup deploys it to $HOME/%s", src, got, dst)
		}
	}
	for _, e := range homeDeployMap {
		if _, ok := pairs[e.src]; !ok {
			t.Errorf("homeDeployMap covers %q but setup-linux.sh no longer deploys it — a stale entry outlives its mechanism (lesson 256)", e.src)
		}
	}
}

// TestHomeDeployExemptionsAreReasoned pins that an exemption cannot be silent:
// every non-content-checked entry carries why. An exemption with no stated
// mechanism is the defect R5 found in setup's own NOTE, where a `sed -i` that
// no longer exists still justified skipping .zshrc.
func TestHomeDeployExemptionsAreReasoned(t *testing.T) {
	for _, e := range homeDeployMap {
		if !e.contentChecked && strings.TrimSpace(e.exemptReason) == "" {
			t.Errorf("homeDeployMap entry %q is exempt from the content check with no reason", e.src)
		}
		if e.contentChecked && e.exemptReason != "" {
			t.Errorf("homeDeployMap entry %q is content-checked but carries an exemption reason", e.src)
		}
	}
}
