---
tags: [spec, verification, templates]
created: "2026-11-11"
id: "BUG-166-review-base-widen"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-166-review-base-widen

## Evidence

- [x] AC1 -> `TestReviewBaseWidensToTheForkPoint` (base is the fork point, `git diff --stat` holds `x.go`), `TestOwnCommitsCountTheWorkBeforeTheSpec` (the fix, which names no issue, is own: 2 of 2).
- [x] AC2 -> `TestReviewBaseNeverNarrows`: on main after a squash, main merged into the branch after the spec (the HARNESS-142 shape), and no `origin/*` ref; each keeps the traced base and an empty `WidenedFrom`.
- [x] AC3 -> the existing `TestResolveReviewBase*` and `TestResolveReviewBaseDetail*` tests pass unchanged. None of them has an `origin/*` ref, so they exercise the trace alone, exactly as before.
- [x] AC4 -> `TestReviewScopeSaysTheBaseWasWidened` (launch line and prompt), plus a dry run of a dev build on a repository of the #3258 shape: `Scope: git diff 112faefd...HEAD (2 of 2 commits are the spec's own; ...; base widened from 01949862f953 to where the branch left the default branch)`, and `git diff --stat <base>...HEAD` lists `x.go`. The dry run created no worktree.

## Test status

- `cd cli && go build ./... && go vet ./... && go test ./...`: all packages ok.
- `golangci-lint run ./...` and `GOOS=windows go vet ./... && GOOS=windows golangci-lint run ./...`: 0 issues.
- Mutation checks: dropping the `isAncestor` guard fails `TestReviewBaseNeverNarrows`; disabling the widening fails `TestReviewBaseWidensToTheForkPoint` and `TestOwnCommitsCountTheWorkBeforeTheSpec`.

## Decisions made during implementation

- The fork point uses the same ref order as `dotf worktree add` (`origin/HEAD`, `origin/main`, `origin/master`), held in the `spec` package because `worktree` already imports `spec`.
- The widening lives in `ResolveReviewBaseDetail`, so the recorded `base_sha` is the widened base, and the archive gate needs no change.
- The work before the spec folder counts as own through a third rule in `OwnCommitsWidened`, not through the scope line alone: the fix the widening exists to include often names no issue.

## Review dispositions

The first review (nan/mimo-v2.6-flash, PASS-WITH-GAPS, three Minors) at 6a18a191:

1. Prompt rule text under widening (Minor, real): applied. `writeOwnCommits` names the third rule ("or come before the spec folder on this branch"), and the closing line no longer lists only two rules. Test: `TestReviewPromptNamesTheWidenedOwnRule`.
2. `forkPoint` stops at the first existing ref whose merge-base fails (Minor, speculative): applied. It tries the next ref instead; the failure mode stays the safe one.
3. No committed tests for the stale-ref and rename-plus-widen compositions (Minor): applied. `TestReviewBaseWidensPastAStaleOriginRef` and `TestReviewBaseWidensAfterTracingARename`.

None of these touches `proposal.md`, `tasks.md` or `features.json`.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the spec, its tests and the resolver's doc comment carry the rule (widen only, never later); there is no operating gotcha beyond it.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: it changes how one resolver picks a commit, not a contract between components.
- [x] New pattern candidate for `00_meta/patterns/`? no: one occurrence, in one tool.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-166-review-base-widen/` -> `specs/archive/BUG-166-review-base-widen/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
