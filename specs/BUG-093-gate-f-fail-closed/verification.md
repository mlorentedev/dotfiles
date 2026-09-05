---
tags: [spec, verification, templates]
created: "2026-09-05"
---

# Verification - BUG-093-gate-f-fail-closed

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
linux impl fails open on unreadable /proc   mutated_rc=0  not caught locally
restored_rc=0
```

**The second gap is real and is not closed by this change.** Reaching it needs an unreadable
`/proc`, which a test cannot construct without injecting the directory read — a restructuring
of a function this change otherwise only moves. Recorded rather than presented as covered.

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
