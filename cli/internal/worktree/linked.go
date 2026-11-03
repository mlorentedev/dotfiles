package worktree

// LinkedWorktrees reads the linked worktrees out of `git worktree list
// --porcelain` output: every entry but the first, which git always gives as the
// main checkout. The main checkout is no worktree to tear down, and its branch
// (main) is a name an operator might type.
func LinkedWorktrees(porcelain string) ([]RawWorktree, error) {
	all, err := ParsePorcelain(porcelain)
	if err != nil || len(all) == 0 {
		return nil, err
	}
	return all[1:], nil
}
