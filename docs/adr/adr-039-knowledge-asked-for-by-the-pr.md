---
id: "ADR-039-knowledge-asked-for-by-the-pr"
type: adr
status: accepted
owner: manu
date: "2026-09-25"
supersedes: []
extends: []
issue: mlorentedev/dotfiles#387
tags: [architecture, decision, harness, lessons, adr, ci, sdd]
created: "2026-09-25"
---

# ADR-039: The PR that produces knowledge names where it went, and CI checks it

## Context

Definition of Done §2 asks that what was learned be written where it belongs, in the session it was learned. It is text, read at the end of the work, and at the end of the work it loses to the next task.

Measured on 2026-09-25 (vault note `10_projects/dotfiles/research/2026-09-25-knowledge-capture-routing.md`):

- dotfiles lessons per commit fell from 0.42 in W36 to 0.18 in W39;
- there was no ADR from W36 on;
- one session merged 13 PRs and wrote 10 tickets with measured root causes, and added no lesson and no ADR.

The knowledge was written, in ticket and PR bodies. Those are the surfaces a mechanism asks for: the spec gate asks for a spec, review-attestation for a review, a ticket for a root cause. Lessons and ADRs were the only ones nothing asked for.

The spec archive had the same gap. `dotf spec archive` printed that promotion "must be done separately", and accepted a "no", or no answer.

## Decision

1. **Every PR body carries a `## Knowledge` section** with three lines: Lesson, ADR and Runbook.
   - Each line is one or more paths, or `none: <reason>`.
   - A path counts when the PR adds, changes or renames it, it sits under `docs/lessons/`, `docs/adr/` or `docs/runbooks/`, and it is not `_index.md`.
   - The rules live in `scripts/check-knowledge-gate.sh`. CI runs it as `knowledge-gate`, reading the body live, so editing the body re-runs it.
2. **No title, branch or label relaxes it.**
   - A rule keyed on a name is escaped by renaming, so there is no stricter rule for `fix:` PRs either: every line already needs a path or a reason.
   - Dependency-bot PRs are skipped exactly as `spec-gate` skips them, by an exact bot login plus the `dependencies` label.
   - Release PRs pass on content, because the release-please footer carries the section.
3. **The archive enforces the promotion step.** A promotion line in `verification.md` is answered `yes: <path>`, with a path that exists, or `no: <reason>`, and `dotf spec archive` refuses anything else.
4. **Policy values change with data**, as ADR-037 §3 sets them. A reason only has to be non-empty until a vacuous one is seen in practice.

## Consequences

- A PR with nothing to capture costs one sentence per line. A PR with something to capture is asked while the author still holds it.
- The author judges their own work. The gate makes the decision visible, and the review and the human merge judge it.
- The check is honoured by the human merge, as `spec-gate` is, until GUARD-017 (#1451) gives `forge/branch-protection.json` an apply. Then it is declared required.
- Other repositories get the gate by GUARD-016's delivery path (#1627), not by copying the workflow.
- Lesson 301 records the failure this answers. The spec is `HARNESS-024-knowledge-capture-gate`.
