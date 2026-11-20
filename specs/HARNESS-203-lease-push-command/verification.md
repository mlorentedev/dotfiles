---
tags: [spec, verification, templates]
created: "2026-11-19"
id: "HARNESS-203-lease-push-command"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-203-lease-push-command

## Evidence

- [x] AC1 (lease push moves the remote; stale `--expect` refused) -> commits `ff5be766`, `b33e2871`, `369a3d55` / tests `TestPush_MovesTheRemoteAfterARebase`, `TestPush_StaleExpectLeavesTheRemote` (2 rows), `TestPrPushLease_PushesAndPrintsTheMove`. Row (b) of the stale test failed first: git printed `Everything up-to-date` and exited 0 with a wrong lease (lesson 615).
- [x] AC2 (refusal table, exit 1, remote refs unchanged) -> commits `483b01a2`, `369a3d55` / tests `TestPrepare_Refuses` (16 rows, stable `Code` per row, all remote refs compared after an unpushed commit), `TestPrepare_RemoteResolutionOrder` (5 rows), `TestPrPushLease_Refusals` (exit 1 for refusals, 2 for usage). The first 12 rows all failed before the refusals existed; the `--remote <path>` row pushed successfully.
- [x] AC3 (pre-push hook runs and can stop the push) -> commit `868eb87d` / test `TestPush_PrePushHookRuns`.
- [x] AC4 (settings pinned, AGENTS.md line, ADR with alternatives) -> commits `31ce8816`, `b19e866d` / tests `TestPushLeaseRules_RawForcedPushStaysDenied`, `TestPushLeaseRules_NoRuleNamesTheCommand`, `TestPushLeaseRules_MatcherMatches`; `AGENTS.md` "Force-updating your own branch"; `docs/adr/adr-066-agents-force-update-their-branch-through-dotf-pr-push-lease.md`.

## Mutation results (2026-11-20, by hand, each mutant applied alone and reverted)

| Mutant | Result |
|---|---|
| default branch from `recorded` only (drop what the remote reports) | killed: 2 rows of `TestPrepare_Refuses` |
| default branch from `reported` only (drop `origin/HEAD`) | killed: 1 row |
| `--expect` accepts `[0-9a-f]{7,40}` | killed: 2 rows |
| stale-lease comparison disabled | killed: `TestPush_StaleExpectLeavesTheRemote` row (b) |
| detached HEAD read on exit code 99 instead of 1 | killed: detached HEAD row |
| `--no-verify` added to the push argv | killed: `TestPush_PrePushHookRuns` |
| `Bash(git push *--force*)` removed from the template's deny | killed: `TestPushLeaseRules_RawForcedPushStaysDenied` |
| `Bash(dotf pr:*)` added to the template's allow | killed: `TestPushLeaseRules_NoRuleNamesTheCommand` |
| leading-`-` remote guard disabled | survived, equivalent: `git remote get-url` already rejects an option-like name, so the row is still refused as `unknown-remote`. The guard stays as defence for the later `ls-remote` and `push` calls |

## Test status

- `go test ./internal/pushlease ./internal/cmd ./internal/claude -count=1`: PASS (pushlease 26 subtests and tests, about 50 s on Windows).
- `go build ./...`, `go vet ./...`, `GOOS=windows|linux|darwin go vet` on the changed packages: clean.
- `golangci-lint run` 2.12.2 (the pinned version), host, `GOOS=windows` and `GOOS=linux`, on the changed packages: no finding other than `gofmt` on working copies checked out with CRLF (`core.autocrlf`); the committed blobs of every changed Go file pass `gofmt -l`.
- Full suite `go test ./...` on Windows, run while three lint passes ran alongside: every package passed except `internal/worktree`, which hit go test's 10-minute timeout (637 s) inside a git process start in `TestListAsksTheLandedHeadsTheCallerHandsIt`. This branch touches nothing under `internal/worktree`; rerun alone with `go test ./internal/worktree -count=1 -timeout 40m`, it passed (`ok ... 945.615s`). On the PR (#3614, head `775f9f6b`) every CI job passed: `test (ubuntu-latest)`, `test (windows-latest)`, `test-windows`, `test-macos`, `integration`, `cli-lint`, `knowledge-gate`, `hygiene`.
- Manual smoke test: none beyond the tests, which run the real git binary against local bare repositories; the command was not run against GitHub.

## Decisions made during implementation

- The stale-lease pre-check (`ls-remote` before pushing) was added after row (b) of the stale test failed: git does not consult a lease on a ref that already equals the pushed commit.
- One `ls-remote --symref <r> HEAD refs/heads/<b>` reads the default branch and the branch sha together; an unreadable remote is a refusal (`remote-unreadable`).
- The default branch is refused by union of what the remote reports and what `refs/remotes/<r>/HEAD` recorded, instead of "recorded, else remote": a stale `origin/HEAD` would otherwise let the new default through. Added a row for it.
- `main`, `master` and the default branch compare case-insensitively: refusing `Main` costs nothing.
- The push names the HEAD sha read once, not `HEAD`, so the printed line and the push agree.
- `Refusal` carries a stable `Code`; tests assert the code, not the prose (`cli/AGENTS.md`).
- Tests run real git against temporary bare repositories instead of a `Deps` seam: AC1 asks for a real repository, and the behaviour under test (lease, hooks, `ls-remote`) is git's own.
- `--expect` missing is a refusal with exit 1, not a cobra required-flag usage error (exit 2), as AC2 states.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-615-git-does-not-check-a-lease-on-a-ref-already-up-to-date.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-066-agents-force-update-their-branch-through-dotf-pr-push-lease.md
- [x] New pattern candidate for `00_meta/patterns/`? no: one occurrence; "move a capability out of shell text into a command that builds argv" may become one if a second deny rule meets the same pressure.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-203-lease-push-command/` -> `specs/archive/HARNESS-203-lease-push-command/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
