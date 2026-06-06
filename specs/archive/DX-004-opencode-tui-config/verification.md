---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - DX-004-opencode-tui-config

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (`interleaved` on 4 NaN models) -> commit `9e31ae3` (`ai/opencode/opencode.jsonc`) / test `tests/opencode.bats`
- [x] AC2 (`tui.json` SSOT: theme + `display_thinking`=ctrl+o) -> commit `9e31ae3` (`ai/opencode/tui.json`) / test `tests/opencode.bats`
- [x] AC3 (`setup-linux.sh` plain-copy deploy, no env substitution) -> commit `9e31ae3` (`setup-linux.sh`) / test `tests/opencode.bats`
- [x] AC4 (`setup-windows.ps1` parity, ASCII-only) -> commit `9e31ae3` (`setup-windows.ps1`) / test `tests/setup-windows.bats`
- [x] AC5 (stale reasoning comment corrected to 1.15.13 + `interleaved`) -> commit `9e31ae3` (`ai/opencode/opencode.jsonc`)
- [x] AC6 (empirical, user-run): in the TUI, the NaN reasoning chain becomes visible after pressing `ctrl+o` (or `/thinking`) -> user-confirmed working 2026-06-05.

## Test status

- Test suite: `bats tests/opencode.bats tests/setup-windows.bats` -> green (guard tests for AC1/AC2/AC3 and AC4 parity); grep-only assertions for CI-container portability (no node/jq dependency).
- Manual smoke test: AC6 exercised in the opencode TUI with a NaN reasoning model; `ctrl+o` (bound to `display_thinking`) reveals the previously-invisible reasoning chain. Confirmed by user 2026-06-05.
- No regressions in existing test suite: yes — full bats run remained green; shellcheck clean on `setup-linux.sh`.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- `interleaved` was schema-valid on opencode 1.15.13 but SDK stream emission was UNVERIFIED at implementation time, so AC6 was scoped as user-empirical rather than test-asserted. Empirical run later confirmed the reasoning part is emitted and rendered — schema-presence did predict runtime behavior here, but the verify-before-claim split was the correct call.
- Mapped only `reasoning_content` (not `provider_specific_fields.reasoning`): the `interleaved` enum covers `reasoning_content` / `reasoning_details` but not the latter, so NaN's secondary reasoning field is intentionally out of scope.
- `tui.json` deployed as a PLAIN COPY (no `substitute_env_placeholders`), unlike `opencode.jsonc` which carries a secret placeholder — the TUI config has no secrets, so the simpler copy path applies.
- Guard tests use grep-only assertions (no node/jq) to stay portable inside the CI Docker container.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons.md`? no — the non-obvious decisions are captured in the "Decisions made" section above; nothing recurring.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — `tui.json` plain-copy deploy is already covered by ADR-012 (deploy = copy with drift assertion).
- [x] New pattern candidate for `00_meta/patterns/`? no — single-project, opencode-specific.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/DX-004-opencode-tui-config/` -> `specs/archive/DX-004-opencode-tui-config/`
- [x] Backlog entry in vault `11-tasks.md` ticked with PR link
- [x] Promotions above executed (if any)
