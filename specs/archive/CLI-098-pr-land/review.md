---
spec: "CLI-098-pr-land"
verdict: "PASS"
reviewed_sha: "d59dcecfbe08504d55cec021e075b71493d2be4c"
reviewer: "nan/deepseek-v4-flash"
date: "2026-10-13"
---

## Adversarial review

**Scope**: `CLI-098-pr-land` — `dotf pr land`: the single-PR decision, the
`--wait` / `--update-branch` loop, the transactional retarget of dependents, the
queue, and the per-repository kernel lock.
**Sources**: `specs/CLI-098-pr-land/spec.md` (the only spec artifact: FTD lane, no
`proposal.md` / `tasks.md` / `features.json` — `review-request.json` records all
three `contract_digests` as empty, so no contract set is measured for
staleness on this spec); diff `git diff
1392964c34405d72096dbe8f076c6f8f56ffa962...HEAD`; the pr-land surface read in
full: `cli/internal/prland/{prland.go,queue.go,lock.go}`, `cli/internal/filelock/*`,
`cli/internal/cmd/pr_land.go`, and every `*_test.go` beside them; `docs/lessons/lesson-344-*.md`,
`docs/runbooks/release-dotf.md`, `.github/merge-grant.yml`, `harness/reviewer-pool.json`.

Scope note (stated, not hidden): the launcher's base `1392964c` is 2026-10-06
(#2033), so `base..HEAD` is the whole of `main` since then — 1278 files, 800+ commits,
not one spec's work. The adversarial pass was applied to the CLI-098 surface named
above; unrelated paths in that union were **not** reviewed.

### Spec and task alignment

- The spec's checklist is fully implemented. Every one of the 23 test names the spec
  names (`TestLandQueue_*`, `TestAcquireLock_*`, `TestPrLand_*`, `TestLand_*`) exists
  in the tree, by name — verified by `grep -n "func Test"`, not asserted.
- All `[x]` checklist claims map to a named test or to a reproduced command; no
  `[x]` without diff evidence.
- Doctrine alignment, cross-checked: 
  `.github/merge-grant.yml` content against `Decide` — conditions `ci-green`,
  `no-conflicts`, `reviewer-triaged` map to `Checks`/`MergeState`/`Untriaged`, and
  `excluded: release-please` to `releasePrefix`. `TestDecideRefusesForEveryDeclaredCondition`
  drives the same map from the file's own keys, so a grant that names a condition
  `Decide` does not enforce fails in CI. This is the strongest part of the change.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in the spec.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | spec vs code | The grant gate is a *refusal* condition of `pr land` but appears in no spec for the command: `Decide` appends "the repository has not opted in to the standing merge grant (…)" whenever `!GrantDeclared`, so `pr land` refuses **every** PR in a repo without `.github/merge-grant.yml` on the base branch. The spec's Scope enumerates the required conditions (open, not draft, not release-please, checks, CLEAN, triage) and its ticked checklist claims to name "each failed condition"; neither mentions the grant. A reader of the spec would conclude the command lands in any repo. | `prland.go` `Decide` (grant branch) + `grantDeclared`; `spec.md` "Scope" items 1–6 and "Checklist" refusal list; `.github/merge-grant.yml` landed by #2229, which is inside `base..HEAD` | `TestDecideRefusesForEveryDeclaredCondition` (`the grant itself`), `TestLand_RefusesInARepositoryWithoutTheGrant`, `TestTheRepositoryGrantIsExactlyTheDecidedOne` | spec artifacts(only `spec.md` exists; it is *not* in this spec's staleness set — `contract_digests` are empty — so editing it does not invalidate this review) or a follow-up ticket |
| Minor | THEORETICAL | resilience / message | When the lock is held but the recorded PID is a dead one, the refusal says "its PID is not recorded yet" instead of naming the stale PID it just read. Reachable in the microsecond window after a take-over (new holder holds the kernel lock, has not yet written its PID) or if the PID file is unreadable. Fail-closed — it refuses — but the diagnosis is wrong exactly when a human is reading it. | `lock.go` `AcquireLock`: the `ErrLocked` branch refuses with "its PID is not recorded yet" for `pid == 0` **or** `!alive(pid)`; the take-over note only runs on the free-lock path | UNTESTED (`TestAcquireLock_ALivePIDWithAFreeLockIsNotATakeOver` covers the mirror case, not this message) | code + tests |
| Minor | THEORETICAL | resilience | A `gh pr merge` that reports failure *after the merge landed* (deletion of the head branch refused, or a client-side timeout) makes `Land` return an error and then `restoreDependents` point the dependents back at a head branch that no longer exists; their `pr edit --base` then fails too, so the caller sees a failed landing for a merged PR. Impact is bounded: the retargeted dependents stay on the target branch, which is where they belong once the branch is gone. | `prland.go` `merge`: any non-nil `gh pr merge` error joins `restoreDependents`; the spec's own read-only run shows gh's non-zero exit being treated as "not merged" | UNTESTED (the merged-then-failed-delete case; `TestLand_AFailedMergeRestoresTheRetargetedDependents` covers the genuine failure) | code + tests |
| Minor | SPECULATIVE | contract / docs | `--registry` defaults to the repo-relative `harness/review-attestation.json`, so `pr land --repo other/name` read from another cwd silently resolves the default against *this* checkout (or fails to find it). No user-visible failure beyond a refusal, and the flag exists to override it. | `cmd/pr_land.go` `StringVar(&registry, "registry", filepath.Join("harness", "review-attestation.json"), …)`; `prLandOptions` → `prtriage.LoadRegistry(registry)` | UNTESTED | code or docs |

