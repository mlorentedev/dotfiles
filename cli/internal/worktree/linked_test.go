package worktree

import "testing"

const linkedPorcelain = `worktree /r/repo
HEAD aaaa
branch refs/heads/main

worktree /w/repo/fix-login
HEAD bbbb
branch refs/heads/tweak

worktree /w/repo/detached
HEAD cccc
detached
`

func TestLinkedWorktreesDropTheMainCheckout(t *testing.T) {
	linked, err := LinkedWorktrees(linkedPorcelain)
	if err != nil {
		t.Fatal(err)
	}
	if len(linked) != 2 || linked[0].Path != "/w/repo/fix-login" {
		t.Fatalf("linked = %+v, want the two worktrees after the main checkout", linked)
	}
}
