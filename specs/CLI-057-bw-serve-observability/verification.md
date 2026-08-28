---
tags: [spec, verification]
created: "2026-08-27"
---

# Verification - CLI-057-bw-serve-observability

## Evidence

Run on the Windows work box (go1.26.0 windows/amd64), 2026-08-27, in worktree `dotfiles-wt-secrets-daemon`.

- [x] **AC1** log + pid on Start → `TestBWServeDaemon_Start_WritesLogAndPID` (real child through `Start()`, pid file == child pid, log carries the child's stdout AND stderr lines plus dotf's start marker); zero State keeps the old contract → `TestBWServeDaemon_Start_NoStateDirWritesNothing`.
- [x] **AC2** bounded log → `TestBWServeState_RotateOverCap` (over the cap → `.log.1`, previous `.1` replaced, fresh log is empty), `TestBWServeState_RotateLeavesLogUnderCap`, `TestBWServeState_RotateFallsBackToTruncate` (rename blocked → truncated).
- [x] **AC3** unlock/lock print the trace → `TestSecretsUnlock_Succeeds_PasswordNeverInOutput`, `TestSecretsLock` (assert `pid 4242` + log path); no pid file → `TestSecretsUnlock_NoPIDFileIsSaidNotGuessed` ("pid unknown", never an invented pid).
- [x] **AC4** detached child + redirected stdio on Windows → `TestBWServeDaemon_Start_WritesLogAndPID` run on this box: the helper carries `bwServeDetachAttr()` (= `CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS`), starts, and its lines land in the log (`--- PASS (0.16s)`).
- [x] **AC5** doctor branches by status tag → `TestCheckBWServeDaemon_States` (6 rows: never started → INFO; pid gone → WARN "daemon exited" with the log's last lines; pid gone + empty log → "log is empty"; pid alive → WARN "alive but nothing answers"; locked → INFO; unlocked → OK), `TestCheckBWServeDaemon_UnreadablePIDFileIsAWarn`, `TestCheckBWServeDaemon_NilProcessAliveReadsAsGone`.
- [x] **AC6** liveness → `TestProcessAlive` (running child true → killed+reaped false), `TestProcessAlive_SelfIsAlive`, `TestProcessAlive_NonPositivePIDIsDead`. Windows impl exercised on this box; Unix impl vetted with `GOOS=linux go vet` and runs on the CI Linux leg.

### Mutation checks (revert the fix, watch the guard fail, restore)

- doctor: `case errors.Is(err, os.ErrNotExist)` → `case err == nil || errors.Is(...)` (a valid pid file rendered as the old "no daemon running" Info):
  `--- FAIL: TestCheckBWServeDaemon_States` — 3 subtests (pid gone with lines, pid gone empty log, pid alive). Restored.
- secrets: `if !d.State.enabled()` → `if true` (bare `cmd.Start()`, the pre-#1315 spawn):
  `--- FAIL: TestBWServeDaemon_Start_WritesLogAndPID — pid file after Start: ... bw-serve.pid: The system cannot find the path specified.` Restored.

## Test status

```
go build ./... && go vet ./... && GOOS=windows go vet ./... && GOOS=linux go vet ./... && go test ./...
ok  	github.com/mlorentedev/dotfiles/cli/internal/cmd	18.394s
ok  	github.com/mlorentedev/dotfiles/cli/internal/doctor	20.136s
ok  	github.com/mlorentedev/dotfiles/cli/internal/secrets	4.946s
(all other packages ok; none skipped)
golangci-lint run ./...   → 0 issues.   (2.12.2, the versions.conf pin)
```

- No regressions in the existing suite: yes.
- Live daemon on this box deliberately NOT exercised: the main session owns `bw serve` here (a live experiment was running), so no `dotf secrets unlock|lock` and no doctor run against it. The Windows-specific claim (AC4) is proven by the Go test on this box instead.

## Decisions made during implementation

- **`*os.File` handed to the child, never a pipe.** `os/exec` connects an `*os.File` directly with no copy goroutine; any other writer needs `Wait()` in a parent that exits in milliseconds. Lesson 241.
- **Rotation at start time, one generation, truncate as fallback.** Rotating on every start would discard exactly the lines a fresh death is diagnosed from; rotating over the cap bounds growth to two files; a rename blocked by an open handle truncates rather than growing.
- **Liveness through the `System` seam, stdlib `syscall` only.** `golang.org/x/sys` stays an indirect dependency; the two Win32 constants the syscall package lacks are local, like `detachedProcess`.
- **`Refs #1315`, not `Closes`.** Closing trips the archive gate, and `dotf spec archive` needs an adversarial review that runs `dotf secrets run` against the live daemon this session must not touch. Review + archive are deferred to the owner — a stated Definition-of-Done skip, not a silent one.
- **Info → Warn, never Fail.** The shellout fallback still works; `reportAgentLaunchability` already prices an absent daemon for the wrappers.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`: yes — lesson 242 (in this PR).
- [ ] ADR-worthy decision: no.
- [ ] New pattern candidate for `00_meta/patterns/`: no.

## Archive checklist

- [ ] `dotf spec review CLI-057-bw-serve-observability` PASS (owner: needs an unlocked daemon on a box this cluster does not own)
- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-057-bw-serve-observability/` -> `specs/archive/CLI-057-bw-serve-observability/`
- [ ] Bitácora #1315 closed with the PR link (ADR-018)
