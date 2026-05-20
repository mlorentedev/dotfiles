---
tags: [spec, tasks, agents, model-selection]
created: "2026-05-19"
---

# Tasks - AI-019-model-tier-policy

## Setup

- [x] Branch: `feat/SDD-003-ci-spec-gate` (bundled by user decision; conscious deviation from atomic-PR ideal — noted in PR body).
- [x] `proposal.md` filled.
- [x] No open questions blocking implementation (Gemini/Copilot empirical TBD accepted).

## Implementation

### Phase 1 — Canonical rule in AGENTS.md

- [ ] Add "## Model Selection (Task-Aware)" section to `AGENTS.md` between "Standing Orders" and "Competence Retention Protocol".
- [ ] Verify: section is ≤45 LOC; references each `ai/<agent>/` overlay by file path; no model name literals (those live in overlays).
- [ ] Manual smoke: re-read AGENTS.md top-to-bottom to confirm flow remains coherent.

### Phase 2 — Per-agent overlays

- [ ] `ai/claude/CLAUDE.md`: append "## Model Tier (per AGENTS.md)" subsection with 3-tier list (claude-opus-4-7 / claude-sonnet-4-6 / claude-haiku-4-5-20251001) + one sentence on `/model` slash command and subagent frontmatter. Verify ≤70 lines.
- [ ] `ai/gemini/GEMINI.md`: same structure. Verify file still ≤70 lines.
- [ ] `ai/copilot/copilot-instructions.md`: same structure with explicit TBD note pointing to AI-017/AI-018.
- [ ] `ai/opencode/opencode.jsonc`: add `_modelTierComment` top-level key with array of 3 lines mapping tier → model ID. Verify file still parses as JSONC (loose JSON), `setup-linux.sh` deploy block still works.

### Phase 3 — Verification

- [ ] `bats tests/agents-md.bats` (if exists) — verify AGENTS.md tests still green.
- [ ] Full bats suite: `bats tests/*.bats` — 645/645 (no regression).
- [ ] `python3 -c 'import json; json.load(open("ai/opencode/opencode.jsonc"))'` — strict JSON parser tolerance check (will fail because of comments; we expect this — opencode reads it as JSONC). Use a JSONC-tolerant validator instead.
- [ ] `wc -l ai/claude/CLAUDE.md ai/gemini/GEMINI.md ai/copilot/copilot-instructions.md` — confirm each ≤70 lines post-edit.
- [ ] Update `verification.md` with evidence + commit hashes.

## Closing

- [ ] Every acceptance criterion ticked.
- [ ] No reformatting drift elsewhere in `AGENTS.md` or overlays.
- [ ] PR body mentions the conscious atomic-PR deviation (SDD-003 + AI-019 + ghostty bundled).
- [ ] After PR merge: tick AI-019 in vault `11-tasks.md` with PR link; move folder to `specs/archive/`.
