---
id: lesson-058-batch-scaffold-n-specs-in-one-pr-from-a-research-w
type: lesson
status: active
created: "2026-05-25"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 058: Batch-scaffold N specs in one PR from a research worktree, defer implementation

**Context:** Did a research worktree comparing 3 reference dotfiles repos (fmontes / holman / mathiasbynens) against this repo. Research surfaced 6 actionable ideas with clear ROI tiering. Two paths forward: (a) open 6 separate PRs, one per spec, paid out over weeks; (b) batch-scaffold all 6 in one PR, defer implementation to per-spec branches.

**Problem:** If each spec gets its own scaffolding PR, the cross-spec dependencies discovered during research (IDEAS-003 enables clean IDEAS-001/002 integration; IDEAS-005 + IDEAS-004 interlock on fresh-machine setup; IDEAS-006 has an abandon gate) get rediscovered or lost across review cycles. Also: 6 small scaffolding PRs each pay the spec-gate / CI / review overhead, but produce zero shippable code. The research doc itself (`research/dotfiles-survey.md`) is the durable index — splitting it across 6 PRs scatters it.

**Solution:** Bundle the research doc + all N spec folders into ONE PR (PR #101, 1,629 LOC, 19 files, 0 code changes). Outcomes: (1) spec-gate workflow PASSED with NO `skip-sdd` label because the new `specs/<id>/` folders ARE the active specs satisfying the gate — confirmed empirically on PR #101; (2) cross-spec dependencies + ordering recommendations live in the PR body as the canonical handoff; (3) implementation work is fully isolated per-spec on `feat/IDEAS-NNN-*` branches with full reviewer attention; (4) the research worktree is disposable after merge. Each spec includes BLOCKER-classified risks, testable AC, features.json skeleton — same shape as SDD-004 so reviewers parse them identically.

**Why:** research outputs go stale fast; batch-capturing them as specs locks in the analytical work before the context window or session memory loses it. **How to apply:** any future research session (compare-X-against-Y, audit-N-instances-of-Z, survey-the-field) → bundle the research doc + every surfaced spec into one scaffold PR before opening implementation branches. Use Tier-1/2/3 + LOC estimate + cross-spec dependencies in the PR body to drive the implementation order.

**Tags:** `#sdd` `#specs` `#research-pattern` `#pr-strategy` `#dotfiles-survey`
