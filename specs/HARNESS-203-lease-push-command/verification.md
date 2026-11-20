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
| `git remote get-url` check disabled (after review finding M2 added URL rows) | killed: 4 rows (`not configured`, `path`, `https URL`, `scp-style URL`) refuse as `remote-unreadable` instead of `unknown-remote`; `--mirror` stays refused by the leading-`-` guard |
| split guard off (`pushurl` / `pushInsteadOf` admitted; after second-signature finding F1) | killed: both rows of `TestPrepare_RefusesASplitRemote`, the push target's `feat/x` force-updated |
| split guard always on | killed: `TestPush_PushURLEqualToURLStillPushes` |
| leading-`-` remote guard disabled | survived, equivalent: `git remote get-url` already rejects an option-like name, so the row is still refused as `unknown-remote`. The guard stays as defence for the later `ls-remote` and `push` calls |

## Test status

- `go test ./internal/pushlease ./internal/cmd ./internal/claude -count=1`: PASS (pushlease 26 subtests and tests, about 50 s on Windows).
- `go build ./...`, `go vet ./...`, `GOOS=windows|linux|darwin go vet` on the changed packages: clean.
- `golangci-lint run` 2.12.2 (the pinned version), host, `GOOS=windows` and `GOOS=linux`, on the changed packages: no finding other than `gofmt` on working copies checked out with CRLF (`core.autocrlf`); the committed blobs of every changed Go file pass `gofmt -l`.
- Full suite `go test ./...` on Windows, run while three lint passes ran alongside: every package passed except `internal/worktree`, which hit go test's 10-minute timeout (637 s) inside a git process start in `TestListAsksTheLandedHeadsTheCallerHandsIt`. This branch touches nothing under `internal/worktree`; rerun alone with `go test ./internal/worktree -count=1 -timeout 40m`, it passed (`ok ... 945.615s`). On the PR (#3614, head `775f9f6b`) the build and test jobs passed: `test (ubuntu-latest)`, `test (windows-latest)`, `test-windows`, `test-macos`, `integration`, `cli-lint`, `knowledge-gate`, `hygiene`. Three gates stay red until the closing steps, by design: `spec-gate` (and its `gate` job) fails with "archive-on-merge violation" because the PR closes #3612 while this spec is still active, and the archive is the last commit of this PR, after the review; `review-attestation` reports no reviewer output while the PR is a draft.
- Manual smoke test: none beyond the tests, which run the real git binary against local bare repositories; the command was not run against GitHub.

- `GOOS=windows golangci-lint run ./...` (2.12.2) over the whole tree: no finding other than `gofmt` on CRLF working copies.

## Review dispositions

First signature: `review.md`, `nan/deepseek-v4-flash`, verdict PASS at `43292f73`. Four earlier launches ended without a verdict and are kept as `review-round1..3.md` plus the timed-out round: two with Bitwarden locked, one with the runner inheriting Claude Code's `BUN_INSPECT` (#3615), one stopped at the 45-minute deadline.

| Finding | Disposition |
|---|---|
| M1: the session-start banner flags this spec from `review-transcript.jsonl` | Ticketed: #3616 (BUG-177). Pre-existing, outside this diff; the archive gate is unaffected. |
| M2: the row named for a URL passed a path; no real URL row | Applied in `f63e62f2`: row renamed to `path`, `https` and scp-style URL rows added, the `get-url` mutant above kills all of them. Test-only, outside the contract files. |
| M3: `tasks.md` Closing unticked | Applied: ticked what holds. The review said ticking would invalidate it; it does not, because the contract digest folds checkbox state (`normaliseContract`, #998). The last box waits for the second signature. |
| M4: AC4's "full gates pass" holds only after the archive commit | Accepted as structural: `spec-gate` turns green with the archive commit, `review-attestation` once the PR leaves draft and a reviewer answers. |
| Q: `leaseRuleMatches` is an emulation | No action, as the review concluded: the command text contains no literal any deny rule names. |

Second signature: `review-second.md`, `anthropic-review/claude-sonnet-5-5`, verdict PASS-WITH-GAPS at `22709ec4`. F1 changed the contract (a new refusal in `proposal.md`), so both signatures are relaunched on the head that carries the fix; the round at `22709ec4` is kept as the record of what F1 was found against.

| Finding | Disposition |
|---|---|
| F1 (REAL): `ls-remote` reads the fetch URL while the push writes `pushurl`; reproduced as a force-update of the push target's default branch | Applied: `split-remote` refusal when `get-url --push --all` differs from `get-url --all`, before any network call. The rows reproduce the reviewer's case and a `pushInsteadOf` variant it did not name; both failed first with the push landing. Contract updated (What, Risks, AC2), ADR-066 §2 and Consequences, `--help`. |
| F2: long-lived branches other than the default are not protected | Recorded as a risk in `proposal.md` and ADR-066 Consequences: they rely on the forge's branch protection. Proposed as accepted, pending the owner's confirmation on the PR. |
| F3: no timeout and no `GIT_TERMINAL_PROMPT=0` | Prompt applied: `gitEnv()` sets `GIT_TERMINAL_PROMPT=0` for every git child, following `worktree.list`, with `TestGitEnv_NeverPromptsOnTheTerminal`. Timeout ticketed: #3617 (HARNESS-205). |
| F4 (REAL): `Prepare` about 11, above `AGENTS.md`'s bar of 10 | Applied: `checkRemote` extracted; measured with the pinned linter's `gocyclo`: `Prepare` 8, no function in the package above 8. |
| F5: `--expect` read fresh just before pushing defeats the lease | No action: an accepted risk already in `proposal.md` and ADR-066. |
| Q: no run against a hosted remote | See the smoke test line under Test status. |

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
