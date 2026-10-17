# Run pilot-r6-20261017T1825Z

**Verdict:** run pilot-r6-20261017T1825Z ended `queue-empty` after 1 iteration, about 1h: 1 done and merged (#2702), 2 parked (#2448 whole, #1975 options 1 and 2), 0 unmerged. Question this answers: what is waiting on a human? **Two design decisions, nothing irreversible.**

## Waiting on you, irreversible first

Landed (reversible, one line of prompt text): **#2702** at `d66265a`, under the standing grant. It adds a scope line to the reviewer prompt (option 3 of #1975): the gate's implementation lives outside the repo, no walk of the home directory, mark what is unreachable UNVERIFIED. The triage disclosed that the run authored and triaged it; PR-Agent was the only reviewer (CodeRabbit sent its rate-limit notice).

Nothing else is awaiting a merge.

## Parked, with the question restated

- **#2448 (HARNESS-181, the reviewer's time budget is advisory and the cap discards the round).** The ticket says "Options (not decided)", so the 3 a.m. test fails, and it needs a spec (WIP 13 of 10, no override). Board Status is `Blocked` (verified through `gh issue view 2448 --json projectItems`). (A) Write-first protocol: the reviewer writes a provisional `review.md` (`verdict: PENDING`) and refines it; runner-agnostic and testable with a fake runner. (B) Steer at the soft mark through the runner's input channel; smaller, but needs a per-runner probe on pi and agy. Cheapest unblock: choose A or B and free a WIP slot.
- **#1975 options 1 and 2 (a wedged tool call costs the round).** Option 1 needs the runner's per-tool deadline, which nothing in this repo passes or probes (a live check); option 2 needs a stall threshold, a rule for which process to kill, and a spec. (A) Wedge detection in the launcher: transcript not growing for N minutes while the process lives, fail loudly naming the tool call. (B) Per-tool deadline per runner. Cheapest unblock: choose A or B and free a WIP slot. #1975 stays open and `In Progress` (it `Refs` #2702).

## Done but unmerged

None.

## Evidence index

| Item | Command | Result |
|---|---|---|
| #2702 test red first | `go test ./internal/spec/ -run TestReviewPromptKeepsTheReviewerInsideTheRepository` before the change | FAIL (four strings missing) |
| #2702 verification | `go test ./internal/spec/ ./internal/cmd/ -count=1`, `go vet`, `GOOS=windows go vet`, pinned `golangci-lint run ./internal/spec/...` | ok, ok, clean, 0 issues |
| #2702 land | `dotf pr triage-queue --pr 2702`, then `dotf pr land 2702` | exit 0 queue clear; `[MERGED] #2702 at d66265a` |
| Board | `gh issue view 2448 --json projectItems`, same for 1975 | Blocked, In Progress |

## Tickets filed

#2704: the autopilot skill's board pre-flight reads a truncated window of a 4891-item board (found while locating the two tickets; the skill lives in the vault, owner-lands).

## Budget spent

1 iteration of 4, about 1h of the 10h window, 2 of 2 queue items handled. No dollar figure exists: the provider exposes none.

## Host-suspend line

Window covered by `caffeinate -i -t 36000`. The run ended early, so no suspend was observed.

## Not done and why

- No spec was opened: WIP is 13 of 10 and the owner granted no override.
- #2659's skill-record part: not in this envelope, and a peer (review-cli-119-8d, #2470) already rewrote those vault lines (vault `626847e9`). Only the `repo-hygiene.yml` comment remains for the owner.

## Next action

Pick A or B for #2448 and for #1975, and free a WIP slot so one spec can open.
