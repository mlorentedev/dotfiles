---
tags: [spec, verification, engine-001, harness-001]
created: "2026-05-13"
---

# Verification - ENGINE-001-deploy-engine-core

> Branch `feat/engine-001-deploy-core`. Machine-readable contract: `features.json` (7 entries, one per AC). Evidence below is reproducible from the test names; the harness sets `passing` after capturing exit 0 (pass-state gating per [[pattern-feature-list-as-primitive]]).

## Evidence

Every acceptance criterion maps to an executable verification in `features.json` and a test/observation here.

- [x] **AC1** (`--check` 0 on fresh tree, ≠0 after hand-edit) -> `bats compile-harness.bats -f 'AC1'` (2 tests green) + observed: `--check` on the real repo prints `[check] OK -> AGENTS.md` / `[check] OK -> ai/claude/CLAUDE.md`.
- [x] **AC2** (idempotent `--refresh`; BEGIN/END markers carry source + sha256) -> `bats … -f 'AC2'` (2 tests green) + observed: re-running `--refresh` against the real vault leaves `git diff` byte-identical; markers carry `sha256:82589eb0b0204879`.
- [x] **AC3** (record committed; `--check` renders offline, no vault) -> `bats … -f 'AC3'` (2 tests green) + observed: `--check` passes with `VAULT_PATH` at an empty dir.
- [x] **AC4** (100-line cap breach fails) -> `bats … -f 'AC4'` (1 test green); deployed `ai/claude/CLAUDE.md` = 90 lines (≤100).
- [x] **AC5** (missing END marker fails loud, no silent append) -> `bats … -f 'AC5'` (1 test green).
- [x] **AC6** (healthcheck wires the offline drift check) -> `bats … -f 'AC6'` (1 test green); `scripts/healthcheck.sh` calls `compile-harness.sh --check`.
- [x] **AC7** (override text in deployed AGENTS.md + CLAUDE.md) -> `grep -q 'No AI attribution' AGENTS.md && grep -q … ai/claude/CLAUDE.md` (exit 0).

## Test status

- Test suite: `bats tests/compile-harness.bats` -> `1..12` all `ok` (12/12). Run targeted, NOT `tests/*.bats` (hangs per #167).
- `features.json`: all 7 verification commands exit 0 (run manually); `jq` schema-valid (unique ids, all `state: verifying`, no sham verifications).
- Static: `shellcheck scripts/compile-harness.sh` clean; `bash -n` + `zsh -n` pass.
- Manual smoke: `--refresh` against the real vault renders §6/§7/§8 of `pattern-git-workflow.md` into both targets; `--check` on the real repo reports no drift; second `--refresh` is a no-op (idempotent).
- No regressions: `bats tests/opencode.bats -f 'pointer to AGENTS.md'` -> 2/2 green after the 80→100 cap bump.

## Decisions made during implementation

- **Single managed region per target**, not per-rule markers (the original proposal sketch). All enforced rules render concatenated under one `<!-- BEGIN/END HARNESS GENERATED -->` block, stable-sorted by manifest order — collapses FM3 (nondeterministic order churn) to a single deterministic block. `proposal.md` was rewritten to this 3-rule/2-target design.
- **Marker carries `sha256:<16>` of the rendered region + SSOT pointer + "do not edit between markers"** so drift is visible in the diff and the source is self-documenting.
- **`english-only` (§8 Language Policy) added to the vault SSOT** `pattern-git-workflow.md` in this change — it was enforced in `AGENTS.md` with no SSOT, which is exactly the #156 regression class.
- **Cap bumped 80→100** on `ai/claude/CLAUDE.md` to fit the generated block; `opencode.bats` updated with the justification inline.
- **Drift guard lives in `healthcheck.sh` (local), not CI** — per the 2026-05-28 decision (ADR-013), `--check` is offline-by-design so the guard runs on any machine without the vault.

## Promotion candidates

> Per the knowledge-placement model (KPM-001), project-bound knowledge stays in this repo's `docs/`; only cross-project brain goes to the vault. **Flagged for user confirmation — promotion is a knowledge-system call.**

- [ ] **Lesson?** Recommend **no** as a standalone vault lesson. The reusable insight (SSOT→generated-block→offline drift guard closing the #156 class) is already captured by ADR-013 + this spec. Optional one-liner to `docs/lessons.md` if desired.
- [ ] **ADR-worthy?** **Already exists** — `docs/adr/adr-013-agent-artifact-deploy-engine.md` (engine) + `adr-012` (copy-with-drift substrate). No new ADR.
- [ ] **New `00_meta/patterns/` pattern?** **Defer.** The "compile canonical block from SSOT + sha-markered region + offline drift check" mechanism is reusable, but per the pattern's own >1-project rule its consumers (PR-2 Windows, PR-3 other agents) are still inside HARNESS-001. Revisit when a second independent project adopts the engine.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/ENGINE-001-deploy-engine-core/` -> `specs/archive/ENGINE-001-deploy-engine-core/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
- [ ] PR merged; worktree `dotfiles-engine` removed (`git worktree remove`) + branch deleted
