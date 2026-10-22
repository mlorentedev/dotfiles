# Run bugs-20261022t0740z

run bugs-20261022t0740z — ended queue-empty after 3 iterations / about 3h20m (07:40Z to 10:58Z) — 3 PRs landed, 0 parked, 0 unmerged.

Waiting on a human: nothing irreversible beyond the three merges below. Nothing from this run is open.

## Waiting on you

- Landed (irreversible): #2934 (`f8ba5e12`, refs #2812), #2948 (`d968fa5a`, refs #2803), #2951 (`2b0b4e31`, refs #2803).
- #2812 and #2803 were unassigned through REST. Their bitácora Status may still read In Progress: assigning flips it, unassigning may not, and GraphQL was rate-limited for the user for most of the run. Check the board and set them back to Backlog.
- #2803 changed approach: the keys go in the test process's environment instead of a repo-config helper, because repo config does not reach clones, a bare remote's `receive-pack`, or the git the code under test starts. Disclosed on the issue and in #2948. Confirm you accept it.
- Four other sessions' PRs (#2920, #2921, #2923, #2924) were untriaged in the queue at arming. They were left to their owners.

## Parked

None.

## Done but unmerged

None.

## Evidence index

| Item | Command | Exit | Where |
|---|---|---|---|
| #2934 | `go test ./internal/hooks/` (new test red before the fix, green after); `go test ./...` | 0 | PR #2934 body and `## Review triage` |
| #2934 | CI on `a82cfb92` | all green | PR checks |
| #2948 | `go test -count=3 ./internal/{gittest,initrepo,lane,converge}/`; `go test ./...`; vet, `GOOS=windows go vet`, golangci-lint v2.12.2 | 0 | PR #2948 body and triage |
| #2951 | `go test -count=3 ./internal/{harness,vault}/`; vet, Windows vet, pinned lint | 0 | PR #2951 body and triage |

## Review

- PR-Agent reviewed every PR. CodeRabbit posted only rate-limit notices on all three.
- #2934: one REAL finding declined with measurement: `0o300` is `-wx------`, not readable.
- #2948: two compliance findings declined (spec-gate green under the threshold; evidence in the body).
- #2951: one DoD finding applied (the triage comment).
- The PR author did every triage; this is disclosed on each PR.

## Decisions

- Scope was chosen at arming with the advisor, because the operator gave no filter: unassigned mechanical bugs, ordered #2872, #2812, #2803, with `max_loc` 50 while specs are at 11 of 10 WIP.
- #2872 was claimed by another session at 07:35Z and skipped.
- #2812: hooks site only. The sshtarget `Include` site needs a design decision for that lane, and `.golangci.yml` is outside the run's write scope.
- `dotf agent auto` is denied on this Mac (no machine identity, #1236/#1547), so the advisor was the harness one the operator named.

## Budget

3 iterations of 6, 3 items of 3. No dollar figure exists for this run.

## Host

`caffeinate -i -t 28800` held from 07:40Z, on AC power; the window was covered.

## Next action

Check #2812 and #2803 on the board (Status back to Backlog), then free one WIP slot by landing #2910.
