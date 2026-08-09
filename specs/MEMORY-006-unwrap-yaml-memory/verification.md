---
tags: [spec, verification, templates]
created: "2026-08-09"
---

# Verification - MEMORY-006-unwrap-yaml-memory

## Evidence

- [x] **AC1** (`dotf doctor` reports, `--fix` migrates, re-run is a no-op) →
      `TestCheckMemoryShape` cases *"wrapped without --fix → FAIL…"*,
      *"--fix migrates and leaves plain files alone"*, *"--fix is idempotent"*.
- [x] **AC2** (de-indent = block indent + uniform residual, both derived) →
      `TestUnwrap` cases *"uniform indent"* and *"first line shallower than the
      rest"*; no literal width appears in `memshape.go`.
- [x] **AC3** (validated against ground truth) → see below.
- [x] **AC4** (migrated files crystallize) → proven by effect: column-0 headers
      go 0 → N on every file (table below), which is what the markers match on.
- [x] **AC5** (frontmatter preserved) → measured on all 17: parses, retains
      `id`/`status`/`tags`/`type`.
- [x] **AC6** (hard breaks and blank-but-indented lines survive) → measured on
      all 17 (table below) and pinned by
      *"the markdown hard break survives the migration"*.
- [x] **AC7** (#862's guard still refuses what this has not migrated) — the two
      are complementary: `IsWrapped` and the shell's `is_yaml_block_scalar` use
      the same structural test, so a file this declines is still refused.

## Ground-truth characterization

`TestUnwrapAgainstVaultGroundTruth` runs the transform over every `MEMORY.md`
the 2026-05-26 wrap touched, taking the **wrapped** side from vault commit
`1c216229` and the **original** from its parent — 23 real before/after pairs
authored by neither this code nor its author.

```console
$ go test ./internal/memshape/ -run GroundTruth -v
ground-truth pairs verified: 23 (skipped 0)
ok      github.com/mlorentedev/dotfiles/cli/internal/memshape    0.116s
```

The corpus deliberately stays **out of this repository**: dotfiles is public and
the vault is private, so committing those files would publish infrastructure
notes and session history. The test skips cleanly where the vault is absent
(including CI) and fails loudly if it verifies zero pairs, so a skip can never be
mistaken for a pass. The shapes it covers are reproduced by the synthetic
fixtures in `memshape_test.go`, which are safe to commit and are what CI runs.

**Two findings came out of this test, not out of review.**

1. **The de-indent assertion had to become a prefix match**, because the May wrap
   was *lossy*: it dropped each file's trailing `# currentDate` section. The
   strongest true statement is that de-indenting recovers the surviving content
   byte-for-byte in the original's own formatting.
2. **One file lost far more than that** — `python-sensor-sdk-platform` went 205
   lines → 36. Filed separately as **#865 (MEMORY-007)**; it is a content
   restore, independent of this shape migration, and fully recoverable from
   `1c216229^`.

## Mutation probe

The guard was verified to fire rather than merely exist. Forcing `residual = 0`
— the "YAML-correct" de-indent that stops at the block indent — turns red:

| Case | with mutation |
|---|---|
| `first line shallower than the rest` | **FAIL** |
| `markdown hard breaks survive` | **FAIL** |
| `uniform indent` | PASS (correct — residual is 0 there) |
| `nested content keeps its relative indent` | PASS (correct) |
| ground truth, all 23 pairs | **FAIL** |

The discrimination matters: CI has no vault, so the synthetic cases had to catch
it on their own, and they do.

## Dry run against the real corpus

Run into a sandbox copy; **nothing real was written** (`git status` in the vault:
0 modified, checked after every run).

```console
$ dotf doctor            # real HOME, read-only
[Auto-memory file shape]
  [FAIL] 17 MEMORY.md files hold their body inside a YAML block scalar …
```

| Project | hard-breaks | col-0 headers | frontmatter |
|---|---|---|---|
| hive | 4 → 4 | 0 → 5 | ok |
| kubelab | 5 → 5 | 0 → 12 | ok |
| pollex | 5 → 5 | 0 → 15 | ok |
| resume | 4 → 4 | 0 → 3 | ok |
| youtube-toolkit | 3 → 3 | 0 → 2 | ok |
| yt-metrics-cli | 3 → 3 | 0 → 2 | ok |
| the other 11 | 0 → 0 | 0 → 2…10 | ok |

Every file gains exactly one line (the closing `---` plus a blank, replacing the
`content: |` opener). Hard breaks are preserved exactly — the property no YAML
emitter could hold. Column-0 headers go from **zero to N in every file**, which
is the whole objective: those are what crystallize matches on.

**The dry-run harness needed its own fix, and that is worth recording.** The
first version used `cp -a`, which preserves symlinks — and every
`projects/*/memory` is a symlink into the vault, so the "sandbox" would have
written straight through to the real files. It only did not because an unrelated
`head` closed the pipe and killed the script early. Fixed with `cp -aL` plus an
assertion that no symlink survives into the sandbox. A dry run that can touch
production is not a dry run, and luck is not isolation.

## Live migration — run and verified

Run against the real corpus after the dry run was reviewed and approved.

```console
$ dotf doctor --fix
[Auto-memory file shape]
  [FIX ] migrated to plain markdown: … (×16 distinct files)

$ dotf doctor
[Auto-memory file shape]
  (1 checks, all ok)
```

**AC4 proven by effect, not inspection.** Crystallize was run against a migrated
project and now succeeds where it previously refused:

```console
$ ./scripts/knowledge-crystallize.sh ~/Projects/pollex
[INFO]    Added currentDate section (2026-08-08)
[INFO]    Updated Last Crystallized to 2026-08-08
[SUCCESS] MEMORY.md line count: 108 / 150
```

And the HARNESS-029 invariant survives the stamp — the handoff block is still
last:

```console
$ grep -n '^## Last Crystallized:\|^# currentDate\|^## Session Handoff' …/pollex/…/MEMORY.md
99:## Last Crystallized: 2026-08-08
101:# currentDate
103:## Session Handoff        <- still the final section
```

That is the whole chain: the migration restores column-0 markers, crystallize
finds them, and `append_before_handoff` places the stamp correctly because it can
finally see the block it must stay above.

### A defect the live run found that the fixtures could not

The first live run emitted `left unchanged, shape not recognised` for
`yt-metrics-cli` — a file that the vault showed as successfully migrated.

Cause: **two project keys alias one vault file.** `-home-manu-Projects-youtube-toolkit`
and `-home-manu-Projects-yt-metrics-cli` both symlink to
`10_projects/yt-metrics-cli/memory` — a rename whose old key was never removed.
The scan listed the file twice, migrated it through the first alias, then re-read
it through the second as already-plain and reported that as a shape failure.

Nothing was corrupted — the message was wrong, not the write. But a check whose
diagnosis misdescribes reality is the defect class this whole ticket is about, so
it was fixed in scope: entries are deduplicated by `filepath.EvalSymlinks`, and
`ErrNotWrapped` during the fix loop is now reported as "already plain markdown"
rather than as an unrecognised shape. Pinned by
*"two project keys aliasing one file are handled once"*.

No mutation probe was needed for that case: the production run **is** the
evidence that the test's assertion fails without the fix.

## Test status

```console
$ go test ./... | grep -v "no test files"
ok  …/cli/internal/doctor    0.032s
ok  …/cli/internal/memshape  0.134s
(13 packages, all ok)

$ go vet ./internal/memshape/ ./internal/doctor/
CLEAN
```

## Decisions made during implementation

- **A doctor check, not a one-shot script.** Running it once is the migration;
  leaving it installed is the guard. The May incident shipped only the rule and
  never swept the stock, which is exactly the half this collapses into one
  artifact.
- **Refuse what is not understood.** A block opener followed by a column-0 key
  means the block ends early — a shape never observed. It is left byte-identical
  with a WARN, never guessed at.
- **Atomic writes.** Temp file in the same directory then rename, so a crash
  cannot leave a truncated `MEMORY.md`.
- **Detection is keyed on the shape, not the key name.** `content` is what this
  machine happens to use; the regex matches any block-scalar key at column 0.

## Promotion candidates

- [x] Lesson for `docs/lessons.md`? **yes** — shipped: *a guard stops new
      violations; it does not clean the stock*.
- [ ] ADR-worthy? **no**.
- [ ] New pattern? **candidate** — "a bulk edit over N files needs a per-file
      size assertion" (from #865). Deferred to that ticket.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/MEMORY-006-unwrap-yaml-memory/`
- [ ] Bitácora ticket closed with PR link (ADR-018)

> **Not archived by this PR yet.** The migration has been proven in a sandbox but
> not run against live memory — that write is Manu's call, per this spec's own
> acceptance. The PR carries `Refs #864`.
