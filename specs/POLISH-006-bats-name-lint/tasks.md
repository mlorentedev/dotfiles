---
tags: [spec, tasks]
created: "2026-06-25"
---

# Tasks - POLISH-006-bats-name-lint

> Lesson → check promotion. TDD.

- [x] bats `tests/check-bats-names.bats` (RED): usage, clean pass, non-ASCII (em-dash + `<=`), duplicates, recursive scan, repo `tests/` passes
- [x] `scripts/check-bats-names.sh` (GREEN): collect `.bats` from args/dirs; (a) non-ASCII in `@test` names, (b) duplicate names; `file:line`; exit 0/1/2
- [x] Fix the 6 existing violations (em-dash → `-`, `<=` glyph → ASCII) in hermes-setup / iac-deploy / opencode / vault-maintenance-weekly
- [x] Wire into CI `lint` job (`ci.yml`)
- [x] Lesson promoted to `docs/lessons.md`
- [x] `shellcheck --severity=error` clean; `opencode.bats` 41 → 44 green
- [ ] PR opened referencing #615