### Test-deletion review

`base..HEAD` deletes many test files (`tests/vault-health*.bats`, `tests/golden/**`,
`tests/check-lessons.bats`, `tests/guard-setup-hooks-order.bats`, …). Those belong to
the vault-health Go port and the setup/harness convergence of other tickets, not to
CLI-098; no pr-land test is deleted or weakened (`git log` over the surface shows the
guard being added and strengthened — `tests/cli-go-tests-unfiltered.bats` exists
precisely so a `cli/**` path filter cannot skip these Go tests). Not assessed further
here, and stated rather than assumed.

### Reproduction (commands run in this session, in this worktree)

- `go test ./internal/prland/ ./internal/cmd/ -count=1` → `ok …/prland 0.201s`, `ok …/cmd 9.886s`, exit 0.
- `GOOS=windows go vet ./...` and `GOOS=darwin go vet ./...` → exit 0 both.
- `golangci-lint run ./...` → `0 issues` (exit 0), **with one warning**: the runner could
  not parse `../../dotfiles-wt-2350/cli/internal/prtriage/execgh_test.go` (a deleted sibling
  worktree) through a stale path, so the 0-issue result should be re-taken with a cleared
  cache before it is quoted as evidence.
- Mutation checks, each reverted; working tree clean apart from this review:
  - drop `--match-head-commit` from the merge → `TestLand_*` FAIL (`want: pr merge 30 --squash --delete-branch --match-head-commit abc123`).
  - reverse `LandQueue`'s loop → 3 `TestLandQueue_*` FAIL.
  - drop the grant refusal from `Decide` → `TestDecideRefusesForEveryDeclaredCondition` and `TestLand_RefusesInARepositoryWithoutTheGrant` FAIL.
  - drop `restoreDependents` after a failed merge → `TestLand_AFailedMergeRestoresTheRetargetedDependents` FAIL.
  - (the head-moved mutation fails to compile — `again` unused — so that guard is proven by
    `TestLand_RefusesWhenTheHeadMovesBetweenReads` only; recorded as a weaker check.)

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | Every spec criterion is implemented and has a named test; negative paths (draft, closed, release-please, pending/failing/no checks, BEHIND, DIRTY, untriaged, unreadable triage queue, moved head, unreadable grant) are covered and four of them mutation-verified in this session. |
| Verification       | A | Evidence is reproducible commands I re-ran myself (tests, two `go vet` targets, lint), and the spec names its tests by identifier rather than "the suite". |
| Scope              | B | The command's change mixes the grant gate (#2229) into the surface the spec describes without recording it there; no unrelated code crept into the pr-land files. |
| Reliability        | B | Fail-closed on every unanswerable read; the retarget/merge step is genuinely transactional and the lock is a kernel lock released on any exit. Two untested edge paths (F2, F3) keep this from an A. |
| Maintainability    | A | Functions are short, comments explain *why* (the #numbers and the measured waits), `filelock` is a clean seam, lint is clean, no dead code. |
| Handoff-readiness  | B | Lesson 344 and the `release-dotf.md` runbook section exist and match the code; the one gap is the spec not naming the grant condition the code enforces. |

### Verdict
PASS

No Blocker; no REAL Major; no dimension graded C or D. The four findings are tracked
below, three of them THEORETICAL/SPECULATIVE and one a REAL but low-risk
contract-completeness gap in a spec artifact that is outside this spec's staleness set.

### Recommended next steps

(All four are outside the contract set as this spec measures it — `contract_digests` are
empty — so none of them invalidates this verdict. Disposition each in the spec's record
or carry it into a follow-up ticket.)

1. **F1** — record the standing-merge-grant requirement in `specs/CLI-098-pr-land/spec.md`
   (Scope conditions and the checklist's refusal list), or file a ticket that says a
   `pr land` reader is expected to learn the gate from `.github/merge-grant.yml` + #2229.
2. **F2** — either name the stale PID in the `ErrLocked` branch of `AcquireLock` or say
   "the holder's PID is not readable"; add a test for that branch.
3. **F3** — distinguish "merge landed, branch deletion failed" from "merge failed" so a
   landed merge is not reported as a refusal, and cover it with a fake-`gh` test.
4. **F4** — resolve `--registry`'s default against the repository being landed (or document
   that the default is cwd-relative).

Closing pass (reviewer side, Definition of Done): Evidence — run, above. Review — N/A, this
run does not comment on the PR. Board — N/A, read-only. Debt / Knowledge — carried as the
four findings and step 1–4, none silently dropped.

**UNVERIFIED in this run** (not findings, gaps in my coverage): no live `gh` call was made,
so nothing here is validated against GitHub's real API responses — the spec's read-only
`dotf pr land 2033` measurement is taken as reported; the Windows lock implementation
(`filelock/lock_windows.go`) was vetted by reading and `GOOS=windows go vet` only; the
`--wait` interaction with a real pending check was not exercised; whether
`dotf spec archive` accepts a folder holding only `spec.md` was not tested.
