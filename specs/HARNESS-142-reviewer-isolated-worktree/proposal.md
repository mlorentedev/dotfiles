---
id: "HARNESS-142-reviewer-isolated-worktree"
type: spec
status: draft # draft | implementing | verifying | archived
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

[AGENT-DRAFT — review before archive]

- `dotf spec review` creates a disposable, detached worktree at the reviewed commit, outside the caller's checkout, and launches the reviewer there for both runners (agy `--add-dir` and the process working directory point at it; pi runs in it).
- When the run ends, by exit, deadline or stall, `spec deadline` copies back only the review's own outputs (`review.md`, the transcript, its `.stderr`) into the caller's `specs/<id>/`, then removes the worktree. `review-request.json` stays written in the caller's tree as today.
- The stray-write check from #3117 runs against the worktree, so a reviewer that wrote outside `specs/<id>/` is still named and still exits 4, but nothing reaches the caller's checkout.
- The misleading comment in `agyReviewerCommand` ("in an isolated worktree") becomes true.

## Out of scope

[AGENT-DRAFT — review before archive]

- Reviewing uncommitted work: the reviewer sees the committed `HEAD`, which is already what `reviewed_sha` records.
- A filesystem sandbox beyond git isolation (agy `--sandbox`, AppContainer on Windows): unchanged.
- Changing the reviewer prompt, the pool, or the stall/deadline semantics (#2979 is closed on the single window).

## Risks / open questions

[AGENT-DRAFT — review before archive]

- MUST resolve before tasks: where the worktree lives. Proposal: under `$WORKTREE_ROOT/<repo>/review-<spec-id>-<short-sha>`, created through the existing `worktree` package so the one-root layout (#1222) and the lease rules apply, versus a plain `os.MkdirTemp`. The first is consistent; the second cannot collide with a human worktree.
- MUST resolve: cleanup when `spec deadline` itself is killed (tmux session closed, machine slept). A leftover review worktree must be recognisable and reapable (`dotf worktree list` shows it; `sweep` removes it), never silently accumulate.
- Toolchain caches: a fresh worktree has no `node_modules`/venv. Go and bats work from shared caches; a repo that needs `make worktree-init` (kubelab) would review slower or fail its suite. Decide whether the launcher runs a declared init step or the prompt says so.
- Windows: removing a worktree whose files a reviewer process still holds open fails. The removal must run after the process tree is gone (it already is, by #1851's termination) and report, not hide, a failed removal.

## Acceptance criteria

[AGENT-DRAFT — review before archive]

- [ ] A review run whose reviewer creates `tests/probe_test.go` and edits a tracked file leaves the caller's `git status --porcelain` byte-identical to before the run, and `specs/<id>/review.md` and the transcript arrive in the caller's tree. Verified by an integration test with a fake reviewer that does both.
- [ ] The same run still names both paths on stderr and exits 4.
- [ ] After a run ends by exit, deadline (124) or stall (125), no review worktree remains in `git worktree list`; a run whose cleanup was interrupted leaves a worktree that `dotf worktree list` marks as a review worktree and `dotf worktree sweep` removes.
- [ ] The reviewer's working directory is the worktree for both runners: the agy argv carries `--add-dir <worktree>`, and a pi run's `pwd` is the worktree. Verified by unit tests on the built argv and the launch spec.

## References

- Issue: mlorentedev/dotfiles#1649 (option 1; option 2 shipped in #3117)
- Code: `cli/internal/spec/review_launch.go` (`agyReviewerCommand`), `cli/internal/cmd/spec.go` (`spec deadline`), `cli/internal/spec/stray_writes.go`
- Related: #955 (review independence is convention), #1222 (one worktree root), #1851 (Windows reviewer process trees)
