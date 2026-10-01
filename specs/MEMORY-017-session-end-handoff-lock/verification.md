---
tags: [spec, verification, templates]
created: "2026-09-30"
---

# Verification - MEMORY-017-session-end-handoff-lock

## Evidence

- [x] AC1 -> `mem.LockHandoffMemory` / `LockHandoffMemoryWithin`, used by both
  `handoffWrite.run` and `SessionEnd`; `TestTwoPathsToOneMemoryShareTheLock`.
- [x] AC2 -> `TestSessionEndWaitsForConcurrentHandoffWrite`.
- [x] AC3 -> `TestConcurrentWritesDoNotLoseAThread`, focused package tests, vet,
  and Unix cross-compilation.

## Test status

- RED: `go test ./internal/mem -run '^TestSessionEndWaitsForConcurrentHandoffWrite$'`
  failed to compile because no shared lock API existed.
- First GREEN exposed a second shipped Windows race:
  `TestConcurrentWritesDoNotLoseAThread` failed in round 4 with `Access is
  denied` because `EvalSymlinks` opened the target file before acquiring the
  lock. Canonicalizing only the parent directory removed that pre-lock handle.
- Stress:
  - `go test ./internal/mem -run '^TestSessionEndWaitsForConcurrentHandoffWrite$' -count=10` -> pass.
  - `go test ./internal/cmd -run '^Test(ConcurrentWritesDoNotLoseAThread|TwoPathsToOneMemoryShareTheLock)$' -count=10` -> pass.
- Full focused suites: `go test ./internal/mem ./internal/cmd ./internal/filelock -count=1` -> pass.
- `go vet ./internal/mem ./internal/cmd ./internal/filelock` -> pass.
- Linux cross-compilation of `./internal/cmd` -> pass.
- `git diff --check` -> pass.

## Decisions made during implementation

- Preserve HARNESS-088's exact canonical key and runtime/cache lock directory;
  this PR changes ownership of the helper, not lock semantics.
- Canonicalize the parent directory rather than the file itself: Windows can
  deny the writer's rename while another goroutine briefly resolves the target
  file before taking the lock.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? no: HARNESS-088 verification and
  MEMORY-017 preserve the Windows reader-vs-rename detail.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: this closes a
  race inside the existing handoff architecture.
- [x] New pattern candidate for `00_meta/patterns/`? no: this is a
  project-specific extension of the existing shared-file lock.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/MEMORY-017-session-end-handoff-lock/` -> `specs/archive/MEMORY-017-session-end-handoff-lock/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
