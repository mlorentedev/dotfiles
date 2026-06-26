---
id: "POLISH-006-bats-name-lint"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-25"
issue: "mlorentedev/dotfiles#615"
tags: [spec, proposal]
template_version: "1.0"
---

# POLISH-006-bats-name-lint

## Why

bats translates each `@test "<name>"` into a shell function name. Names with **non-ASCII characters** (em-dash, `<=`) or **duplicated within a file** make bats silently fail to register the test (`unknown test name`) or refuse to parse the whole file — while the suite still exits 0. The result is a "green" test that never ran.

This is a **lesson → enforceable check** promotion (`pattern-lesson-promotion`): the class recurred — duplicate names (`docs/lessons.md` 2026-06-25) + an em-dash that silently skipped a test in #607 — and the auto-curation analyzer dogfood (CURATOR-001, `mlorentedev/knowledge#135`) flagged it, emitting proposal #615. Implementing it surfaced **6 more** non-ASCII `@test` names already in the suite, 3 of them silently skipped in `opencode.bats` (44 declared, only 41 run).

## What

1. `scripts/check-bats-names.sh` — scans `<path>...` (`.bats` files or dirs) for (a) non-ASCII characters in `@test` names and (b) duplicate names within a file; fails `file:line`. Exit 0 clean / 1 violation / 2 usage.
2. Wired into the CI `lint` job (`.github/workflows/ci.yml`).
3. The 6 pre-existing violations fixed (em-dash → `-`, `<=` glyph → ASCII), recovering the silently-skipped tests.

## Out of scope

- Single-quoted `@test '...'` names (the repo convention is double quotes) — add if it ever appears.
- A pre-commit hook (CI lint is the gate for now; pre-commit can follow).
- Asserting executed-vs-declared counts at runtime (the name lint is the cheaper deterministic layer per `pattern-lesson-promotion`).

## Acceptance criteria

- [x] `scripts/check-bats-names.sh` flags non-ASCII `@test` names and duplicates with `file:line`; exits 0/1/2.
- [x] `tests/check-bats-names.bats` covers: usage error, clean pass, non-ASCII (em-dash + `<=`), duplicates, recursive dir scan, and the repo's own `tests/` passing.
- [x] CI `lint` job runs the check.
- [x] All 6 existing violations fixed; `opencode.bats` recovers 41 → 44, all green.
- [x] `shellcheck --severity=error` clean.

## References

- Proposal + provenance: `mlorentedev/dotfiles#615` (auto-curation candidate, accepted).
- Analyzer dogfood: `mlorentedev/knowledge#135` (CURATOR-001 S1).
- Rules: `pattern-lesson-promotion`, `pattern-memory-consolidation`.
