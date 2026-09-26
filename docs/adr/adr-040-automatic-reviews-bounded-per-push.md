---
id: "ADR-040-automatic-reviews-bounded-per-push"
type: adr
status: accepted
owner: manu
date: "2026-09-25"
supersedes: []
extends: [adr-037-review-runners-bounded-and-budgeted]
issue: mlorentedev/dotfiles#1756
tags: [architecture, decision, review, pr-agent, ci, inference]
created: "2026-09-25"
---

# ADR-040: An automatic PR review runs in full once, then incrementally past a bound

## Context

pr-agent ran a full `/review` on every push to a pull request. #1053 put it there so that a fix pushed after a review would be read too.

Measured from 2026-09-19 to 2026-09-26:

- 39 reviews across 18 PRs, 2.2 per PR on average, and 7 on one PR (#1732: the opening plus six pushes).
- Each review is one request to a NaN pool of 5 concurrent slots, shared with pi, qq, hive and the spec reviews (#1107).
- #1732's seventh review failed: the primary model timed out after 361 s, and the fallback hit the concurrency limit.
- Every re-review edited the persistent Guide comment, so the triage queue reopened for a re-triage even when nothing new had been found.

Comparable tools bound this:

- Copilot reviews once unless "Review new pushes" is on.
- Codex reviews on open, or on `@codex review`.
- Claude Code Review defaults to once, after the PR is created.
- CodeRabbit reviews each push incrementally and pauses after five reviewed commits.

ADR-037 bounded how long one review may run. Nothing bounded how many reviews one PR could trigger.

## Decision

1. **A full review runs when a PR is opened, reopened or marked ready,** and whenever someone comments `/review`.
2. **A push is reviewed incrementally** (`/review -i`: the commits since the previous review), and only once **three non-merge commits** are newer than that review.
   - `scripts/pr-agent-push-gate.sh` decides this before PR-Agent starts.
   - "The previous review" is the one PR-Agent itself picks. The gate mirrors its rule, so the gate never starts a run that PR-Agent would then decline.
   - The gate fails open: input it cannot read means the push is reviewed.
3. **Below the bound nothing runs, and nothing reports.** The published-review guard (#1107) is skipped with PR-Agent, so a push held back is not mistaken for a failed inference. The guard, the triage queue and the attestation all recognise the incremental heading through the reviewer registry.
4. **Agents iterate on a draft PR** and mark it ready when it is ready. Reviewers skip drafts, so the full review reads the finished change. `pr-stewardship` carries this instruction to every agent.

## Consequences

- #1732's seven reviews would have been two or three.
- A PR that is ready gets one full review, and a fix that follows gets a smaller, incremental one.
- A one-commit fix after a review is not reviewed automatically. Its author can ask with `/review`, and the triage owed on the PR still applies.
- `/review -i` is undocumented upstream: its docs have been commented out since v0.24. The design reads the pinned v0.45.0 source, and the tests pin the headings and identities it depends on. A version bump that changes them fails in `tests/pr-agent-config.bats` and `tests/pr-agent-push-gate.bats`, not in production.
- The pr-agent ports in other repositories keep reviewing every push until they adopt the gate.
- Lesson 307 records the guard hazard this nearly shipped.
