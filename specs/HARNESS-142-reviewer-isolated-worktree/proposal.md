---
id: "HARNESS-142-reviewer-isolated-worktree"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-04"
issue: "mlorentedev/dotfiles#1649"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# HARNESS-142: Reviewer isolated worktree

## Why

<!-- from issue #1649: HARNESS-142: the spec-review launcher lets a reviewer write into the reviewed tree, and nothing notices or cleans it -->

`dotf spec review` runs the reviewer with unattended shell in the caller's own checkout (`--add-dir <repoRoot>` for agy, the same reach for pi). A reviewer that mutates code to probe it, or leaves scratch files, writes into the tree the author is still working in. In garsync a leftover probe test turned the repository's own suite red, and a `git add -A` would have shipped it. #3117 made those writes loud (named on stderr, exit 4), but loud is after the fact: the author still has to clean the tree, and a reviewer can still edit the diff it is judging while it judges it.

## What

- `dotf spec review` creates a disposable worktree, detached at the reviewed commit, under the one worktree root (#1222) through the `worktree` package, and launches the reviewer there for both runners: agy's `--add-dir` and tmux's start directory point at it, so pi runs in it too.
- The review worktree is marked as such in its metadata: no issue lease (`Issue: 0`, so it never blocks the author's own `worktree add --issue` as an idle holder, #3138), `ReapOK: true` with a lease of the deadline plus a margin, and a kind marker so `dotf worktree list` labels it and `done` does not read it as someone's work. Its name carries the spec id, the review slot and a nonce, so concurrent arms on one spec do not collide.
- Before launch, the spec folder's uncommitted content is copied into the worktree, so the reviewer sees what it sees today: an unstaged `proposal.md`, the earlier rounds that rotation kept (`review-round1.md`, …) and the transcript the reviewer is asked to read. Rotation runs in the caller's tree before the copy, as today.
- When the run ends, by exit (0 or 4), deadline (124) or stall (125), `spec deadline` copies back only the review slot's own outputs (the slot's review file, `review.md` or with `--second` `review-second.md`, its transcript and the transcript's `.stderr`) into the caller's `specs/<id>/`, then removes the worktree. The slot's request file is written in the caller's tree, as today.
- The stray-write check from #3117 runs against the worktree, so a reviewer that writes outside `specs/<id>/` is still named and still exits 4, but nothing reaches the caller's checkout.
- `dotf spec review` refuses to launch when the caller's tree has any change outside `specs/<id>/` that `git status` reports, tracked or untracked (ignored files excepted), unless `--allow-dirty`: the reviewer reviews the commit and would no longer see them, and that must not happen silently. Uncommitted source is not copied into the worktree, because that would stop the review being a review of the commit. The check reuses `SnapshotTree`, which already excludes the spec folder.
- The reviewer prompt says the tree is a fresh worktree at `<sha>` with no build artifacts, and to install what the suite needs before running it.
- The comment in `agyReviewerCommand` that already claims "an isolated worktree" becomes true.

## Out of scope

- A repository-declared init step the launcher runs before the reviewer: #3177. This spec uses the prompt route, and its first review on a repository with a real init is the evidence #3177 decides from.
- A filesystem sandbox beyond git isolation (agy `--sandbox`, AppContainer on Windows): unchanged.
- The reviewer prompt's other content, the pool, and the stall and deadline semantics (#2979 closed on the single window).

## Risks / open questions

Resolved before tasks:

- Where the worktree lives: under the worktree root through the `worktree` package, not `os.MkdirTemp`. A temp directory stays registered in `.git/worktrees`, nothing lists it, `git worktree prune` skips it while the directory exists, and macOS `/var/folders` outlives reboots, so a crashed cleanup accumulates silently. Under the root, `list` shows it and `sweep` reaps it.
- Repositories that need an init step: the prompt route here, the declared step in #3177.
- Sweep's ordinary gates (merged pull request, clean tree) can never pass for a branchless review worktree that an interrupted review left untracked output in: a review worktree is reaped once its lease passes and it is clean or its output was copied back (`output_saved`), with `--force`; an unreadable state keeps it, and Gate f still refuses one a reviewer is running in (found while implementing; see `verification.md`).

Still open, each covered by a task:

- `AddRunner` has no detached-add path; it gains one on the interface rather than a shell-out beside it.
- Windows: removing a worktree whose files a reviewer still holds fails. Removal runs after the process tree is gone (#1851's termination) and reports a failed removal rather than hiding it; `sweep` is the backstop.

## Acceptance criteria

- [ ] AC1. A review run whose reviewer creates `tests/probe_test.go` and edits a tracked file leaves the caller's `git status --porcelain`, restricted to paths outside `specs/<id>/`, byte-identical to before the run; inside `specs/<id>/` the only change is the slot's sanctioned outputs (review file, transcript, `.stderr`, request file), and the review file and transcript arrive. Verified by an integration test with a fake reviewer that does both.
- [ ] AC2. The same run still names both paths on stderr and exits 4.
- [ ] AC3. After a run ends by exit, deadline (124) or stall (125), no review worktree remains in `git worktree list`, and the slot's outputs were copied back in every case, `review-second.md` for a `--second` run. A run whose cleanup was interrupted leaves a worktree that `dotf worktree list` labels as a review worktree, holds no issue lease, and `dotf worktree sweep` removes once its lease passes and its output was copied back, or it is clean. A worktree whose output was never copied back is kept and listed as dirty, so sweep never deletes the only copy of a verdict.
- [ ] AC4. The reviewer works in the worktree for both runners: the agy argv carries `--add-dir <worktree>`, the tmux launch starts in it, and the prompt names it as a fresh worktree at the reviewed sha. Verified by unit tests on the built argv, the launch and the prompt.
- [ ] AC5. `dotf spec review` refuses to launch on a caller tree with a tracked change or an untracked file outside `specs/<id>/`, names the paths and `--allow-dirty`; with the flag it launches and says the reviewer will not see them. An uncommitted change inside `specs/<id>/` does not refuse.
- [ ] AC6. A second-round first-signer review in the worktree finds the caller's uncommitted first-round verdict as `review-round1.md`, and an uncommitted `proposal.md` edit, in the worktree's `specs/<id>/`.

## References

- Issue: mlorentedev/dotfiles#1649 (option 1; option 2 shipped in #3117)
- Code: `cli/internal/spec/review_launch.go` (`agyReviewerCommand`), `cli/internal/cmd/spec.go` (`spec deadline`), `cli/internal/spec/stray_writes.go`
- Related: #955 (review independence is convention), #1222 (one worktree root), #1851 (Windows reviewer process trees)
