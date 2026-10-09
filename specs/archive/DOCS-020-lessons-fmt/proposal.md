---
id: "DOCS-020-lessons-fmt"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-10-06"
issue: "mlorentedev/dotfiles#2038"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "Every parallel PR that adds a lesson conflicts on the hand-kept index (three rebases on 2026-10-06); the owner asked for the homogenisation and the Go port now (18 active, limit 10, 2026-10-06)"
---

# DOCS-020: One lesson format, a generated index, and `dotf lessons fmt`

## Why

<!-- from issue #2038: DOCS-020: one lesson format, an index generated from it, and dotf lessons fmt to keep both -->

The 339 lessons, measured on 2026-10-06, were written in three shapes:

- 50 have no frontmatter and give the date inline (`**Date:**`).
- The vault-era lessons have an `id` and a `created` date but no `title`.
- The newest have a `title` and a `date`.

Their H1 also comes in three forms (`# Lesson 026: …`, `# Lesson 215 — …`, `# 213 — …`).

The index, `docs/lessons/_index.md`, was a hand-kept second copy of each title and date. 77 of its titles differed from their H1, and two of its dates were a day off. Two parallel sessions adding a lesson each conflicted on its last line: #2025 was rebased three times for it on one evening.

`scripts/check-lessons.sh` only checks membership, and it is shell. New logic belongs in Go (ADR-020), and ADR-042 asks for these checks as one `dotf` binary rather than a script per repository.

The owner asked for the formats to be homogenised and the check ported to Go (2026-10-06).

## What

- **`internal/lessons`.** It normalises each `lesson-NNN-<slug>.md` to one form and leaves the body as written:
  - frontmatter starts with `id` (the file name), `type: lesson`, `status`, `title` and `created`, and keeps every other key;
  - the H1 equals the title, with no `Lesson NNN` prefix;
  - the date lives only in `created`, so there is no `date:` key and no inline `**Date:**`.
  
  It regenerates each directory's `_index.md` table between generated markers and keeps the index's prose. It refuses a number used twice anywhere, lessons with no index, and lessons in a category with no root index.
- **`dotf lessons fmt [--check] [--dir]`.** It is idempotent; `--check` writes nothing and exits 1, naming every file that would change.
- **One-time migration.** A file with no `title` takes the existing index row's title, so the 77 curated titles survive; after that the file is the source. This PR runs `fmt` on all 339 lessons and the index.
- **CI.** `repo-hygiene.yml` runs `fmt --check` with `go run` from the checkout, so it never waits for a release.
- **`dotf init`.** Its `lessons-index.md` template is the generated shape, and a test holds it as a fixed point of `RenderIndex`.
- **Vault.** The lesson-writing skills (architecture-session, crystallize, handoff, spec) say "run `dotf lessons fmt`" instead of "register in `_index.md`". The vault's `lesson.md` and `lessons-index.md` templates take the canonical form. Vault commit `02192bb9`; the skill records are refreshed here.

## Out of scope

- **Pre-commit and the removal of `check-lessons.sh`.** Pre-commit runs the installed `dotf` (0.64), which lacks `lessons`. Swapping the hook before a release carries the command would break every commit (#1814 class). PR 2, after the release and the `DOTF_VERSION` bump, swaps the hook and deletes the script with `tests/check-lessons.bats` and the overlapping `tests/guard-lesson-numbers-unique.bats`.
- **Keeping `_index.md` out of PR diffs** (a bot regenerating it on `main`). That is an owner decision for its own ticket. Until then, a conflict on the index is resolved with one command.
- **The body sections of old lessons.** They are content, and rewriting them automatically would be invasive.
- **Other repositories' copies of `check-lessons.sh`** (#1573): they can call `dotf lessons fmt --check` once it is released.

## Risks / open questions

- **339 files change in one PR.** The diff is mechanical and a second `fmt` changes nothing. The review samples one lesson of each shape and the index, rather than reading every file.
- **Changed ids.** Old `id` values such as `lesson-213` become the file stem. Nothing resolves lessons by `id` (links use file names), and the vault does not index repo lessons.
- **A parallel PR adding a lesson in the old shape.** `fmt --check` fails it in CI and names the remedy.

## Acceptance criteria

- [ ] AC1: every lesson shape found in the repository normalises to the canonical form, and normalising the result changes nothing.
- [ ] AC2: the index table is generated from the files between markers, sorted by number, and the index's prose is kept.
- [ ] AC3: a number used twice, lessons without an index, or a category without the root index is an error.
- [ ] AC4: `dotf lessons fmt --check` exits 1 naming every unformatted file, and exits 0 on the formatted repository.
- [ ] AC5: the `dotf init` index template is exactly what `fmt` renders for no lessons.
- [ ] AC6: CI runs `fmt --check` on every PR.
- [ ] AC7 (PR 2): pre-commit runs `dotf lessons fmt --check`, and `check-lessons.sh` and its two bats files are deleted. Every assertion they made is in Go first, including the one only `guard-lesson-numbers-unique.bats` made: a lesson wikilink that names no lesson is an error.

## References

- #2038, #1573 (fleet hygiene guards), #1286, #1343; ADR-020, ADR-042.
