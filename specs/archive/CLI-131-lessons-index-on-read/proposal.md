---
id: "CLI-131-lessons-index-on-read"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-10-16"
issue: "mlorentedev/dotfiles#2554"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "owner 2026-10-17 'ok' to the advisor plan: one override open at a time, the CLI-128 one is archived (12 active, limit 10) (12 active, limit 10, 2026-10-16)"
---

# CLI-131-lessons-index-on-read

> **Naming**: file lives at `<repo>/specs/CLI-131-lessons-index-on-read/proposal.md`. `CLI-131-lessons-index-on-read` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #2554: CLI-131: render docs/lessons/_index.md on read instead of committing it -->

`docs/lessons/_index.md` is a committed table generated from the lesson files, so every pair of PRs that each add a lesson conflicts on it, and under a merge queue a conflict ejects the PR. Its readers are people and agents looking for a lesson; nothing consumes it at run time (owner, 2026-10-15). The files are already the source of truth, so the table can be rendered when someone reads it. Part of #2546 (queue throughput), narrows #2538.

## What

- `dotf lessons index [--dir D]` prints the index to stdout, rendered from the lesson files (one table per directory that holds lessons). It never writes a file.
- `dotf lessons fmt` no longer reads, writes or requires an `_index.md`. It still normalises each lesson, rejects a number used twice, and rejects a wikilink that names no lesson (over the lessons only: there is no hand-written index prose left to check).
- `dotf lessons fmt --check` reports a committed `_index.md` anywhere under the lessons directory as drift, with the fix ("remove it; render with `dotf lessons index`"), so a repo that still commits one is told on its next release instead of keeping a table that silently goes stale.
- `dotf init` stops scaffolding `docs/lessons/_index.md`.
- The session-start staleness warning reads the newest `lesson-*.md` instead of the index's mtime, so deleting the index does not silence it.
- dotfiles deletes its `docs/lessons/_index.md`.

## Out of scope

- Any change to `.github/workflows/repo-hygiene.yml` or `.pre-commit-config.yaml` commands: both run `lessons fmt --check`, which stays the right command. Only their comments mention a generated index, and those are reworded.
- Deleting `_index.md` in the other repos that `dotf init` scaffolded it into (11 on this machine: none runs the hook or CI, so none goes red; each removes its own on its next sync).
- A cache or a gitignored copy of the rendered index: "render on read" means no file at all.

## Risks / open questions

- **A committed index in another repo goes stale silently** if `--check` ignores it. Decided: `--check` flags it (loud). Blast radius measured 2026-10-17: 11 repos carry an `_index.md` and only dotfiles runs `lessons fmt` (hook and CI), so nothing else turns red when this ships.
- **Title seeding.** `fmt` seeded a missing frontmatter `title` from the index table. Measured: 0 of the lessons lack a title, so the seed path is deleted rather than kept for nothing.
- **Session-start staleness.** It stat'd `_index.md`; with no index and no fallback it returned "" and the warning died silently. The fix reads the newest lesson file, and a test pins it.
- Shell portability: nothing here is shell; Go in `cli/`.

## Acceptance criteria

- [ ] `dotf lessons index` prints a table with one row per lesson, sorted by number, and writes no file; a directory of category subfolders gets one section each.
- [ ] `dotf lessons fmt` and `--check` pass on a tree with lessons and no `_index.md`, and never create one.
- [ ] `dotf lessons fmt --check` exits 1 naming a committed `_index.md` and the fix; `fmt` does not delete it.
- [ ] A number used twice and a dangling wikilink are still errors without an index.
- [ ] `dotf init` scaffolds no `docs/lessons/_index.md`.
- [ ] The staleness warning fires on an old newest lesson and is silent on a fresh one, with no index present.
- [ ] `docs/lessons/_index.md` is gone from dotfiles and `dotf lessons fmt --check --dir docs/lessons` exits 0 there.

## References

- Bitácora board: mlorentedev/dotfiles#2554 (child of #2546, narrows #2538)
- Related: lesson 374 (a generated file git merged is no longer generated), lesson 280 (the same conflict, measured)
- Existing code: `cli/internal/lessons`, `cli/internal/cmd/lessons.go`, `cli/internal/initrepo/scaffold.go`, `cli/internal/mem/session_start.go`
