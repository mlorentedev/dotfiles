---
tags: [spec, tasks, copilot, docs-drift]
created: "2026-05-19"
---

# Tasks - SDD-005-github-copilot-instructions-sync

## Setup

- [x] Branch: `feat/SDD-005-github-copilot-instructions-sync` (off main).
- [x] Vault entry exists in `11-tasks.md`.

## Implementation (TDD)

### Phase 1 — Parity test (red)

- [ ] Create `tests/docs-drift.bats` with parity case: strip blockquote lines (`^> `) from both files, compare with `cmp -s`. Expect FAIL pre-fix.

### Phase 2 — Sync content (green)

- [ ] Edit `.github/copilot-instructions.md`:
  - Re-align fallback line wording to include "from the current repo"
  - Append "## Model Tier (per AGENTS.md "Model Selection")" section with the same TBD wording as `ai/copilot/copilot-instructions.md`
- [ ] Re-run parity test → expect PASS.

### Phase 3 — Full regression

- [ ] `bats tests/*.bats` → must remain green (target 645 + new cases).
- [ ] Manual sanity: `diff` the two files, confirm only the pointer-banner pointer-reference line (markdown link form vs plain) differs.

## Closing

- [ ] Every AC ticked.
- [ ] `verification.md` filled.
- [ ] PR opened referencing this spec folder.
- [ ] Post-merge: tick SDD-005 in vault `11-tasks.md` and archive spec folder.
