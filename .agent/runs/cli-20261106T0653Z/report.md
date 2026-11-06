# run cli-20261106T0653Z

**Ended queue-empty after 4 items.** 2 PRs merged (#3230; #3224, see below), 1 closed as already fixed (#3077), 2 parked waiting on the owner (#3114, the second half of #3122). Nothing irreversible is waiting on a human, and both merges can be reverted.

## Waiting on you

1. **#3122, second half: approve `skip-sdd` or wait for a spec slot.** It drops an inherited `GIT_DIR` (and six siblings) from every `RealGitRunner` git call. Without that, a hook's `GIT_DIR` points list, status and merge checks at another repository. By spec-gate's count it is 53 lines; 22 of them are 11 call sites moving to one helper. The commit is ready: `69daf4d7` on `fix/worktree-runner-gitdir-env`, in worktree `fix-worktree-runner-gitdir-env`. The pre-push hook refuses it without the label, so **it exists only in that worktree: do not remove it.** It has a test that fails without the fix. The options are on #3122.
2. **#3114: approve `skip-sdd` or wait for a spec slot.** It is about 69 lines and changes what `dotf worktree done <name>` accepts, a public contract. It is not written; the prototype in the ticket is the plan. The options are on #3114.
3. **#3185 (from before this run):** A or B on the issue.

## Done

- **#3224 merged, closing #3124.** `dotf lessons renumber` keeps a lesson's front matter as written and changes only the number, including the id's quoting and spacing. Before this, it re-quoted kubelab's ids and turned kubelab's CI red.
  - pr-agent reviewed `a62f0fea` and found an unspaced `id:` gaining a space. That is fixed in `f21c04d3`, which was not re-reviewed.
  - The PR author did the triage. CodeRabbit did not review.
- **#3230 merged** (Refs #3122). done's known-worktrees note now reads git's worktree list through a package var, and a table test covers its four outcomes, including the two failure paths that had no test.
  - pr-agent reviewed it and found no major issues; its two compliance notes are dispositioned on the PR. The PR author did the triage.
- **#3077 closed as already fixed.** #3058 and #3096 had fixed it on main. `TestParseOrigin_ReadsAnSSHHostAlias` pins its exact origin.

## Filed (found while working)

- #3232: doctor and the worktree runner each keep a list of git's repository-location variables. Merge them once both packages are in scope.

## Out of the queue

- #3086 was dropped at arming: its fix is in the vault's autopilot skill, outside this run's write paths.

## Incidents

- **A command ran in the shared checkout.** A `cd` into a worktree that `worktree add` had refused to create failed, and the `git checkout <sha> -- <files>` and `git commit` after it ran in `~/Projects/dotfiles`, where an earlier `cd` in the same command line had left the shell. Checked right after: `main` was clean, at the same commit, with no new commit and no change. Later commands guard the `cd` with `|| exit`.
- **GraphQL ran out twice.** The budget is shared with every session. Board edits and one `dotf pr land` waited for the reset; nothing was merged by hand.
- **The first wake-lock was killed** (exit 144) about an hour in, and was restarted. System sleep is set to 0 on this Mac mini, so the run was not exposed.

## Evidence index

- Baseline before any change: `go test ./internal/lessons/... ./internal/worktree/... ./internal/cmd/...`, all ok.
- #3224: `go test ./internal/lessons/` ok; golangci-lint 0 issues; vet clean on darwin and windows; CI green on `f21c04d3`.
- #3230: `go test ./internal/cmd/` ok; golangci-lint 0 issues; CI green on `2e7f4220`; merged at `2e7f422`.
- `69daf4d7`: `go test ./internal/worktree/` ok; its GIT_DIR test fails with the scrub removed (mutation run).
