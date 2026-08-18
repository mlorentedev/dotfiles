---
tags: [spec, tasks, templates]
created: "2026-08-16"
---

# Tasks - GUARD-002-review-attestation

> TDD order. One task = one focused commit. `[AC<n>]` maps a task to an acceptance criterion in `proposal.md`; `[P]` marks a task with no dependency on another unchecked one.

## Setup

- [x] Branch created from main: `feat/review-attestation-gate`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — the
      `pending` exit-code question is explicitly **deferred**, not open: the
      message names the state and no caller branches on it in v1

## Implementation

Ordered so the classifier is provably correct against captured reality before any
workflow depends on it. The fixtures come first because the entire value of this
gate is telling two shapes apart, and one of those shapes is a real vendor
artifact nobody can reproduce on demand once the quota resets.

- [x] [AC1] [P] Capture the three live rate-limit payloads (#1007, #1009, #1013)
      verbatim into `tests/fixtures/review-attestation/pr-*.raw.json` — captured
      2026-08-16, all three carrying the `rate limited by coderabbit.ai` marker.
      Banked rather than re-fetched on demand: the comments do persist on closed
      PRs, but a fixture is the only form of this evidence that survives the
      vendor changing its output
- [x] [AC1] Derive minimal, hand-checkable fixtures from the raw captures (one
      per state), plus a real-review and a human-review payload
- [x] [AC5] Declare reviewers in a config file (login + the marker identifying its
      "could not review" notice), CodeRabbit as the first entry, with a comment
      naming #786 as the intended second
- [x] [AC1] [AC2] [AC3] `scripts/check-review-attestation.sh`: classify
      `attested | declined | pending` from a PR JSON payload, offline, reading the
      reviewer list from config. Observe each state red before making it green
- [x] [AC6] Fail closed on unreadable input — no `gh`, unauthenticated, malformed
      JSON, empty response. Assert with a test that feeds each shape
- [x] [AC4] The declared escape: `merged-unreviewed` label **and** a non-empty
      `## Unreviewed merge rationale` body section. Three negative tests (label
      only, section only, empty section) before the positive one
- [x] [AC7] Thin CI caller on `pull_request` + `issue_comment`, mirroring
      `spec-gate.yml`'s shape, non-required in branch protection
- [ ] Report the gate's own state on this PR, as the first real exercise

## Closing

- [x] Every acceptance criterion is covered by at least one feature with a
      non-vacuous verification command
- [x] Every acceptance criterion has a matching entry in `features.json`
- [x] Lint passes (`shellcheck scripts/check-review-attestation.sh`, `bash -n`)
- [x] Tests pass (`bats tests/check-review-attestation.bats`)
- [x] Non-vacuity measured, not assumed: each classifier branch observed failing
      under a mutation before it was made to pass
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] Adversarial review passes before archive (`dotf spec review GUARD-002-review-attestation`)

## Notes carried from the evidence

The bug this gate exists for is **not** "the reviewer was unavailable" — it is
"the absence of a review was reported as `pass`". Any implementation that ends
with an unreviewed PR reading green has failed regardless of its test count.

`gh pr checks` renders the rate-limited state as `CodeRabbit  pass  0  Review
rate limited`. The word `pass` is the defect, and it comes from the vendor's
check status, which this gate cannot change — it can only add a second, honest
signal beside it.

## Machine-readable features

`features.json` sits beside this file, one feature per acceptance criterion
(f1–f7), each with a shell command whose exit 0 is the pass condition.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the
harness, after running `verification` and capturing exit code 0, may set that
terminal state. Every entry starts `pending` with empty `evidence`.

## A fifth state, after the reviewer changed underneath the gate (#1061)

- [x] `exempt` — a change whose diff carries nothing a review could act on. Added
      because the second reviewer's ticket-compliance analysis is structurally
      wrong on release PRs (it looks for the CHANGELOG's referenced code in a
      diff that cannot contain it), so those PRs were about to be excluded from
      review — and excluding them without this would have manufactured the
      unreachable state of #1061.
- [x] Matched by **diff signature**, never by author or branch. The author of a
      release PR reads as a human because the tool runs under a PAT; a branch
      rule would let any branch named to match walk past. The reviewer's half
      matches on branch and the gate's on file set — two independent conditions,
      so no single string gets a change past both.
- [x] Fails closed on the set-theory identity: an empty file list matches every
      signature under subtraction, which would turn a truncated payload into a
      blanket pass. Guarded by name.
- [x] The pairing itself is guarded, across both files. The first version was not
      — removing the reviewer exclusion broke no test, which is the half-applied
      state the design exists to prevent, present in the change that designed it.
