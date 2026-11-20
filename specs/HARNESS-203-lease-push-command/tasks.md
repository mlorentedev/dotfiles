---
tags: [spec, tasks, templates]
created: "2026-11-19"
id: "HARNESS-203-lease-push-command"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-203-lease-push-command

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers:** `[P]` = no dependency on another unchecked task. `[AC<n>]` = serves acceptance criterion n of `proposal.md`.

## Setup

- [x] Branch created from main: `feat/3612-lease-push-command` in lane `feat-lane-primary` (base `origin/main` 31360ae3)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (all four are accepted risks, none blocking)

## Implementation

Layout: logic in a new package `cli/internal/pushlease` (`pushlease.go`), the cobra command in `cli/internal/cmd/pr_push_lease.go`, registered in `cli/internal/cmd/pr.go`. Every test runs real git against a temporary clone of a local bare remote; nothing is stubbed. The fixture sets `core.hooksPath` in the clone to an empty temporary directory, so this machine's global hooks never decide a test.

Test commands run from `cli/`. Expected outputs are the go test verdict lines.

- [x] [P] [AC1] Fixture and first failing test in `cli/internal/pushlease/pushlease_test.go`: `TestMain` calls `gittest.Quiet()`; `newFixture(t)` makes a bare remote whose `HEAD` is `main`, a clone with `main` and a pushed `feat/x`, and the local `core.hooksPath`. `TestPush_MovesTheRemoteAfterARebase` amends `feat/x`, calls `Push` with `Expect` = the remote sha read before, and asserts the remote ref equals local `HEAD` and the output line is `feat/x <old>..<new>`. Run `go test ./internal/pushlease -run TestPush_ -count=1`. Expected: FAIL to compile (`undefined: Push`).
- [x] [AC1] Minimal `pushlease.go`: `Request{Dir, Remote, Expect}`, `Plan{Branch, Remote, Expect, Args}`, `Prepare(Request) (Plan, error)` reading the checked-out branch, `Push(Plan, out, errOut io.Writer) error` running `git -C <dir> push --force-with-lease=refs/heads/<b>:<sha> <remote> HEAD:refs/heads/<b>` through `exec.Command` (no shell), then printing the line. Same command. Expected: PASS.
- [x] [AC1] `TestPush_StaleExpectLeavesTheRemote`, two rows: (a) someone else's commit lands on the remote `feat/x` after the read; (b) the remote already equals local `HEAD` and `--expect` names another sha (git may report "up to date" before it consults the lease). Each: an error, nothing printed, the remote ref unchanged. Recorded 2026-11-20: row (b) failed first, git printed "Everything up-to-date" and exited 0 with the wrong lease; the fix either way is a pre-check: `git ls-remote <remote> refs/heads/<b>` must equal `--expect`, else a `*Refusal` ("stale lease"); the lease still covers the race after that read.
- [x] [P] [AC2] `TestPrepare_Refuses` in the same file, a table, each row asserting a `*Refusal` with its stable `Code` (not prose, per `cli/AGENTS.md`) and every remote ref unchanged after an unpushed commit: detached HEAD; on `main`; on `master`; on the remote default with `refs/remotes/origin/HEAD` set to `develop`; the same with it unset so `ls-remote --symref` must find it; `--expect` empty, 7 hex, `origin/feat/x`, 39 hex; `--remote` not configured, and a path. Recorded 2026-11-20: all 12 rows failed first, none refused; the path row pushed successfully.
- [x] [AC2] Implement the refusals in `Prepare`, local checks before any network call, in this order: branch (`git symbolic-ref --quiet HEAD`, exit 1 = detached), `main`/`master` backstop (any case), `--expect` shape (`^[0-9a-f]{40}$` or 64, lowercased first), remote resolution (`branch.<b>.pushRemote`, `remote.pushDefault`, `branch.<b>.remote`, `origin`; no leading `-`; `git remote get-url` must succeed), one `ls-remote --symref <r> HEAD refs/heads/<b>` (failure = `remote-unreadable`), default branch by union of what the remote reports and what `refs/remotes/<r>/HEAD` recorded (neither = `default-unresolved`), stale lease (the branch sha equals `--expect`). Rows added with the union: the remote default changed since `origin/HEAD` was recorded; neither source known; the remote unreadable; `--remote --mirror`. Every git child gets `gitenv.WithoutRepository(os.Environ())` and `-C <dir>`. Same command. Result: PASS, 16 rows.
- [x] [AC2] `TestPrepare_RemoteResolutionOrder`: `--remote` beats `pushRemote` beats `pushDefault` beats `branch.<b>.remote` beats `origin`. Result: PASS, 5 rows.
- [x] [P] [AC3] `TestPush_PrePushHookRuns`: the fixture's hooks directory gets a `pre-push` that writes a marker file and exits 1; `Push` returns an error, the marker exists, the remote ref is unchanged. Run `go test ./internal/pushlease -run TestPush_PrePush -count=1`. Expected: PASS (proves no `--no-verify`); a mutation adding `--no-verify` to the argv must turn it red, recorded in `verification.md`. Result 2026-11-20: PASS; the `--no-verify` mutant FAILS it (the push succeeded past the hook).
- [x] [AC1] [AC2] Failing test `cli/internal/cmd/pr_push_lease_test.go`: `dotf pr push-lease --expect <sha>` in a fixture clone exits 0 and prints the line; on `main` it exits 1 with the reason on stderr; without `--expect` it exits 1. Run `go test ./internal/cmd -run TestPrPushLease -count=1`. Expected: FAIL (`unknown command "push-lease"`).
- [x] [AC1] [AC2] `cli/internal/cmd/pr_push_lease.go`: `newPrPushLeaseCmd()` with `--expect` (required) and `--remote`, registered in `pr.go`; `Long` names what it refuses and that it never derives the sha. Same command. Expected: PASS.
- [x] [P] [AC4] `cli/internal/claude/push_lease_rules_test.go`: reads `ai/claude/settings.json`; asserts `Bash(git push *--force*)` is in deny, and that no deny, allow or ask rule matches `dotf pr push-lease --expect <sha>` under an emulated matcher named `leaseRuleMatches` (a distinct name: #3608's branch adds `bashRuleMatches` to the same package). Run `go test ./internal/claude -run TestPushLeaseRules -count=1`. Expected: PASS against today's template (it pins what must not change).
- [x] [AC4] `AGENTS.md`, outside the generated block, under "Change Management & Engineering": one line naming `dotf pr push-lease --expect <sha>` as the way an agent force-updates its own branch after a rebase, and that a raw forced push stays denied.
- [x] [AC4] `docs/adr/adr-066-agents-force-update-their-branch-through-dotf-pr-push-lease.md`: decision, the refusals, and an alternatives section (narrowed globs plus a parsing hook, #3608, six FAIL rounds and the measured 28 to 40 rows; a raw-text rule, measured 50 false refusals of 362 push commands from 23 transcripts; this command).
- [x] Refactor for clarity: functions under the repository's complexity bar, names, comments only where the why is not obvious.

Added after the second signature (finding F1, reproduced by the reviewer), so both signatures are relaunched on the result:

- [x] [AC2] `TestPrepare_RefusesASplitRemote`, two rows: `remote.origin.pushurl` and `url.<other>.pushInsteadOf` point the push at a second bare repository whose `HEAD` is `feat/x`; each asserts a `split-remote` refusal and that repository's refs unchanged. `TestPush_PushURLEqualToURLStillPushes` guards against refusing a `pushurl` equal to the `url`. Recorded 2026-11-20: both split rows failed first, no refusal, and the push force-updated `feat/x`, the default branch of the push target; the equal-URL test passed.
- [x] [AC2] Implement: `resolveRemote` compares `git remote get-url --all` with `get-url --push --all` (both expand `insteadOf`/`pushInsteadOf`) and refuses with `split-remote` when they differ, before any network call; the reason names the remote, never a URL. Result: PASS; mutants "guard off" (both split rows fail) and "always split" (`TestPush_PushURLEqualToURLStillPushes` fails) killed.
- [x] `TestGitEnv_NeverPromptsOnTheTerminal`, then `gitEnv()` adds `GIT_TERMINAL_PROMPT=0` to every git child (precedent: `worktree.list`). Recorded: failed first (`GIT_TERMINAL_PROMPT=1` inherited), then PASS.
- [x] Refactor: `checkRemote` takes the read, default-branch and lease checks out of `Prepare`. Measured with the pinned linter's `gocyclo`: `Prepare` 8 (the review counted about 11 before), every function in the package at most 8, under `AGENTS.md`'s bar of 10.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `go build ./... && go vet ./... && go test ./...` from `cli/`, plus `GOOS=windows go vet ./...` and `GOOS=linux go vet ./...`
- [x] Pinned `golangci-lint run`, and `GOOS=windows golangci-lint run ./...`
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in, with the `--no-verify` mutation result
- [ ] PR opened as a draft referencing this spec folder; `## Knowledge` section; adversarial review by another vendor first, then a second signature (`risk: high`)
