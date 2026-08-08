---
id: "HARNESS-027"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-08"
issue: "mlorentedev/dotfiles#411"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, skill, cross-agent]
template_version: "1.0"
---

# HARNESS-027

## Why

<!-- from issue #411: HARNESS-027: post-PR review-triage skill — wait for CI + CodeRabbit, evaluate suggestions -->

Every change here ships through a pull request, and after opening one there is a manual, repeatable gap: wait for the checks, read what the reviewers said, decide what to do about each remark. Nothing marks when to come back, so the step is skipped exactly as often as it is remembered — and a review nobody dispositions is a review nobody paid for. It is also the **Review** item of the Definition of Done (HARNESS-056), which needs something executable behind it rather than an instruction.

## What

A cross-agent skill that, on demand or once a PR's checks settle: watches the run once, reports CI honestly (naming the failing check and the command to read its log), waits for asynchronous review comments, confirms a review actually ran, then gives every substantive comment one of three dispositions — **apply** in scope, **defer** to a ticket, **skip** with a reason — presents the table, and acts only after human confirmation. It never merges.

The common case is nothing to do, and the skill says so in one line rather than manufacturing a table for an empty review.

## Out of scope

- Merging, under any condition. Merge is supervised and auto-merge is forbidden repository-wide.
- Applying suggestions without confirmation, in bulk or otherwise.
- Reviewing someone else's PR — that is review, not triage of your own change.
- Any binding to a particular review vendor. Who wrote a comment does not enter the judgement.

## Risks / open questions

- **Vendor coupling.** Naming today's review service would hardcode it into an agnostic skill and rot the moment it changes — the user's own automations are expected to post comments in future. Resolved by triaging on **surface and content**, never on author.
- **Status notices read like passes.** A comment landing seconds after the PR opens is usually a quota or queue notice, not a review. Measured on this repository: a reviewer posted 9 seconds after creation with "Review limit reached — next review available in 11 minutes". Treating that as a clean review claims coverage that never existed.
- **Ceremony.** A triage procedure that demands a table and a reply for an empty review trains everyone to skip it. Resolved with an explicit early exit and by limiting posted replies to threads someone is actually waiting on.

## Acceptance criteria

- [x] The skill triggers on demand and by default after a PR's checks finish, waiting a configurable interval.
- [x] It reports CI pass/fail with a failing-check summary and the command that reads the log.
- [x] It gives every substantive comment a proposed apply/defer/skip with a one-line rationale.
- [x] It never applies a change or merges without explicit human confirmation.
- [x] It distinguishes "reviewed, no findings" from "the review never ran", and refuses to claim the first when the second happened.
- [x] It exits in one line when there is nothing to dispose of.
- [x] It names no review vendor: dispositions depend on the claim, not the author.

## References

- Bitácora board: mlorentedev/dotfiles#411 (see the `issue:` frontmatter field)
- `00_meta/skills/pr-review-triage/SKILL.md` — the skill itself
- `00_meta/patterns/pattern-track-or-fix.md` — the two-exits rule, here applied to someone else's findings
- `00_meta/patterns/pattern-change-lifecycle.md` — the Definition of Done whose Review item this executes (#820)
- `00_meta/patterns/pattern-git-workflow.md` §9 — merge policy
