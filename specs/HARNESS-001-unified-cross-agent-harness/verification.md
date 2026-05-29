---
tags: [spec, verification, harness-001, epic]
created: "2026-05-28"
---

# Verification - HARNESS-001-unified-cross-agent-harness

> **Epic in `draft`.** No implementation merged yet; evidence is filled in as each PR in the sequence lands. This file is the running ledger that gates archive.

## Evidence

Map every epic acceptance criterion to the PR that proves it.

- [ ] AC1 (manifest + schema parse, fail on malformed) -> PR-1 `<hash>` / test `<name>`
- [ ] AC2 (Linux/Windows compiler parity) -> PR-2 `<hash>` / test `<name>`
- [ ] AC3 (source-marker + committed artifacts) -> PR-1 `<hash>` / test `<name>`
- [ ] AC4 (CI drift guard fails on hand-edit) -> PR-1 `<hash>` / test `<name>`
- [ ] AC5 (no-attribution override block generated from enforce:true) -> PR-1 `<hash>` / test `<name>`
- [ ] AC6 (SDD-008 skills produced by engine) -> PR-4 `<hash>` / test `<name>`
- [ ] AC7 (line caps ≤80 / ≤50 asserted) -> PR-1 + PR-3 `<hash>` / test `<name>`
- [ ] AC8 (ADR-013 recorded; child index present) -> this PR `<hash>`

## Test status

- Test suite: `<command> -> <output / coverage %>` (per PR; do NOT run `bats tests/*.bats` headless — session-start-config.bats #14 hangs on real Obsidian, see #167; run targeted files)
- Manual smoke test: `<what was exercised>`
- No regressions in existing test suite: `<yes / no>`

## Decisions made during implementation

- Engine core lives in the umbrella spec; SDD-008 stays skill-specific and becomes the first consumer (decision 2026-05-28).
- New ADR goes to repo `docs/adr/adr-013` (KPM-001: dotfiles' own ADRs live in the repo, not the vault), superseding the repo copies of adr-001/008.
- Manifest format lean: JSON (`jq` + `ConvertFrom-Json`, no new dep). To be confirmed at PR-1.

## Promotion candidates

- [ ] Lesson for `90-lessons.md`? yes — "tracer-bullet a deploy pipeline on its smallest enforced payload to de-risk cross-OS compile parity before scaling to N artifacts" (capture when PR-1 lands).
- [ ] ADR-worthy decision? yes — ADR-013 (authored as part of this epic, not deferred).
- [ ] New pattern candidate for `00_meta/patterns/`? Maybe — "manifest-driven agent-artifact deploy (generate-and-commit)" if it recurs beyond dotfiles. Defer until a 2nd project needs it.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-001-unified-cross-agent-harness/` -> `specs/archive/HARNESS-001-unified-cross-agent-harness/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
