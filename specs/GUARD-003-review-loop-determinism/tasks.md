---
tags: [spec, tasks, templates]
created: "2026-08-18"
---

# Tasks - GUARD-003-review-loop-determinism

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.
>
> **No task here carries `[P]`.** This spec's ordering is not a preference: it is
> blocked in sequence, and starting at the wrong end blocks the repository.


## Blocked on, before any task here starts

- [ ] B1. **#1047 merges** — the classifier learns to count comment-shaped review
      output. Until then there is no verdict worth re-evaluating. Held by another
      session.
- [ ] B2. **#1041 ships** — the verdict lives on the revisable commit status
      rather than a check-run frozen at creation. Making the gate required before
      this blocks every PR.

## Implementation

- [ ] T1. [AC5] Write the guard first: a case asserting the gate's workflow
      declares a `workflow_run` trigger on `pr-agent`. Observe it red before the
      trigger exists, **and assert the mutation reached the file** — an invalid
      mutation and an absent guard produce the same green.
- [ ] T2. [AC1] [AC4] Add the `workflow_run` trigger and recover the pull request
      from the run payload. A trigger that re-evaluates the wrong PR is worse
      than no trigger, so the recovery is the substance of this task, not the
      trigger line.
- [ ] T3. [AC2] Confirm the re-evaluation writes the commit status and that the
      status carries the latest verdict. Depends on B2 having landed the status
      as the authoritative channel.
- [ ] T4. [AC3] A declined or absent review still ends red after the trigger
      fires. Verified by fixture, not by reasoning — the trigger must not become
      a path to green.
- [ ] T5. [AC1] Verify on a **live** PR: a PR-Agent run completing flips the gate
      with no human action and no manual re-run. Inspection does not satisfy AC1.
- [ ] T6. [AC6] Document the required-check adoption with its ordering
      constraint, in the gate's own workflow and in the reviewer registry comment
      block, so the constraint travels with the thing it constrains.

## Closing

- [ ] Every acceptance criterion covered by a verification command whose failure
      mode has been observed
- [ ] Lint passes (`shellcheck scripts/check-review-attestation.sh`, `bash -n`)
- [ ] Tests pass (`bats tests/check-review-attestation.bats`)
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in with output produced in the session, not recalled
- [ ] PR opened referencing this spec folder
- [ ] Adversarial review passes before archive (`dotf spec review GUARD-003-review-loop-determinism`)

## A note on this spec's own surface

The trigger lands in `.github/workflows/review-attestation.yml`, which is
GUARD-002's surface and currently unheld. It does **not** touch
`.github/workflows/pr-agent.yml` — that is TOOL-013's, held elsewhere. The
`workflow_run` trigger names the pr-agent workflow but is declared in the gate's
own file, so the two lanes stay separate and no coordination is required beyond
the workflow's name remaining stable. If that name changes, T1's guard fails,
which is the intended behaviour.
