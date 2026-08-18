---
id: "GUARD-003-review-loop-determinism"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-18"
issue: "mlorentedev/dotfiles#1052"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# GUARD-003-review-loop-determinism

> **Naming**: file lives at `<repo>/specs/GUARD-003-review-loop-determinism/proposal.md`. `GUARD-003-review-loop-determinism` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #1052: GUARD-003: the review gate never re-evaluates when our own reviewer finishes, because GITHUB_TOKEN comments emit no events -->

GUARD-002 answers "did a review actually happen" and answers it correctly. What it cannot do is **notice when the answer changes**. GitHub suppresses workflow events for comments created with `GITHUB_TOKEN` to prevent recursion, and PR-Agent — the reviewer this repository shipped to stop CodeRabbit's quota gating throughput — publishes with `GITHUB_TOKEN`. So our own reviewer can produce a complete review and the gate will never look at it, holding whatever verdict it computed twenty seconds after the PR opened, which is `pending` on every PR. Merging #1047 makes PR-Agent's output *countable*; it does not make it *counted*.

## What

Three observable changes, in a required order.

**The gate re-evaluates when the reviewer finishes.** A `workflow_run` trigger on the `pr-agent` workflow completing. `workflow_run` is not a comment event, so it is outside the `GITHUB_TOKEN` suppression that causes the defect. No external infrastructure: GitHub already emits this event.

**The verdict becomes enforceable rather than advisory.** The `review-attestation` **commit status** joins branch protection's required checks, so a merge is refused mechanically until the gate is green. Never the check-run, which cannot be revised after creation (#1041) — a required check-run frozen at `pending` blocks every PR permanently.

**The triage loop gains a wake-up.** `pr-review-triage` already reads the comments, produces a per-item disposition and applies under confirmation; what it lacks is anything telling it the moment has arrived.

The wake-up is a **pull**, not a push, and that is a finding rather than a compromise. CI cannot wake a local agent session: `workflow_run` re-evaluates the gate, and GitHub notifies the human, but nothing reaches an agent. So the mechanism an agent can actually use is a query it runs at a checkpoint — `dotf pr triage-queue`, listing open PRs whose reviewer output is newer than their last recorded disposition.

That requires "dispositioned" to be mechanical rather than remembered, which it currently is not: today the disposition lives in a chat message and evaporates. The triage record becomes a comment on the PR under a stable heading, so the Review item of the Definition of Done stops being a claim and becomes an artifact. The queue reads that marker; the skill writes it; both take the string from `harness/review-attestation.json`, which already owns the reviewer registry.

## Out of scope

- **Applying reviewer comments without confirmation.** In scope: applying them *with* per-item confirmation, which is what `pr-review-triage` already requires. Out of scope: removing that confirmation. Reviewer output is untrusted data — CodeRabbit's own comments say so — and a non-frontier model's "add this line", applied unattended to `.github/workflows/`, `scripts/check-*`, `harness/` or `secrets/`, is privilege escalation of the same shape as the defect #1019 fixed by pinning the gate's checkout to the base branch.
- **Auto-merge, in any form.** The loop ends at a pushed commit and a recorded disposition. The merge stays human, repository-wide rule.
- **Replacing CodeRabbit.** Two reviewers run alongside each other for a bounded window by decision in #786.
- **A pending-to-declined deadline.** Once the gate re-evaluates on reviewer completion, "pending forever" stops being the common case. Revisit only if measurement says otherwise.

## Risks / open questions

- **`workflow_run` fires for the workflow, not for a pull request.** The event carries the run, and the PR association has to be recovered from it. AC4 exists because a trigger that re-evaluates the wrong PR is worse than no trigger.
- **Ordering is load-bearing and the wrong order blocks the repository.** `#1047` -> `#1041` -> this trigger -> required-in-branch-protection. Making the gate required before #1041 lands blocks every PR, because the frozen `attestation` check-run is red on all of them.
- **A trigger must not become a path to green.** It re-evaluates; it never attests. A reviewer that declined, or never ran, still ends red (AC3).
- **Lane collision, to be arbitrated by the owner.** `.github/workflows/pr-agent.yml` is TOOL-013's surface and held by another session; a third agent is active on harness surfaces. This spec's own surface is `.github/workflows/review-attestation.yml` and `scripts/check-review-attestation.sh`, which is GUARD-002's and unheld.

## Acceptance criteria

- [ ] AC1. A PR-Agent review completing causes the gate to re-evaluate, with no human action and no manual re-run, verified on a live PR rather than by inspection.
- [ ] AC2. The re-evaluation publishes to the commit status, and the status carries the latest verdict rather than the first.
- [ ] AC3. A PR whose reviewer declined, or never ran, still ends red after the trigger fires.
- [ ] AC4. The trigger re-evaluates the pull request the reviewer actually ran on, and no other.
- [ ] AC5. A guard fails when the gate's workflow loses its `workflow_run` trigger, and has been observed failing on that specific mutation — including confirmation that the mutation reached the file, since an invalid mutation and an absent guard produce the same green.
- [ ] AC6. The required-check adoption is documented with its ordering constraint, so it cannot be enabled before #1041.
- [ ] AC7. Reviewer output that has not been dispositioned is **findable without anyone remembering to look**: a read-only query lists open PRs carrying reviewer output newer than their last recorded triage. It lists; it never applies.
- [ ] AC8. "Dispositioned" is mechanical, not a judgement: a PR leaves the queue when a triage record newer than the reviewer's output exists on it, and re-enters when the reviewer speaks again.
- [ ] AC9. The queue and the gate read the same reviewer registry. Neither hardcodes a marker string the other also knows.

## References

- Bitácora board: `mlorentedev/dotfiles#1052`
- GUARD-002 (#906, shipped in #1019) — the gate this extends; #1041 and #1045 are its two open defects
- #786 / TOOL-013 — PR-Agent, the second reviewer whose output triggered this
- `00_meta/patterns/pattern-verification-fails-toward-unproven.md` — fourth arrival: a second producer of a consumed signal appears, and the consumer was written when there was one
