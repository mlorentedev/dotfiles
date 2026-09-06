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

Gate f answers `true` — "assume something is in there" — on every path where the SCAN cannot
run, so the caller refuses instead of deleting.

**That is narrower than "cannot observe processes", and the wording matters.** Gate f compares
paths as strings, so an occupant reaching the worktree through a bind mount in another mount
namespace is observed and read as *outside*. It is not protected, and this change does not claim
to protect it — see *Out of scope* and #1523. Linux keeps the real
`/proc` walk and now also answers `true` when `/proc` is unreadable. Every other platform
answers `true` unconditionally until a real implementation exists.

`dotf worktree sweep` says so, before the counts:

```
note: no process-liveness check on this platform, so nothing can be reaped.
      Gate f refuses rather than guesses; remove one deliberately with `dotf worktree done`.
```

`SweepReport` gains `ProcessDiscovery bool` so a JSON consumer can tell the two apart too.

## Scope of Gate f — what it can see, and what it never will

**Gate f is the second layer, not the first.** A worktree only reaches `gateF` after it is already
`StateReapable`, which requires on-disk metadata with `reap_ok` and an expired lease — a signal the
tool writes and owns. Gate f is the best-effort check for "somebody is sitting in it anyway", and
best-effort is a bound, not a hedge:

- Reading `/proc/<pid>/cwd` is gated by `ptrace_may_access`. An ordinary user's scan can never read
  root's processes, another user's, or a same-uid process that set `PR_SET_DUMPABLE=0`
  (`systemd --user`, `ssh-agent`, browser sandboxes).
- Measured on one ordinary desktop: **571 processes, 364 out of reach** — 344 owned by other users,
  and 20 owned by the caller and still denied.

So `UninspectableProcesses` is **non-zero on every Linux machine, by construction**. Round 3 caught
the previous framing calling it a partial-scan *warning*: a warning that fires on every run is one
readers learn to skip, and it was the answer to round 1's Major, so the mitigation signalled
nothing. It is now reported as coverage — a statement of the gate's reach — and no decision is
taken on it, because none can be: an unreadable process stays unreadable however many there are.

## Out of scope

- **A real Windows implementation** (`ToolHelp32Snapshot` + per-process cwd) and a Darwin one
  (`libproc`). Both are the correct eventual fix; neither can be written or verified from
  Linux, and a half-working pseudo-implementation fails the same silent way as the defect.
- **The other fail-open classifications CodeRabbit raised on #1515** — zero-timestamp metadata
  and detached HEAD both reaching `StateReapable`, and `clean.go` inferring descendant safety
  from a directory name. Same family, separate change, dispositioned in #1515's triage.
- **Injecting the `/proc` directory read.** It would close the one mutation gap below, but it
  restructures a function this change is otherwise only moving.

- **The mount-namespace fail-open, declared rather than fixed.** `/proc/<pid>/cwd` is resolved in
  the *process's own* namespace, so a container bind-mounting the worktree reports its own view and
  the string comparison misses it. Reproduced here, not cited:

  ```
  docker run -d -u 1000:1000 -v /tmp/wtmount-repro/repo-wt-feat:/wt-mount-target \
             -w /wt-mount-target busybox sleep 60
  host readlink /proc/2658284/cwd  -> /wt-mount-target
  worktree path on host            -> /tmp/wtmount-repro/repo-wt-feat
  dev/ino of /proc/<pid>/cwd       -> 50/992409
  dev/ino of the worktree path     -> 50/992409     <- the same directory
  string comparison                -> False         <- the gate says "outside"
  ```

  The same weakness produces the deleted-then-recreated case round 3 probed, and the same fix
  retires both: identity by `dev`+`ino` (`os.SameFile`) instead of by string, walking parents for
  the subdirectory case.

  **Not fixed here because it cannot be given a test that fails in CI, and this spec's own lesson
  is that a fix without one is worse than a declared gap.** The reproduction needs a mount
  namespace: `unshare -Urm` is refused on the development machine
  (`kernel.apparmor_restrict_unprivileged_userns = 1`, Ubuntu's default since 24.04), and whether
  a GitHub `ubuntu-latest` runner permits one is not measurable from here. A test that skips on the
  runner asserts nothing while reporting green — the exact shape of the vacuous test round 3 found
  in this same file. **Tracked as #1523**, whose first task is that measurement rather than a patch.

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
  the constant is not flipped, `sweep` reports a liveness check it never performed. Pinned in
  **both** directions, which round 3 showed it previously was not: the Linux side by
  `TestProcessDiscoveryIsAdvertisedOnLinux` and the non-Linux side by
  `TestUnsupportedPlatformDoesNotAdvertiseDiscovery`. The earlier Linux test skipped when the
  constant was false and then asserted a package-level `var` was non-nil, which it cannot be — so
  flipping the constant left the suite green.

- **A green Linux suite is not evidence about Windows, and `go vet` is not either.** Round 2 shipped
  a red `test (windows-latest)` while every local check passed: `GOOS=windows go vet` compiles the
  non-Linux file and never runs it. Any test that expects a reap must drive the `hostProcessInside`
  seam, because Gate f answers "occupied" unconditionally off Linux. AC7 now names the Windows
  **test** leg, and the build-tag-flip proxy in `verification.md` executes that path locally.

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
5. `gateF` — the function production calls — refuses a `StateReapable` worktree when Gate f
   answers `true` and allows it when Gate f answers `false`, both directions driven by a test seam.

   **Amended after round 3.** This named `isCandidateForReap`, a bool wrapper no production caller
   used: the refusal was pinned on a function that could not prevent a deletion. The wrapper is
   deleted.

5b. The **re-check inside `reapSingleWorktree`** runs **last**, immediately before the removal.
   Amended after round 4: it previously ran before `isDirty`/`isMerged`, which shell out to git,
   so that latency sat inside the window between "nobody is in there" and the deletion. Pinned by
   `TestGateFIsTheLastCheckBeforeRemoval`. The re-check — the one that runs under the lock immediately
   before removal — refuses when a process arrives after the candidate scan. This needs a seam that
   answers `false` then `true`; a constant answer cannot reach it, because a constant `true` is
   refused by the first call and a constant `false` never exercises the second.
6. `dotf worktree sweep` reports the absence of process discovery before its counts, and states
   Gate f's **reach** when the scan ran without full visibility — as coverage, not as a warning
   (see *Scope of Gate f*).
7. `go build`, `go test ./...`, `golangci-lint run` and `GOOS=windows go vet ./...` are clean, and
   the **`test (windows-latest)` leg is green**. The vet is a compile check and cannot see a
   runtime failure on that platform; round 2 proved that by shipping one. Locally this is
   established with the build-tag-flip proxy, which runs the non-Linux implementation on Linux.
8. Every mutation round 3 reported as surviving now fails the suite, with the anchor asserted
   before each mutation is applied.
