---
id: "BUG-060-crystallize-handoff-order"
type: spec
status: implementing
created: "2026-08-08"
issue: "mlorentedev/dotfiles#850"
tags: [spec, tasks]
template_version: "1.0"
---

# Tasks — BUG-060-crystallize-handoff-order

TDD order. Tests were written to fail against the pre-fix script before the fix was kept.

## 1. Reproduce

- [x] Confirm the defect on a real file: crystallize `web` and observe `# currentDate` landing
      below `## Session Handoff`.
- [x] Locate both bare-append call sites: `.sh` `update_current_date` / `stamp_last_crystallized`,
      `.ps1` `Update-CurrentDate` (L127) / `Set-LastCrystallized` (L154).

## 2. Failing tests first

- [x] `tests/knowledge-crystallize.bats`: handoff stays last on a fresh file.
- [x] `tests/knowledge-crystallize.bats`: idempotent across two runs.
- [x] Verify both FAIL against `8b64995^` (`not ok 14`, `not ok 15`).

## 3. Fix

- [x] `.sh`: add `append_before_handoff`, route both call sites through it.
- [x] `.ps1`: add `Add-SectionBeforeHandoff`, route both call sites through it.
- [x] Verify the two BATS cases now pass.

## 4. Guard the twin

- [x] `tests/knowledge-crystallize-ps1.bats`: helper exists; no bare `Add-Content` of a stamp
      survives; both call sites route through the helper. (Static — `pwsh` absent locally,
      behaviour covered by CI `test-windows`.)

## 5. Repair the damage

- [x] Move `# currentDate` above the handoff block in `web`'s `MEMORY.md`; commit in the vault
      (`bf028001`).

## 6. Close out

- [x] Full suite green (34/34 across both files).
- [ ] PR opened, CI green, merged by a human.
- [ ] `dotf spec archive BUG-060-crystallize-handoff-order`, issue #850 closed.

## Deferred (not this PR — see proposal "Out of scope")

- [ ] Vault-wide assertion that `## Session Handoff` is the last heading in every `MEMORY.md`.
- [ ] The `dotf vault crystallize` port and the caller/doc repoint inventory → #490 (CLI-021).
