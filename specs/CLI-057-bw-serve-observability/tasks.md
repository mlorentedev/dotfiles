---
tags: [spec, tasks]
created: "2026-08-27"
---

# Tasks - CLI-057-bw-serve-observability

> TDD order. One PR off main, `Refs #1315` (archive + adversarial review deferred to the owner — see proposal "Risks").

## Setup

- [x] Branch `feat/bw-serve-observability` off `origin/main` (84448bb) in worktree `dotfiles-wt-secrets-daemon`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC1] [AC2] Failing tests in `cli/internal/secrets/bwserve_state_test.go`: `BWServeState` paths derive from the deploy dir; `openLog` appends, rotates over the cap, replaces a stale `.log.1`; `ReadPID` / `LastLogLines` round-trip
- [x] [AC1] [AC2] Implement `bwserve_state.go`: `NewBWServeState(dotfilesDir)`, `LogPath`/`PIDPath`, `openLog` (rotate-or-truncate), `WritePID`, `ReadPID`, `LastLogLines`
- [x] [P] [AC6] Failing test `TestProcessAlive` (helper process: alive → kill+wait → dead) in `procalive_test.go`
- [x] [AC6] Implement `procalive_unix.go` (`kill -0`, EPERM = alive) and `procalive_windows.go` (`OpenProcess` + `GetExitCodeProcess`, access-denied = alive)
- [x] [AC1] [AC4] Failing test `TestBWServeDaemon_Start_WritesLogAndPID`: `newCmd` seam returns a real helper process carrying `bwServeDetachAttr()`; assert pid file = child pid, log carries the child's stdout+stderr lines; kill + reap
- [x] [AC1] Wire `Start()`: `State` field, open log, redirect stdio to the `*os.File` (no copy goroutine, so `dotf` can exit), write pid after `cmd.Start()`
- [x] [P] [AC5] Failing table test in `checks_bw_serve_test.go`: absent/no pid, absent/pid dead (last lines shown), absent/pid alive, locked, unlocked — by status tag
- [x] [AC5] `System.ProcessAlive` seam (prod `secrets.ProcessAlive`, `newSys` default), `checkBWServeDaemon(sys, cfg, rep)` reads the pid file under `cfg.DotfilesDir`
- [x] [AC5] Mutation check: remove the pid-file branch, watch the table fail, restore (3 subtests red, see `verification.md`)
- [x] [P] [AC3] Failing tests: `unlock` and `lock` output names pid + log path; no pid file → "pid unknown"
- [x] [AC3] `bwDaemonAddr` sets `State` from `env.DotfilesDir(env.Home())`; unlock/lock print `Trace()`

## Closing

- [x] Every acceptance criterion is covered by at least one test
- [x] `features.json` entries carry non-vacuous verification commands
- [x] `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./... && golangci-lint run ./...` clean
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder; `## Review triage` posted; `dotf pr triage-queue` exit 0
