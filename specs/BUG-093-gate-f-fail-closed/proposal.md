---
id: "BUG-093-gate-f-fail-closed"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-05"
issue: "mlorentedev/dotfiles#1516"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-093-gate-f-fail-closed

## Why

`dotf worktree sweep` deletes worktrees, and Gate f is the check that asks whether a live
process is still using one. It read `/proc` with no build-tag split and returned `false` — "no
process is inside" — whenever `os.ReadDir("/proc")` failed. On Windows `/proc` never exists, so
the gate answered "nobody is in there" on every single run, and the caller deletes on that
answer. A reaper whose entire premise (#1500) is fail-closed had one gate that failed open, on
a platform this repository actively supports and gates CI on.

## What

Gate f answers `true` — "assume something is in there" — on every path where it cannot
actually observe processes, so the caller refuses instead of deleting. Linux keeps the real
`/proc` walk and now also answers `true` when `/proc` is unreadable. Every other platform
answers `true` unconditionally until a real implementation exists.

`dotf worktree sweep` says so, before the counts:

```
note: no process-liveness check on this platform, so nothing can be reaped.
      Gate f refuses rather than guesses; remove one deliberately with `dotf worktree done`.
```

`SweepReport` gains `ProcessDiscovery bool` so a JSON consumer can tell the two apart too.

## Out of scope

- **A real Windows implementation** (`ToolHelp32Snapshot` + per-process cwd) and a Darwin one
  (`libproc`). Both are the correct eventual fix; neither can be written or verified from
  Linux, and a half-working pseudo-implementation fails the same silent way as the defect.
- **The other fail-open classifications CodeRabbit raised on #1515** — zero-timestamp metadata
  and detached HEAD both reaching `StateReapable`, and `clean.go` inferring descendant safety
  from a directory name. Same family, separate change, dispositioned in #1515's triage.
- **Injecting the `/proc` directory read.** It would close the one mutation gap below, but it
  restructures a function this change is otherwise only moving.

## Risks / open questions

- **`sweep` becomes inert on Windows.** Accepted deliberately: an inert sweep costs disk, a
  wrong one costs uncommitted work. The command now states it rather than printing `reaped 0`,
  which is indistinguishable from a clean machine.
- **The unsupported branch cannot be executed by the Linux CI leg.** Mitigated two ways rather
  than one: a package-level seam makes the *caller's* refusal testable anywhere, and
  `sweep_proc_other_test.go` (`//go:build !linux`) asserts the *platform answer* on the Windows
  leg. Neither alone is sufficient — a seam test passes over a wrong implementation, and the
  platform test cannot see whether the caller honours the answer.
- **`processDiscoverySupported` can drift from reality.** If a Windows implementation lands and
  the constant is not flipped, `sweep` reports a liveness check it never performed. Pinned by a
  test, and called out in the constant's own comment.

## Acceptance criteria

1. `isHostProcessInside` has no single definition shared across platforms; Linux and non-Linux
   are separate build-tagged files.
2. On Linux, every failure that prevents the **scan** answers `true`, not `false`:
   `filepath.Abs` error, `filepath.EvalSymlinks` error, `os.ReadDir("/proc")` error.

   **Amended after round 1, and the amendment is the finding.** This criterion originally read
   *"every failure path answers `true`"*, which the reviewer correctly showed the code does not
   and must not satisfy: `os.Readlink("/proc/<pid>/cwd")` fails with EACCES for every process
   this user does not own, `/proc/1/cwd` included, so answering `true` there would make `sweep`
   permanently inert on Linux. The original wording described a system that cannot exist.
   A per-process failure is therefore **three-valued** — inside, outside, or unreadable —
   and unreadable is counted and reported rather than folded into either answer.

2b. A process that has exited between the directory read and the link read counts as
   **outside**, not unreadable: `/proc` is a snapshot, processes leave during every scan, and
   counting them would make the partial-scan signal fire constantly and mean nothing.

2c. The target path is resolved with `filepath.EvalSymlinks` before comparison, because
   `/proc/<pid>/cwd` is already resolved by the kernel and `filepath.Abs` is not.
3. On any non-Linux platform, `isHostProcessInside` answers `true` for every input.
4. `processDiscoverySupported` is `false` exactly where there is no implementation.
5. `isCandidateForReap` refuses a `StateReapable` worktree when Gate f answers `true`, and
   allows it when Gate f answers `false` — both directions driven by a test seam.
6. `dotf worktree sweep` reports the absence of process discovery before its counts, and
   reports a non-zero uninspectable count when the scan ran but could not see everything.
7. `go build`, `go test ./...`, `golangci-lint run` and `GOOS=windows go vet ./...` are clean.
8. The caller-side fix is mutation-proven: reverting it fails the suite, with the anchor
   asserted before the mutation is applied.
