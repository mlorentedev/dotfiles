---
id: lesson-170-a-health-report-over-a-tree-another-process-is-wri
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 170: A health report over a tree another process is writing is a dirty read, and re-running it is how you find out

**Context**: Running `/insights` and then `/vault-doctor` over the knowledge vault. `vault_health` reported frontmatter errors; the plan was to backfill them.

**Problem**: Two runs minutes apart disagreed — the first found 30 issues (truncated, 25 errors), the second 46 (9 errors), with an entire project's worth of errors having vanished in between. The instinct is to suspect the checker's cache. The cause was a second agent session committing to the same vault: 32 commits in 90 minutes, one of them a 143-file frontmatter backfill that landed at 04:09:20 UTC, precisely between the two reads. Spot-checking the seven files still listed as errors, **four already carried the exact keys the report said they lacked**. Repairing from that snapshot would have produced a diff "fixing" four already-correct files — noise indistinguishable from work, layered on top of another session's live edits.

**Solution**: Verify each flagged item against the file on disk before acting on any aggregate scan, and read a disagreement between two runs as evidence of a *writer*, not of a flaky tool. Then wait for the tree to go quiet — clean working tree plus a no-commit interval — and re-diagnose from scratch instead of repairing from the stale snapshot.

**Rule**: A scan of shared mutable state is a report about a moment that has already passed, and its aggregate counts are the least trustworthy part of it — they are the part carrying no per-item evidence. Before acting on one, take a lock, pin a revision, or re-verify the specific items you intend to touch. Never let the first read decide the work when a cheap second read can tell you whether the ground is moving.
