---
id: lesson-076-archived-spec-issue-complete-verify-high-value-ope
type: lesson
status: active
created: "2026-06-01"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 076: Archived-spec ≠ issue-complete; verify "high-value open" items against git before implementing

**Context:** 2026-06-01 backlog-reconciliation session. Picked the top "high-value open" backlog items to implement (BUG-024, SDD-009, #156).
**Problem:** All three "high-value open" items were already shipped and merged — never ticked in the vault or closed on GitHub (backlog over-reported pending work ~3×). Worse, when reconciling GH issues I closed #193 (HERMES-001) on the archived-spec signal, but the user flagged the agent box still needs bootstrap/config/backups — the issue tracked broader operational scope than its archived spec, so the close was premature.
**Solution:** Before implementing any "open" backlog item, verify against git/PRs/archived-specs first (extends [[pattern-verify-against-source-of-truth]]). Two reliability tiers: archived-spec = deterministic "this spec shipped" (full-id keyed, dodges ticket-number reuse); PR-title match = advisory/brittle. But archived-spec proves the SPEC merged, NOT that the broader ISSUE/epic is operationally complete — an issue (esp. "add agent X to ecosystem") can outscope its spec. So: tick vault + close issues on archived-spec, but only after confirming the issue's scope == the shipped work; when unsure, ask the owner (the #193 reopen). Also: GH issues with no TICKET-NNN prefix escape id-keyed sweeps (#156 slipped) — a content-level sweep catches those. Mechanical enforcement shipped as scripts/check-backlog-merged.sh (SDD-012b, PR #203).
**Tags:** `#backlog` `#reconciliation` `#verify-before-act` `#drift` `#process`
