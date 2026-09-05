---
tags: [spec, verification, templates]
created: "2026-09-05"
---

# Verification - BUG-093-gate-f-fail-closed

## Round 1 review: FAIL — disposition

`agy/gemini-3.1-pro-high`, random draw, not the implementer. Verdict **FAIL** against
`2d9f4114`, on 1 Blocker and 2 Majors. **All three were real and all three are applied.**

| # | Finding | Disposition |
|---|---|---|
| Blocker | `filepath.Abs` does not resolve symlinks, `/proc/<pid>/cwd` does — a symlinked worktree path never matches, so the gate fails open with a process inside | **Applied.** `filepath.EvalSymlinks` on the target before comparison; `TestGateFSeesThroughASymlinkedWorktreePath` is the regression, mutation-proven |
| Major | `processCwdInside` returns false on `os.Readlink` error, contradicting AC2 | **Applied, and AC2 amended** — see below |
| Major | `sweep_proc_linux.go` has zero test coverage; only the seam is exercised | **Applied.** `sweep_proc_linux_test.go`, 6 tests against real `/proc` with live child processes |

### The AC2 amendment, declared

**AC2 was changed after a review, which is the move a spec gate exists to catch, so it is
recorded here rather than left to be noticed.** The change was made under a **FAIL**, not a
PASS, and it makes the criterion weaker in letter and truer in substance:

AC2 read *"every failure path answers `true`, not `false`"*. The reviewer showed the code does
not satisfy it and **must not**: `os.Readlink("/proc/<pid>/cwd")` returns EACCES for every
process this user does not own — `/proc/1/cwd` included — so answering `true` there would make
`sweep` refuse on every Linux machine, permanently. The original wording described a system
that cannot exist.

The amendment splits the two failure classes that the one sentence had conflated: a failure of
the **scan** (Abs, EvalSymlinks, ReadDir) still answers `true`, while a failure to read **one
process** is now three-valued, and `unreadable` is counted and reported rather than folded into
either answer. The round-2 reviewer reads the amended criterion and this note together.

## Evidence

Every command below was executed in this session; the `features.json` entries carry the
verbatim verification commands and their outcomes.

- [x] **AC1** build-tag split — `sweep_proc_linux.go` and `sweep_proc_other.go` both exist and
      `sweep.go` carries no definition of `isHostProcessInside`.
- [x] **AC2** Linux fails closed — no `return false` follows the `os.ReadDir("/proc")` error
      branch.
- [x] **AC3** non-Linux answers `true` — `GOOS=windows go vet` and `GOOS=darwin go vet` clean;
      the runtime assertion is `TestUnsupportedPlatformAnswersTrueForEveryPath`, which executes
      on the `test (windows-latest)` leg and nowhere else.
- [x] **AC4** `processDiscoverySupported` false exactly where unimplemented —
      `TestUnsupportedPlatformDoesNotAdvertiseDiscovery` (Windows leg) and
      `TestProcessDiscoveryIsReportedForThisPlatform` (Linux).
- [x] **AC5** caller honours both answers —
      `TestGateFRefusesWhenProcessDiscoveryCannotAnswer` and
      `TestGateFAllowsAReapableWorktreeWhenNothingIsInside`, driven through the
      `hostProcessInside` seam. The second exists so the first cannot pass vacuously by the gate
      refusing everything.
- [x] **AC6** the inertness is reported before the counts, in `newWorktreeSweepCmd`.
- [x] **AC7** `go build ./...`, `go test ./...`, `GOOS=windows go vet ./...`,
      `golangci-lint run` (2.12.2, the `versions.conf` pin) — all `rc=0`, `0 issues`.
- [x] **AC8** mutation-proven — see below.

## Test status

- Go suite: `go test ./...` → all packages pass, `rc=0`.
- Gate f subset: `go test ./internal/worktree/ -run 'GateF|ProcessDiscovery' -v` → 3 PASS.
- Lint: `golangci-lint run ./...` → `0 issues`, on the pinned 2.12.2.
- Cross-compile: `GOOS=windows go vet ./...` and `GOOS=darwin go vet ./internal/worktree/`
  clean — this is what compiles `sweep_proc_other.go` and its test file at all.
- No regressions: the pre-existing `worktree` tests are untouched and still pass.

### Mutation results

Anchors asserted before applying, so a mutation that failed to land could not be reported as
"the tests caught nothing":

```
caller ignores Gate f                      mutated_rc=1  CAUGHT
EvalSymlinks removed (the round-1 Blocker) mutated_rc=1  CAUGHT
    --- FAIL: TestGateFSeesThroughASymlinkedWorktreePath
linux impl fails open on unreadable /proc  mutated_rc=0  not caught locally
restored_rc=0
```

**The third gap is real and is not closed by this change.** Reaching it needs an unreadable
`/proc` *directory*, which a test cannot construct without injecting the directory read — a
restructuring of a function this change otherwise only moves. Recorded rather than presented as
covered. Note it is a different thing from an unreadable `/proc/<pid>/cwd`, which round 1
raised and which is now covered by `TestVanishedProcessCountsAsOutsideNotUnreadable` and the
uninspectable count.

### One defect committed while fixing this one, and what caught it

Changing Gate f's return type broke `sweep_proc_other_test.go`, and **`go test ./...` on Linux
stayed green** — it never compiles a `//go:build !linux` file. That is precisely the defect
this spec exists to fix, reproduced by the author mid-fix. `GOOS=darwin go vet` caught it,
which is why AC7 names the cross-compile vet rather than treating the Linux suite as sufficient.

## Decisions made during implementation

- **Two test files, not one, and neither is redundant.** The Linux seam test proves the
  *caller* refuses; `sweep_proc_other_test.go` proves the *platform answer* that feeds it. A
  seam test passes over a wrong implementation, and a platform test cannot see whether the
  caller honours the answer. The original defect survived an adversarial review precisely
  because only the code-reading check was available.
- **`sweep` is deliberately inert on Windows** rather than approximated. A half-working
  pseudo-implementation fails the same silent way as the bug being fixed — the same reasoning
  that kept the pipe path on Windows in SEC-002.
- **The inertness is announced before the counts, not after.** `reaped 0` from a refusing gate
  and `reaped 0` from a clean machine are the same string; a reader who takes the first for the
  second concludes the tool ran.
- **Reported severity was on the wrong platform.** CodeRabbit filed this as a macOS exposure.
  This repo does not run macOS and does run Windows, where `test-windows` is a required check.
  The finding was right and its blast radius was understated.

## Promotion candidates

- **A guard for the `features.json` state vocabulary.** Measured across `specs/archive/` during
  this work: seven different terminal spellings in use — `verified` (42), `passing` (39),
  `done` (15), `verifying` (15), `passed` (12), `implemented` (6), against 523 `pending`.
  Nothing validates the field, so each session invented one. Same shape as the closed `Outcome`
  vocabulary #1435 introduced for the gate ledger. Not filed as a ticket yet.
