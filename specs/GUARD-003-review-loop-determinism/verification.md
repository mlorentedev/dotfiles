---
tags: [spec, verification, templates]
created: "2026-08-18"
---

# Verification - GUARD-003-review-loop-determinism

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

Re-measured 2026-10-21 on `origin/main` at `856f49fc`, except where a line says
otherwise.

- [x] AC1 -> attestation run 39540679532 (`workflow_run`), 2026-10-21: `re-evaluating PR #2844 because the reviewer finished`, then `[OK] attested`, with no human action. The `/review` path is AC4's T11/T12.
- [x] AC2 -> statuses on `9be46c48` (PR #2844): `03:46:44 failure [FAIL] pending`, then `03:49:37` and `03:49:53 success [OK] attested`. Test `every path still publishes the verdict as a commit status [AC2]`.
- [x] AC3 -> tests `every non-pull_request path still fails [#1041]` and `AC3: a PR with no reviewer output is pending, not declined`.
- [ ] AC4 -> automatic path: run 39540679532 resolved #2844 from its head SHA, and test `a pull_request reviewer run still resolves by head SHA [AC4]`. `/review` path: T11 (#1128) here; live proof is T12, owed after merge.
- [x] AC5 -> `tests/review-attestation-workflow.bats` AC1 cases, observed failing on their mutations (table below, 2026-08-18).
- [x] AC6 -> the adoption order in the header of `review-attestation.yml`; `review-attestation` is a required check on `main` (branch protection read 2026-10-21).
- [x] AC7-AC9 -> `dotf pr triage-queue` (`cli/internal/prtriage`), with the marker read from `harness/review-attestation.json` (`triage.marker`).

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

-
-

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson: yes. `docs/lessons/lesson-502-a-workflow-run-from-a-comment-triggered-run-has-lost-its-pull-request.md`: a comment-triggered run's `workflow_run` names the default branch, so the producer must hand the PR forward.
- [x] ADR: no. The one ordering decision lives in the header of `review-attestation.yml`, beside what it constrains (AC6).
- [x] Pattern: no. This spec already fed `pattern-verification-fails-toward-unproven` (its fourth arrival); no second pattern.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/GUARD-003-review-loop-determinism/` -> `specs/archive/GUARD-003-review-loop-determinism/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)


## Session evidence — the trigger (T1, T2)

Guards written before the trigger and observed red, with their diagnostics:

```
not ok 2 ... [AC1]   no workflow_run trigger: our own reviewer finishing wakes nothing
not ok 3 ... [AC1]   job if: does not admit workflow_run — the trigger would fire and skip
not ok 4 ... [AC4]   grep 'workflow_run.head_sha' failed
```

Green after implementation, 6/6. Every guard then run against its own mutation,
with the mutation's arrival checked by **checksum before and after** rather than
against `HEAD` — on a dirty tree the latter reports an invalid mutation as
applied, which is the false green the exercise exists to detect:

| mutation | landed | guards failed |
|---|---|---|
| `workflow_run` trigger deleted | yes | 1 |
| `workflow_run` names `ci` instead of `pr-agent` | yes | 1 |
| job condition no longer admits `workflow_run` | yes | 1 |
| concurrency key loses its head_sha discriminator | yes | 1 |
| PR resolution removed | yes | 1 |
| **control: a pattern that does not exist** | **no** | 0 (correctly not counted) |

`bats tests/check-review-attestation.bats` — 37/37, no regression.
`actionlint .github/workflows/review-attestation.yml` — clean.

**AC1 is NOT satisfied by any of this.** `workflow_run` fires only for the copy
of a workflow on the default branch, so the trigger cannot be exercised by the
pull request that introduces it. AC1 says "verified on a live PR", and that
verification is owed after merge, on the next PR that PR-Agent reviews. Recorded
here rather than ticked, because a criterion that cannot be met yet is not met.

## A defect prevented, worth its own line

The concurrency group was keyed on `pull_request.number || issue.number`. A
`workflow_run` payload has neither, so every such run would have collapsed into
one group keyed on the empty string — and `completed` is not in the
cancel-exempt list, so a reviewer finishing on one PR would have cancelled the
re-evaluation of another. That is #1040 exactly, one workflow over, arrived by a
different route, and it would have shipped inside the fix for the defect #1040
caused.


## Session evidence — the wake-up (T6-T9)

`dotf pr triage-queue`, run against the live repository:

```
3 pull request(s) awaiting a disposition:

  #1051  docs(spec): archive HARNESS-061-effectiveness-probes (#852)
         github-actions reviewed, never triaged, 2026-08-17 19:05
  #1049  chore(main): release 0.43.1
         github-actions reviewed, never triaged, 2026-08-17 18:47
  #1048  refactor(docs): slim AGENTS.md to crisp boundary rules...
         github-actions reviewed, never triaged, 2026-08-17 18:47
exit=1
```

Three real PRs carrying PR-Agent reviews nobody had acted on — which is the
condition the command exists to surface, found on its first run.

`#1056` was expected in that list and is absent. Checked rather than assumed:
it had merged four minutes earlier, so it is not an open PR. The command was
right and the expectation was stale.

Unit coverage is 8 table cases plus two registry cases, in
`cli/internal/prtriage/prtriage_test.go`: never triaged, triaged, **re-reviewed
after triage** (the case that matters — pushing a fix makes the reviewer
re-review and the old disposition no longer covers it), both spellings of a bot
login, a declared reviewer commenting without a marker, an undeclared author
using one, the marker quoted mid-prose, and a registry with an empty marker
being refused rather than matching everything.

