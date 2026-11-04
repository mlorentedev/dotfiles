## Lanes

Parallel sessions coordinate through the repository, not through chat. The contract is `dotf`'s (ADR-050 in the dotfiles repository):

- **Queue:** `.dotf/lane.json` lists the issues in priority order. `dotf lane status` shows each lane's phase, derived from git and the forge; `dotf lane plan` prints what to do about it.
- **Worktree and branch:** one issue is one branch and one worktree, started with `dotf lane next`, which leases the worktree to that issue.
- **Claim:** a session claims an issue before it starts; two sessions never hold the same one.
- **Landing:** one PR at a time through `dotf pr land <N>`, after `dotf pr triage-queue --pr <N>` is clear. Never `gh pr merge --auto`.
- **Prompt:** `.dotf/lane-prompt.md` is what a successor lane starts from.
