---
id: "MEMORY-006-unwrap-yaml-memory"
type: spec
status: implementing
created: "2026-08-09"
issue: "mlorentedev/dotfiles#864"
tags: [spec, tasks]
template_version: "1.0"
---

# Tasks — MEMORY-006-unwrap-yaml-memory

TDD order. The oracle is real data, so the characterization corpus is captured
before the transform exists.

## 1. Characterize before transforming

- [x] Locate the wrap event and confirm it is a single bulk edit, not an ongoing
      process — vault commit `1c216229`, `2026-05-26 21:17:41`, nothing since.
- [x] Census the shapes: block indent per file, and whether the body indent is
      uniform. Result: **16 of 17 keys** open at 4 and continue at 6; only `hive`
      is uniform.
- [x] Build the ground-truth harness over `1c216229` / `1c216229^` — the
      **historical corpus of 23 pairs**, asserted exactly so a regression in
      detection cannot hide behind a single surviving pass.
- [x] Keep the corpus out of the repo. dotfiles is public, the vault is private:
      the test reads the vault, skips where absent, and never commits content.

## 2. The transform (`cli/internal/memshape`)

- [x] `IsWrapped`: structural detection — `---` first line plus a block-scalar
      opener **inside the frontmatter**. Bounded at the terminator so a migrated
      file whose body legitimately contains `note: |` is not re-flagged forever.
- [x] `Unwrap`: de-indent by block indent **+ uniform residual**, both derived.
      A single line at the block indent forces the residual to 0.
- [x] Refuse what is not understood (column-0 key after the opener; empty block).
- [x] Table-driven tests per branch, including the residual-0 regression and
      trailing-blank-line fidelity.
- [x] Mutation-verify: forcing `residual = 0` must turn the relevant synthetic
      cases **and** the ground-truth pairs red.

## 3. The doctor check (`cli/internal/doctor`)

- [x] `checkMemoryShape`: verify always, repair under `--fix`, idempotent.
- [x] Deduplicate by `EvalSymlinks` — two project keys can alias one vault dir.
- [x] Atomic write: temp file in the same directory, `Sync`, then rename.
- [x] Register in the non-quick sweep.
- [x] Tests, including the alias case and "verify-only writes nothing".

## 4. Migration

- [x] Dry run into a **dereferenced** sandbox copy, with an assertion that no
      symlink survives into it.
- [x] Review the per-file diff before any live write.
- [x] Run `dotf doctor --fix` — 16 distinct files.
- [x] Verify by effect: crystallize succeeds, and HARNESS-029 still holds.
- [x] Commit the migrated files to the vault (`9f73dfc4`).

## 5. Close out

- [x] Lesson in `docs/lessons.md`.
- [x] File what was found along the way rather than mentioning it — **#865**
      (the wrap also truncated content).
- [ ] PR #866 merged, CI green.
- [ ] `dotf spec archive MEMORY-006-unwrap-yaml-memory`, #864 closed.

## Counts, stated once so they stay consistent

Three different numbers are correct for three different questions:

| Number | What it counts |
|---|---|
| **17** | project keys under `~/.claude/projects/` holding a wrapped `MEMORY.md` |
| **16** | distinct files those keys resolve to — `youtube-toolkit` and `yt-metrics-cli` alias one vault directory after a rename |
| **23** | pairs in the **historical** corpus at `1c216229`, which includes entries since archived or renamed, and files unwrapped by hand later (dotfiles' own) |

The migration target is **16 files / 17 keys**. The test corpus is **23 pairs**.
