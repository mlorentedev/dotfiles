package worktree

import (
	"strings"
	"testing"
	"time"
)

func TestParsePorcelain(t *testing.T) {
	porcelainOutput := `worktree /home/manu/Projects/dotfiles
HEAD eaf8a91c1d0b5e2060db028562d29486c67ff752
branch refs/heads/main

worktree /home/manu/Projects/dotfiles-wt-agents
HEAD 7b00e84b803a60a7ad9bbff8e68407c91fa049b6
branch refs/heads/feat/cli-075-dotf-worktree-lifecycle

worktree /home/manu/Projects/dotfiles-wt-detached
HEAD 11223344556677889900aabbccddeeff00112233
detached

worktree /home/manu/Projects/dotfiles/vendor/submodule
HEAD aabbccddeeff0011223344556677889900aabbcc
branch refs/heads/main
gitdir /home/manu/Projects/dotfiles/.git/modules/submodule
`

	entries, err := ParsePorcelain(porcelainOutput)
	if err != nil {
		t.Fatalf("unexpected error parsing porcelain output: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	// Entry 0: Main
	if entries[0].Path != "/home/manu/Projects/dotfiles" {
		t.Errorf("expected path /home/manu/Projects/dotfiles, got %s", entries[0].Path)
	}
	if entries[0].Branch != "main" {
		t.Errorf("expected branch main, got %s", entries[0].Branch)
	}
	if entries[0].Detached {
		t.Errorf("expected main not to be detached")
	}

	// Entry 1: Feature branch
	if entries[1].Path != "/home/manu/Projects/dotfiles-wt-agents" {
		t.Errorf("expected path /home/manu/Projects/dotfiles-wt-agents, got %s", entries[1].Path)
	}
	if entries[1].Branch != "feat/cli-075-dotf-worktree-lifecycle" {
		t.Errorf("expected branch feat/cli-075-dotf-worktree-lifecycle, got %s", entries[1].Branch)
	}

	// Entry 2: Detached
	if !entries[2].Detached {
		t.Errorf("expected entry 2 to be detached")
	}

	// Entry 3: Submodule
	if !entries[3].IsSubmodule {
		t.Errorf("expected entry 3 to be detected as submodule")
	}
}

func TestFilterSubmodules(t *testing.T) {
	cases := []struct {
		gitdir   string
		expected bool
	}{
		{"/home/user/repo/.git/worktrees/wt-1", false},
		{"/home/user/repo/.git/modules/my-submodule", true},
		{"/home/user/repo/.git/modules/nested/sub", true},
		{"", false},
	}

	for _, tc := range cases {
		got := IsSubmoduleGitDir(tc.gitdir)
		if got != tc.expected {
			t.Errorf("IsSubmoduleGitDir(%q) = %v, expected %v", tc.gitdir, got, tc.expected)
		}
	}
}

func TestClassify(t *testing.T) {
	now := time.Now()

	t.Run("main repository is always active", func(t *testing.T) {
		info := Info{
			Path:   "/home/user/repo",
			Branch: "main",
			IsMain: true,
		}
		state, _ := Classify(info, now)
		if state != StateActive {
			t.Errorf("expected main repo to be StateActive, got %s", state)
		}
	})

	t.Run("active lease keeps worktree active", func(t *testing.T) {
		info := Info{
			Path:   "/home/user/repo-wt-feature",
			Branch: "feat/x",
			Metadata: &Metadata{
				ReapOK:         true,
				CreatedAt:      now.Add(-2 * time.Hour),
				LeaseExpiresAt: now.Add(1 * time.Hour), // active
			},
		}
		state, _ := Classify(info, now)
		if state != StateActive {
			t.Errorf("expected active lease to yield StateActive, got %s", state)
		}
	})

	t.Run("dirty worktree is StateDirty even if lease expired and PR merged", func(t *testing.T) {
		info := Info{
			Path:     "/home/user/repo-wt-feature",
			Branch:   "feat/x",
			Dirty:    true,
			PRMerged: true,
			Metadata: &Metadata{
				ReapOK:         true,
				CreatedAt:      now.Add(-2 * time.Hour),
				LeaseExpiresAt: now.Add(-1 * time.Hour), // expired
			},
		}
		state, _ := Classify(info, now)
		if state != StateDirty {
			t.Errorf("expected dirty worktree to yield StateDirty, got %s", state)
		}
	})

	t.Run("freshly created clean branch is StateActive due to 15m minimum age guard", func(t *testing.T) {
		info := Info{
			Path:     "/home/user/repo-wt-feature",
			Branch:   "feat/fresh",
			Dirty:    false,
			PRMerged: true, // hypothetical
			Metadata: &Metadata{
				ReapOK:         true,
				CreatedAt:      now.Add(-5 * time.Minute), // only 5m old (< 15m)
				LeaseExpiresAt: now.Add(-1 * time.Minute),
			},
		}
		state, reason := Classify(info, now)
		if state != StateActive {
			t.Errorf("expected <15m worktree to be StateActive, got %s", state)
		}
		if !strings.Contains(reason, "age < 15m") {
			t.Errorf("expected reason to mention age guard, got %s", reason)
		}
	})

	t.Run("all fail-closed gates met yields StateReapable", func(t *testing.T) {
		info := Info{
			Path:     "/home/user/repo-wt-feature",
			Branch:   "feat/done",
			Dirty:    false,
			PRMerged: true,
			Metadata: &Metadata{
				ReapOK:         true,
				CreatedAt:      now.Add(-24 * time.Hour),
				LeaseExpiresAt: now.Add(-1 * time.Hour), // expired
			},
		}
		state, _ := Classify(info, now)
		if state != StateReapable {
			t.Errorf("expected StateReapable, got %s", state)
		}
	})

	t.Run("unmerged PR yields StateUnmerged", func(t *testing.T) {
		info := Info{
			Path:     "/home/user/repo-wt-feature",
			Branch:   "feat/wip",
			Dirty:    false,
			PRMerged: false,
			Metadata: &Metadata{
				ReapOK:         true,
				CreatedAt:      now.Add(-24 * time.Hour),
				LeaseExpiresAt: now.Add(-1 * time.Hour),
			},
		}
		state, _ := Classify(info, now)
		if state != StateUnmerged {
			t.Errorf("expected StateUnmerged, got %s", state)
		}
	})

	t.Run("nil metadata refuses reap (Gate a fail-closed)", func(t *testing.T) {
		info := Info{
			Path:     "/home/user/repo-wt-manual",
			Branch:   "feat/manual",
			Dirty:    false,
			PRMerged: true,
			Metadata: nil, // manually created worktree without .dotf-worktree.json
		}
		state, reason := Classify(info, now)
		if state != StateActive {
			t.Errorf("expected metadata-less worktree to be StateActive, got %s", state)
		}
		if !strings.Contains(reason, "no dotf metadata") {
			t.Errorf("expected reason to mention missing metadata, got %q", reason)
		}
	})

	t.Run("reap_ok false keeps worktree active", func(t *testing.T) {
		info := Info{
			Path:     "/home/user/repo-wt-hold",
			Branch:   "feat/hold",
			Dirty:    false,
			PRMerged: true,
			Metadata: &Metadata{
				ReapOK:         false, // explicit hold
				CreatedAt:      now.Add(-24 * time.Hour),
				LeaseExpiresAt: now.Add(-1 * time.Hour),
			},
		}
		state, reason := Classify(info, now)
		if state != StateActive {
			t.Errorf("expected reap_ok=false to be StateActive, got %s", state)
		}
		if !strings.Contains(reason, "reap hold") {
			t.Errorf("expected reason to mention reap hold, got %q", reason)
		}
	})
}
