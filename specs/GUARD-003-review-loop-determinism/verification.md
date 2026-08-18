---
tags: [spec, verification, templates]
created: "2026-08-18"
---

# Verification - GUARD-003-review-loop-determinism

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] Criterion 1 -> commit `<hash>` / test `<name>`
- [ ] Criterion 2 -> commit `<hash>` / test `<name>`
- [ ] Criterion 3 -> commit `<hash>` / test `<name>`

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

- [ ] Lesson for the repo's `docs/lessons.md`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

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
