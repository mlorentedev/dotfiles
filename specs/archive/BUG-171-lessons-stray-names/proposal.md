---
id: "BUG-171-lessons-stray-names"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-12"
issue: "mlorentedev/dotfiles#3415"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# BUG-171: a lesson in another naming scheme passes fmt --check

## Why

<!-- from issue #3415: lessons fmt --check passes a lesson whose filename it does not recognise -->

`dotf lessons fmt --check` reads only `lesson-NNN-*.md`. It skips any other markdown file in the lessons directory without a word, so a lesson named in another scheme passes the check and never reaches `dotf lessons index`. The hygiene composite in ci-workflows (CI-005) replaced per-repository scripts, some of which recognised more than one scheme. svqtriana.github.io used date-named lessons until svqtriana.github.io#43 renamed them, so a contributor following the old habit there gets a green check and a lesson nobody can find.

## What

- `fmt --check` fails on any `*.md` file in no `lesson-NNN-<slug>.md` form, both in the lessons directory and in its first-level category directories (where `Plan` reads lessons). It names each file as `unrecognised name: <path>` and prints one fix line.
- `fmt` without `--check` renames nothing. It names each such file as `left alone`.
- The directory's own documents stay accepted: `_format.md` and `README.md`. So does `_index.md`, which the committed-index check reports already, and any file that is not markdown.

## Out of scope

- Renaming the files automatically. The number is a choice (the next free one, or a retired one still cited somewhere), and `renumber` refuses a file with no number.
- Directories deeper than the first level. `Plan` does not read lessons there either.
- A configurable allowlist. Two names cover every checkout measured.

## Risks / open questions

- Blast radius across the fleet. Measured on 2026-11-12 on every local checkout under `~/Projects` with a `docs/lessons`: the only non-lesson markdown files are `_index.md` and `_format.md`. The one exception is a stale svqtriana.github.io checkout, whose date-named files were renamed upstream. Resolved.
- The fleet sees this change only when a release carries it. That is the normal path for a `dotf` check. Resolved.

## Acceptance criteria

- [ ] AC1: In a copy holding `2026-12-01-stray.md`, at the root or in a category directory, `fmt --check` exits non-zero, names that file, and states the expected `lesson-NNN-<slug>.md` form.
- [ ] AC2: `_format.md`, `README.md` (at either level) and non-markdown files are not named, and a repository holding only those passes.
- [ ] AC3: `fmt` without `--check` leaves such a file in place and names it.

## References

- #3415, svqtriana.github.io#43, #2700 (committed index)
- `cli/internal/lessons/lessons.go`, `cli/internal/cmd/lessons.go`
