---
id: lesson-131-bats-silently-drops-test-names-with-non-ascii-char
type: lesson
status: active
created: "2026-06-25"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 131: bats silently drops @test names with non-ASCII chars or duplicates — lint them

**Context**: HARNESS-043 (#607) had a `@test` name with an em-dash; bats 1.13.0 reported "executed 36 instead of 37" and exited 0. A prior lesson noted duplicate `@test` names break parsing. The auto-curation analyzer (CURATOR-001, #135) flagged the recurrence; implementing the proposed lint surfaced 6 more non-ASCII `@test` names already in the suite (em-dash, `<=`), 3 of them silently skipped in opencode.bats (44 declared, only 41 run).

**Problem**: bats translates `@test "<name>"` into a shell function name. Non-ASCII bytes make bats fail to register the function ("unknown test name"); duplicate names make it refuse to parse the whole file. In both cases the SUITE still exits 0 — a green test that never ran. CI's `bats tests/*.bats` was passing while 3 opencode tests were dead.

**Solution**: `scripts/check-bats-names.sh` scans `tests/*.bats` for (a) non-ASCII characters in `@test` names and (b) duplicate names within a file, failing with `file:line`. Wired into the CI `lint` job. Fixed the 6 existing violations (em-dash to `-`, `<=` glyph to ASCII `<=`), recovering the silently-skipped tests (opencode.bats 41 -> 44, all green). **Promoted to check**: `scripts/check-bats-names.sh`.

**Rule**: Keep `@test` names ASCII and unique within a file. A passing bats suite does not prove every test ran — only an executed-vs-declared count or a name lint proves that. If a test name carries an em-dash or a `<=` glyph, it is probably not running.

---
