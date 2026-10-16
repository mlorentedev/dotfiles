# Run cli-2026-10-16: disarm report

**Verdict:** run cli-2026-10-16 ended with `queue-empty` at 06:09Z, about 1.4 h after arming at 04:45Z. All 5 items are done: 4 PRs merged, 0 parked, 0 unmerged. Nothing irreversible is waiting on you.

## Waiting on you: irreversible first

- Nothing irreversible from this run. The four merges below were own PRs under the standing grant, landed with `dotf pr land`. None touches `.github/**`, release automation or the envelope.
- **#2591 (filed this run).** PR-Agent's workflow-level concurrency lets an `opened` run cancel the `ready_for_review` run, so a draft marked ready at once is never reviewed. Measured on #2589. The fix is in `.github/workflows/pr-agent.yml`, which is frozen for the run, so the decision is yours. Option (B) in the issue closes the class.
- Carried over, not from this run: release #2571 (0.72.0) and land #2580, then #2582.

## Parked

None.

## Done

| Item | PR | State | Reviewed by |
|---|---|---|---|
| #2519 BUG-146: full shas in the provenance mismatch | #2589 | merged `d7485473` | PR-Agent, after a manual `/review` (see #2591) |
| #2540 CLI-121: `doctor --fix` refreshes a stale opencode catalog | #2592 | merged `7c4a1c5f` | PR-Agent; the real `opencode models --refresh` was verified live |
| #1281 CLI-052: session-start triage probe from a subdirectory | #2593 | merged `da4beffe` | PR-Agent; the probe was verified live from `cli/` |
| #2486 CLI-116: a step's output precedes its result line | #2590 | merged `79cf7e5d` | PR-Agent at ad9972e3, then CodeRabbit at 05:43Z. CodeRabbit's Major finding (the barrier's own writes were unbounded under the mutex) was applied in 18514b7b, which no reviewer re-read. The merges and lesson renumbers are disclosed on the PR. |
| #2414 CLI-112 | none | closed earlier, with evidence that #2574 fixed it | |

CodeRabbit was rate-limited on every PR except #2590, where it reviewed once its limit reset. Each triage comment says which reviewers ran.

## Debt found on the way

- Flaky `TestResolveWorktreeAndMainRepoRoot`: it took the machine-wide worktree lock. Fixed inside #2590 (commit `1d96234d`).
- The PR-Agent cancellation race, filed as #2591.
- The lesson number collided twice with peers (449, then 451). The #1514 guard (`lessons fmt` refuses duplicates) caught it, so no new ticket was filed. The lesson is now 452.
- `test-windows` went red on #2590 because of the CLI-119 zombie spec. That came from main: #2528 was closed before the spec archived. The peer's #2597 fixed it, and re-merging main cleared it.

## Evidence index

| Item | Command | Exit |
|---|---|---|
| #2590 | `go test -race` ordering, split-mark, prefix and full-pipe tests (the full-pipe test fails on the previous `sync`); `go test ./internal/cmd/ ./internal/worktree/`; `golangci-lint run` | 0, 0, 0 issues |
| #2592 | `go test ./internal/doctor/`; mutation of the mtime guard turns a row red; live `opencode models --refresh` exits 0 in 4 s | 0 |
| #2593 | the test fails without the fix and passes with it; live `mem session-start` from `cli/` renders `[pr-triage]` (installed 0.71 renders nothing) | 0 |
| all lands | `dotf pr triage-queue --pr N`, then `dotf pr land N`, plus a manual read of `pulls/N/reviews` | 0 |
| exit | repo-wide `dotf pr triage-queue` at 06:09Z: "[OK] no reviewer output is awaiting a disposition" | 0 |

The full outputs are in each PR's Verification section and its `## Review triage` comment.

## Budget spent

5 items of 5, about 84 minutes of wall clock. There was 1 failed iteration: `test (windows-latest)` on ad9972e3 ("0 of 4000"), fixed in the next commit. The run never had consecutive failures. No dollar figure exists for this run.

## Host suspend

Covered for the run's active span. `caffeinate -i -t 36000` (pid 62792) was alive throughout. The run ended early with `queue-empty`, about 8.5 h before the envelope's 14:45Z expiry. The lock was released at disarm (06:12Z).

## Next action

Owner: decide #2591 (the PR-Agent cancellation race, a frozen `.github/**` path). Option (B) is recommended.
