---
tags: [spec, tasks]
created: "2026-09-05"
---

# Tasks - BUG-093-gate-f-fail-closed

## Implementation

- [x] Move `isHostProcessInside`, `isNumericPID`, `processCwdInside` out of `sweep.go` into
      `sweep_proc_linux.go` behind `//go:build linux`.
- [x] Make every Linux failure path answer `true` (`filepath.Abs` error, `os.ReadDir` error),
      where it previously answered `false`.
- [x] Add `sweep_proc_other.go` (`//go:build !linux`) answering `true` unconditionally, with
      the reason and the eventual implementation named in the comment.
- [x] Add `processDiscoverySupported`, one value per build-tagged file.
- [x] Route Gate f through a package-level `hostProcessInside` variable so the refusing branch
      is reachable from a test on any platform.
- [x] Surface `SweepReport.ProcessDiscovery` and print the note before the counts.

## Tests

- [x] `sweep_proc_test.go` — both seam directions, plus a platform-consistency assertion.
- [x] `sweep_proc_other_test.go` (`//go:build !linux`) — the platform answer itself, executed
      on the `test (windows-latest)` leg.
- [x] Mutation harness with anchor assertion; caller-side revert is CAUGHT.

## Verification

- [x] `go build ./...`, `go test ./...`, `golangci-lint run` (pinned 2.12.2), `GOOS=windows go
      vet ./...`, `GOOS=darwin go vet ./internal/worktree/` — all clean.
- [x] All six `features.json` verification commands executed, all `rc=0`.

## Not done, deliberately

- [ ] Real process enumeration on Windows / Darwin — out of scope, stated in the proposal and
      in `sweep_proc_other.go`. `sweep` stays inert there until it exists.
- [ ] Injecting the `/proc` directory read to close the second mutation gap.
- [ ] The sibling fail-open classifications from #1515's review (zero-timestamp metadata,
      detached HEAD, `clean.go` name-based allowlist) — same family, dispositioned separately.

## Round 2 (after the round-1 FAIL)

- [x] Blocker: resolve the target with `filepath.EvalSymlinks` before comparing against
      `/proc/<pid>/cwd`, which the kernel has already resolved.
- [x] Major: make a per-process read three-valued (`inside` / `outside` / `unreadable`), count
      the unreadable ones, and report them. A vanished process counts as outside, not unreadable.
- [x] Major: `sweep_proc_linux_test.go` — six tests against the real `/proc` with live child
      processes, including the symlink regression.
- [x] Amend AC2 to state what the code can actually guarantee, and declare the amendment in
      `verification.md` rather than letting it pass silently.
- [x] `SweepReport.UninspectableProcesses` + the partial-scan note in the command.
- [x] **Undeclared at the time, recorded here after round 3 named it:** `dab7b6e` drove the
      `hostProcessInside` seam in five pre-existing `sweep_test.go` tests. Necessary — Gate f
      answering `true` off Linux makes any test that expects a reap fail on the Windows leg — but
      it was a test change made after the review launched, and it moved the reviewed sha.

## Round 3 (after the round-3 FAIL — the surviving-mutation round)

- [x] Reframe `Uninspectable` from partial-scan warning to Gate f **reach**: `Scope of Gate f` in
      the proposal, new CLI wording, three code comments rewritten. Measured first, then designed:
      the uid-filtered rescue does not work (20 of 227 same-uid processes are EACCES).
- [x] Split the vacuous sibling test into `TestGateFMatchesAProcessInASubdirectory` and
      `TestGateFDoesNotMatchASiblingSharingANamePrefix`, each with its own root, no `t.Skip`.
- [x] `TestUninspectableProcessesAreCountedByTheRealScan` — drives the producer, anchored on
      `/proc/1/cwd` being unreadable so it cannot go vacuous on a root-run machine.
- [x] `TestSweepRefusesWhenAProcessArrivesAfterTheGate` — the under-lock re-check, via a seam that
      answers `false` then `true`, with a call-count anchor.
- [x] `TestProcessDiscoveryIsAdvertisedOnLinux` replaces the unfalsifiable Linux pin.
- [x] `TestGateFRefusesWhenTheTargetCannotBeResolved` — the `EvalSymlinks` branch, which needed
      only a nonexistent path.
- [x] Delete `isCandidateForReap` (no production caller); move its seam tests onto `gateF`.
- [x] `strconv.Itoa` for the hand-rolled `itoa`; document `cwdVerdict`'s permissive zero value.
- [x] Amend AC5/AC5b/AC6/AC7 and correct three false claims in `verification.md`.
- [x] Reproduce the mount-namespace fail-open independently; declare it in *Out of scope* and file
      it as **#1523** rather than shipping a fix with no CI-runnable test.
- [x] Execute the non-Linux path locally with the build-tag-flip proxy — `go vet` cannot see it.
