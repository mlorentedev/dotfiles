---
id: lesson-044-chore-close-spec-lifecycle-pattern-for-features-th
type: lesson
status: active
created: "2026-05-19"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 044: "chore: close spec lifecycle" pattern — for features that shipped piecemeal before archive

**Context:** TERM-001-ghostty-bootstrap had its proposal scaffolded on 2026-05-17 but the implementation shipped piecemeal across PR #38 (tmux truecolor) + commit b00353e (full ghostty bootstrap) + commit 7424731 (config translation) before the spec lifecycle was formally closed. By 2026-05-19 the feature was 100% live on main with bats green, but the spec folder still sat in specs/ (not archive/) with tasks.md as a skeleton.</context>
<problem>SDD-001's archive criterion ("move folder to specs/archive/ on merge") is straightforward when a single PR ships the feature. It's awkward when implementation lands across multiple commits over multiple days — there's no single "merge" event to trigger archival, and tasks.md / verification.md don't get filled because the work was done. The risk: spec folders accumulate in active state indefinitely after the feature is shipped, polluting `ls specs/` and breaking the "active spec = WIP" invariant that the spec-gate CI relies on.</problem>
<parameter name="solution">Apply the "chore: close spec lifecycle" pattern: a small atomic PR that introduces ZERO production code changes and only does the archive housekeeping. Three artefacts in this PR: (1) fill tasks.md retroactively as a map from existing artefacts to the original TDD plan; (2) fill verification.md with the evidence map (commit hashes for each AC + bats output); (3) features.json with the harness-facing contract; (4) git mv specs/<id>/ → specs/archive/<id>/.

Example: PR #66 for TERM-001 — branched as `chore/TERM-001-close-spec-lifecycle`, single commit `chore(spec): close TERM-001-ghostty-bootstrap lifecycle`, 6 files changed (3 archive moves + 3 spec-folder additions), 160 insertions / 98 deletions, zero production code.

The pattern is NOT a workaround for SDD discipline. The proposal WAS filled before implementation (2026-05-17). The pattern is the final step that turns "shipped artefacts" into "archived spec + audit trail". The risk to guard against: scope creep — be strict about ZERO production code in close-out PRs.

The PR title prefix `chore(spec):` makes the intent obvious at review time.</parameter>
<parameter name="tags">["sdd", "spec-lifecycle", "housekeeping", "atomic-prs"]</parameter>
</invoke>
**Problem:** 
**Solution:**
