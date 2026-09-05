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