`go build`, `go vet`, `go test ./...`, and golangci-lint at the pinned version:
**0 issues**.

## AC1 — verified by this pull request itself

#1056 merged at 01:39, putting the `workflow_run` trigger on the default branch,
which is the only place it fires from. This PR is therefore the first one it can
act on: PR-Agent reviews it, the trigger fires, and the gate re-evaluates with
no human comment and no manual re-run. The sequencing constraint that made AC1
unverifiable turned into the vehicle that verifies it.

Evidence to capture on this PR: a `review-attestation` run whose event is
`workflow_run`, with no human comment preceding it.


## A decision recorded with its measurement: the check-run stays as it is

Raised by a parallel session as a measurement rather than a ticket, which is the
right shape — this is the spec's own surface. The argument: #1056 exempted
`pending` because a check-run frozen red on every correct PR teaches its readers
to scroll past it, and `declined` should now get the same treatment, because
*"CodeRabbit declines, PR-Agent attests five minutes later"* has become the
standard sequence rather than an edge case.

The argument is good and the measurement does not support it.

**All five open PRs agree between the two signals**, taken at one moment:

```
#1046  check-run=fail   status=fail     (genuinely unreviewed — correct)
#1048  check-run=pass   status=pass
#1049  check-run=pass   status=pass
#1057  check-run=pass   status=pass
#1058  check-run=pass   status=pass
```

**And the case the argument was built on did not stay frozen.** #1059's head sha
carries two attestation check-runs:

```
attestation: failure  @02:24:43
attestation: success  @02:34:38
```

A later `pull_request` run created a fresh one, and GitHub displays the newest
per name. The original measurement was taken in the window between the two —
a real window, and not a permanent state.

So the mechanism exists and its blast radius is narrower than the argument
requires: it bites only when a review lands and no further push follows. #1058
narrows even that, since every push now creates a run and therefore a check-run.

**The decision is to leave `declined` failing, and the reason is not the size of
the window.** It is that the check-run is not the verdict. #1041 established
that and #1056 shipped it: the commit status is the revisable signal, it is what
later runs correct, and it is what branch protection will require (AC6). Making
the check-run track the truth more closely is polishing the signal this spec
deliberately decided not to trust — and every hour spent on it argues, by
implication, that it should be trusted.

Revisit if a PR is ever observed merging with the two signals disagreeing at
merge time. That is the failure that would matter, and it has not been seen.

## T10: the queue honours the exempt signature (#1196), 2026-10-09

Unit tests in `cli/internal/prtriage`:

- `TestEvaluate_AnExemptDiffSignatureIsNotPending`: exactly the signature, one file more, one file fewer, a file list never read, and a duplicate path.
- `TestFetchDropsAPendingPRWhoseDiffIsAnExemptSignature`: also asserts that a quiet PR costs no files call.
- `TestFetchKeepsAPRWhoseFileListFillsAPage`: a full page of files keeps the PR pending.
- `TestTheRealRegistryDeclaresTheReleaseSignature`.

Mutations, each run once and then reverted:

| Mutation | Result |
|---|---|
| Disable the exemption in `Evaluate` | 4 failures |
| Drop the length equality, so only a subset is checked | `one_file_fewer` fails |

Live against `mlorentedev/dotfiles`, with release PR #2168 open and its CodeRabbit notice re-edited after the last triage:

```
$ dotf pr triage-queue                      # released 0.65.0
  #2168  chore(main): release 0.66.0
         coderabbitai reviewed again after the last triage
exit 1
$ $TMPDIR/dotf-exempt pr triage-queue       # this branch
[OK] no reviewer output is awaiting a disposition
exit 0
```


## T11: a review asked for with /review (#1128), 2026-10-21

The defect, live on 2026-10-20: slash run 39365425846 on an open PR finished at
03:42:53, and attestation 39365734960 answered `no open pull request for
fd0be466`, which is main's SHA. On 2026-10-21 one more defect turned up in the
same path. Every comment-triggered reviewer run, skipped ones included,
completes with main's SHA, so they all shared one concurrency group, and
attestation run 39539977021 was cancelled by the next one.

The guards in `tests/review-attestation-workflow.bats` run the gate's real
`Resolve the pull request` step under a stub `gh`. They were observed red before
the change: 12, 13, 18, 19 and 20 failed. Case 13 resolved PR #99 by SHA, which
is the guess the fix must never make. All 20 pass after the change, under bash 5,
zsh and macOS `/bin/bash` 3.2. `actionlint` is clean on both workflows.

Each mutation was checked by checksum before and after, and then reverted:

| mutation | landed | failed |
|---|---|---|
| numeric validation removed | yes | 14 |
| open-state check removed | yes | 15 |
| artifact trusted on every event | yes | 16 |
| issue_comment branch dropped | yes | 12, 13 |
| group loses the run id | yes | 18 |
| `actions: read` removed | yes | 19 |
| upload no longer limited to issue_comment | yes | 20 |
| control: a pattern that does not exist | no | none (correctly not counted) |

Cases 14 and 15 also pass on the old code, which never reads an artifact. The
mutations are what show they are not vacuous on the new code.

**T12 is owed after merge.** `workflow_run` fires only for the copy of the
workflow on the default branch, so this PR cannot exercise its own fix.
