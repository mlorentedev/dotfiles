---
id: lesson-115-release-please-can-close-a-multi-pr-issue-from-a-b
type: lesson
status: active
created: "2026-06-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 115: release-please can close a multi-PR issue from a build-only sub-PR's `Refs` — keep the parent issue out of sub-PR footers

**Context**: CLI-018 was split into PR-B0 (§4 coverage port, build-only, #522) and PR-B (the deletion, tracked by #509). #522's footer deliberately said `Refs #509`, *not* `Closes`, because the deletion was not done.

**Problem**: When #522 merged, release-please rolled it into the 0.13.0 release PR (#523), whose generated changelog rendered the reference as `closes #509`. Merging the release PR auto-closed #509 — the work-gate for the still-unstarted deletion vanished while `healthcheck.ps1`/`doctor.ps1` were verifiably still in `main`. Same premature-close class as #488 earlier in the convergence.

**Solution**: Verified the deletion was genuinely undone (both `.ps1` present), reopened #509 with the remaining scope and a note on the cause.

**Rule**: A build-only sub-PR of a multi-PR issue must not reference the parent issue in its footer *at all* — release-please aggregates any issue mention into the release's `closes` list regardless of the `Refs` vs `Closes` keyword. Reference the sub-task or nothing; reserve the issue reference for the final PR that completes it. After any release that swept a multi-PR issue, re-check that issue is still open.
