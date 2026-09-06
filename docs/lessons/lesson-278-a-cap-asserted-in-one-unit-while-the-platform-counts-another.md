---
id: lesson-278
type: lesson
status: active
created: "2026-09-05"
owner: manu
tags: [lesson, harness, doctrine, measurement, units, guard]
---

# 278 — A cap asserted in one unit while the payload overflowed in another

## What happened

`.gemini/GEMINI.md` measured 11974 characters and 12047 bytes against a 12000 cap. The
pipeline test read `wc -m`, so it passed. The platform documents the cap in characters, but
nothing here had verified that it counts that way, and its documented behaviour on overflow is
to drop the tail silently. The 73-byte gap was 33 em-dashes, four section signs, an accented
vowel and an ellipsis.

The same unit confusion had already produced a wrong claim the same evening (#1241): persona
prose was said to cost zero because the roster renders skill ids only. It costs zero
characters and two bytes per em-dash. The per-id budget was tracking the measure that was not
growing.

## The lesson

**A guard that measures in a unit the consumer may not use is a guard on the wrong quantity.
When you cannot establish which unit the platform counts, remove the question instead of
guessing: make the two units track each other.** The fix normalises typographic punctuation to
ASCII in the capped payload, so bytes and characters agree however the platform counts, and the
test asserts both units so they cannot separate again.

Normalise only what does not alter the lexicon. Dashes, curly quotes and the ellipsis mean the
same folded. Accents and section signs stay: folding them would change a word or a reference.

## Applied

- `scripts/compile-harness.sh`: `deploy_doctrine` folds typographic punctuation to ASCII in a
  capped payload, using hex escapes because a literal curly quote in shell source fails
  ShellCheck SC1112.
- `tests/skills-pipeline.bats`: asserts the cap in bytes and in characters. Goes red on the
  pre-fix main with `is 12047 BYTES, at or over its 12000 cap (chars: 11974)`.
- Recorded in `specs/HARNESS-111`; fixed in #1513.
