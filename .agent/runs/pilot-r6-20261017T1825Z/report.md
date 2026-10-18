# Run pilot-r6-20261017T1825Z

**Verdict:** run pilot-r6-20261017T1825Z ended `queue-empty` after 1 iteration, about 1h: 1 done and merged (#2702), 2 parked (#2448 whole, #1975 options 1 and 2), 0 unmerged. Question this answers: what is waiting on a human? **Two design decisions, nothing irreversible.**

## Waiting on you, irreversible first

Landed (reversible, one line of prompt text): **#2702** at `d66265a`, under the standing grant. It adds a scope line to the reviewer prompt (option 3 of #1975): the gate's implementation lives outside the repo, no walk of the home directory, mark what is unreachable UNVERIFIED. The triage disclosed that the run authored and triaged it; PR-Agent was the only reviewer (CodeRabbit sent its rate-limit notice).

Nothing else is awaiting a merge.

## Parked, with the question restated

- **#2448 (HARNESS-181, the reviewer's time budget is advisory and the cap discards the round).** The ticket says "Options (not decided)", so the 3 a.m. test fails, and it needs a spec (WIP 13 of 10, no override). Board Status is `Blocked` (verified through `gh issue view 2448 --json projectItems`). (A) Write-first protocol: the reviewer writes a provisional `review.md` (`verdict: PENDING`) and refines it; runner-agnostic and testable with a fake runner. (B) Steer at the soft mark through the runner's input channel; smaller, but needs a per-runner probe on pi and agy. Advisor recommendation: **A**. B's premise is unverified (both runners are one-shot, so an input channel at the soft mark probably does not exist), and the soft mark failed because it asks for compliance after 241 tool calls, while write-first asks at turn 1. Cheapest unblock: free a WIP slot (shared with #1975-A, see below).
- **#1975 options 1 and 2 (a wedged tool call costs the round).** Option 1 needs the runner's per-tool deadline, which nothing in this repo passes or probes (a live check); option 2 needs a stall threshold, a rule for which process to kill, and a spec. (A) Wedge detection in the launcher: transcript not growing for N minutes while the process lives, fail loudly naming the tool call. (B) Per-tool deadline per runner. Advisor recommendation: **A**. The repo already chose a launcher-side bound once (HARNESS-152, because per-runner flags covered only agy); B reintroduces that per-runner gap. The spec must settle N from measured transcript timestamps, and the first increment fails loudly and names the call. #1975 stays open and `In Progress` (it `Refs` #2702).

**One spec, not two.** #1975-A and #2448-A share the launcher's termination path, so they are one spec ("a stopped review run leaves a record and names why"). The recommendations are posted on both park comments.

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

Free one WIP slot and open one spec covering #1975-A and #2448-A.

## Addendum, 2026-10-18: what happened after the run ended

The run's park was superseded by an owner-authorised chain (relayed by session dotfiles-20): spec CLI-094 covered #1975-A and #2448-A and landed in three increments (#2712 spec, #2719 PENDING refusal, #2722 write-first prompt, #2726 stall watcher and stop record). Round 1 of the independent review failed on a real blocker (the transcript sink buffered until EOF, so the watched file stayed empty; fixed in #2743 with lesson 482). Round 2 passed with gaps and CLI-094 was archived in #2779, closing #1975 and #2448. The follow-ups are #2777 (measure inter-event gaps before trusting the 10m stall timeout) and #2778 (stop lines reach only the `.stderr` side file). Report of that arc: `.agent/runs/cli094-20261018T1035Z/report.md` on chore/autopilot-cli094-20261018.

Corrections and lessons for this run itself:
- The wake-lock died three times (a `caffeinate` started with `&` dies with the tool call; `run_in_background` survives). A power cut then killed the tmux server and the scratchpad; the round-1 `review.md` was intact in the review worktree, so my first "lost" wording was wrong.
- `dotf pr land` waited about 20 minutes on the shared GraphQL budget at times; REST (`gh api`) did not.
- The run authored and triaged its own PRs and only PR-Agent reviewed them; that was disclosed in each triage comment.
