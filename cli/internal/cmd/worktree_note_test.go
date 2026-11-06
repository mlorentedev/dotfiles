package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestKnownWorktreesNote(t *testing.T) {
	const takes = "; done takes a path, or the slug add was given"
	cases := []struct {
		name      string
		porcelain string
		err       error
		want      string
	}{
		{"a listing failure says only what done takes", "", errors.New("git worktree list: exit 128"), takes},
		// A line over bufio.Scanner's 64 KiB token limit is the one input
		// ParsePorcelain refuses, so it stands in for any unparseable listing.
		{"a listing that does not parse says only what done takes",
			"worktree /repo\n" + strings.Repeat("x", 70*1024) + "\n", nil, takes},
		{"no linked worktree says so", "worktree /repo\nHEAD abc\nbranch refs/heads/main\n", nil,
			takes + "; this repository has no linked worktrees"},
		{"each linked worktree is named by slug and branch",
			"worktree /repo\nHEAD abc\nbranch refs/heads/main\n\nworktree /wt/repo/fix-a\nHEAD def\nbranch refs/heads/fix/renamed\n", nil,
			takes + "; known worktrees, as slug (branch): fix-a (fix/renamed)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			saved := listWorktrees
			t.Cleanup(func() { listWorktrees = saved })
			listWorktrees = func(string) (string, error) { return c.porcelain, c.err }
			if got := knownWorktreesNote(t.TempDir()); got != c.want {
				t.Errorf("note = %q\nwant  %q", got, c.want)
			}
		})
	}
}
