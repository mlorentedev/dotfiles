---
tags: [spec, verification]
created: "2026-06-25"
---

# Verification - POLISH-006-bats-name-lint

## Evidence

- [x] Lint flags violations → `bats tests/check-bats-names.bats` 7/7 (non-ASCII em-dash + `<=`, duplicates, recursive scan, usage, clean pass).
- [x] Real-codebase find → `./scripts/check-bats-names.sh tests/` flagged 6 non-ASCII `@test` names (hermes-setup:125, iac-deploy:60, opencode:157/230/240, vault-maintenance-weekly:119).
- [x] Silent-skip confirmed + recovered → `opencode.bats` ran 41/44 before (em-dash + `<=` "unknown test name"); after the fix, **44/44, 0 failures** (recovered tests 26/37/38 pass).
- [x] CI wired → `.github/workflows/ci.yml` `lint` job runs the check.
- [x] `shellcheck --severity=error scripts/check-bats-names.sh` clean; `bash -n` OK.

## Decisions made during implementation

- **ASCII-only `@test` names is the rule** — a passing bats suite does NOT prove every test ran; the lint is the cheap proof. (Runtime executed-vs-declared assertion deferred — name lint is cheaper per `pattern-lesson-promotion`.)
- **Fix only `@test` name lines** — non-ASCII in test bodies/comments is harmless; the fix touched only the quoted names (em-dash → `-`, `<=` glyph → ASCII).

## Promotion candidates

- [x] Lesson for `docs/lessons.md`? **done in this PR** — "bats silently drops @test names with non-ASCII chars or duplicates".
- [ ] ADR-worthy? no.
- [ ] New pattern? no — instance of `pattern-lesson-promotion`.

## Note (self-improvement loop)

This is the first end-to-end pass of the auto-curation loop: analyzer (CURATOR-001 dogfood) detected the recurrence → proposal #615 (human gate, accepted) → check shipped here. Demonstrates CURATOR slices S1, S3, and S5 manually.
