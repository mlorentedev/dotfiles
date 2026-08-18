---
id: lesson-059-incomplete-migration-file-rename-leaves-callers-st
type: lesson
status: active
created: "2026-05-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 059: Incomplete migration: file rename leaves callers stale

**Context:** SDD-007 renamed GEMINI.md → AGY.md across the dotfiles project. Three separate cleanup PRs (#105, #108, #109) all post-merge found different surfaces still referencing the old name or stale assumptions.
**Problem:** When migrating an identity file (X.md → Y.md), it's tempting to think the work is "rename the file in repo + deploy from new path". But the FILE rename touches 3 distinct surfaces that all need updating in lockstep:

1. **The filesystem**: pre-migration installs accumulate the orphan file. Setup must `rm -f old_name` BEFORE copying new_name, idempotently.
2. **Tooling that READS by name**: healthcheck assertions, init-project copies, bats tests — every callsite that grep'd for "X.md" or did `Test-Path X.md` is now broken or asserting stale state.
3. **The CONTENT inside the new file**: the H1, body text, and embedded references may still describe the old tool. The rename ships a hollow shell unless the migration also refreshes prose.

In SDD-007 specifically: PR #102 did the file rename. PR #105 (orphan + healthcheck refs + init-project refs), PR #108 (stale content inside AGY.md), and PR #109 (hc path mismatch) each unwound one of the three surfaces — three separate post-merge fix PRs for what should have been one cohesive migration.</problem>
<parameter name="solution">For any rename/migration PR going forward, the checklist BEFORE merge is:

```bash
# 1. Find every caller of the old name (every surface)
grep -rIn "OLD_NAME" --exclude-dir={node_modules,.git,specs/archive} .

# 2. Confirm every hit is either:
#    - Updated to NEW_NAME, OR
#    - An intentional historical reference (archive, vault lesson), OR
#    - A cleanup statement (rm -f OLD_NAME with explanatory comment)

# 3. Check the NEW file's content, not just its filename
head -20 path/to/NEW_NAME  # Does the H1 / body still describe OLD?

# 4. For each callsite, ask: "what happens on the next setup run for a
#    user whose machine has the pre-migration state?" — if the answer
#    isn't "the orphan gets removed and the new state is reconciled",
#    the migration is incomplete.
```

Better still: add a `tests/` regression guard that bans the old name in the canonical SSOT (e.g. `! grep -qF '# GEMINI.md' "$DOTFILES_DIR/ai/agy/AGY.md"`). That makes "completion" mechanical rather than reviewer-vigilance.

The pattern is "incomplete migration" — same root cause class as "set -u: requires ${1:-} for optional positional parameters" (one-place fix masking missed-callsite bugs). When you change something, all its callers need to come with you.
**Solution:** 
**Tags:** `#migration` `#refactoring` `#pre-merge-discipline` `#regression-guards` `#dotfiles`
