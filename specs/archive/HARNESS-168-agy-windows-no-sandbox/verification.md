---
tags: [spec, verification, templates]
created: "2026-09-29"
---

# Verification - HARNESS-168-agy-windows-no-sandbox

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> `TestAgyReviewerCommandOmitsSandboxOnWindows`
- [x] Criterion 2 -> `TestAgyReviewerCommandKeepsSandboxOutsideWindows`
- [x] Criterion 3 -> `TestReviewerCommandDoesNotGivePiAgySpecificFlags`

## Test status

- Fail-first: `TestAgyReviewerCommandOmitsSandboxOnWindows` failed because the
  existing argv contained `--sandbox`.
- Targeted suite: the Windows/no-sandbox, non-Windows/sandbox, permission,
  workspace and pi-isolation tests pass.
- Static checks: `go build ./...`, `go vet ./internal/spec`,
  `golangci-lint run --new-from-rev=main`, `git diff --check`, and
  `jq empty specs/HARNESS-168-agy-windows-no-sandbox/features.json` -> pass.
- Manual smoke test: direct headless agy 1.2.13 with the new Windows argv shape
  ran `git rev-parse HEAD` in the worktree and returned
  `d691f613a7f0f6d79870ab318853046696069078` in 30 seconds, without UAC.
- No regressions: targeted reviewer-command tests pass. The complete
  `internal/spec` suite hit its existing 10-minute timeout in an unrelated
  git-staleness test under concurrent worktree load.
- Review round 1 dispositions:
  - **Applied:** added AC4 to `features.json` with a focused non-vacuous command;
    the real agy probe remains recorded above as live evidence.
  - **Applied:** restored the sandbox assertion in
    `TestReviewerCommandGivesAgyReachIntoTheRepo`, conditional on the platform,
    so non-Windows retention and the Windows exception are both pinned.
- Review round 2 dispositions:
  - **Applied:** removed AC4/F4 from the machine-verifiable contract. A live agy
    probe depends on local OAuth state and Windows execution policy, so presenting
    the F1 unit test as that probe was vacuous. The real probe remains explicit
    manual evidence in this file.
  - **Declined as independently covered:** the top-level wiring test runs on the
    host OS, while the two explicit `agyReviewerCommand` tests execute both OS
    branches deterministically.
- PR review dispositions:
  - **Applied:** archive checklist now reflects the completed mechanical archive
    and promotion decisions. Issue closure intentionally waits for merge.
  - **Declined:** `review.md` names the active spec paths that existed at review
    time. The archive gate subsequently moved those exact artifacts; rewriting
    an independent verdict after the move would corrupt its historical scope.
  - **Accepted risk:** Windows reviews run with standard-user access and
    automatic tool approval. The lost AppContainer boundary is the explicit
    compatibility decision in #1838; process-tree hardening remains separate.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Keep `--sandbox` on non-Windows platforms, where it provides isolation
  without requiring elevation.
- On Windows, omit only the forced sandbox flag. The process remains
  non-elevated and bounded by the reviewer pool, worktree and deadline, but has
  the same user-level reach as direct interactive agy.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? no: the platform-specific security
  tradeoff is recorded in the spec and beside the launcher branch.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: this is a
  compatibility exception within the existing review architecture.
- [x] New pattern candidate for `00_meta/patterns/`? no: this is specific to the
  current Antigravity Windows sandbox implementation.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/HARNESS-168-agy-windows-no-sandbox/` -> `specs/archive/HARNESS-168-agy-windows-no-sandbox/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)

The issue remains open until the implementation PR merges; closing it earlier
would make the board report completion before the code reaches `main`.
