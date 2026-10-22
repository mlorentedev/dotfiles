# Run tool031-20261022T0735Z: report

**Verdict:** ended `budget-items` (2 of 2) after about 3 hours. 2 items done, 0 parked, 0 merged, 4 PRs unmerged. Nothing irreversible happened: no merge, no release, no secret sync, and nothing was published outside GitHub.

## Waiting on you

1. **Merge the A1 stack in order: #2940, then #2941, then #2942.** #2943 (A2) is based on #2940 and is independent of #2941 and #2942. All four are ready, CI green, triaged, with `dotf pr triage-queue --pr N` at exit 0. Each squash-merge retargets the next PR to `main`, and the next PR then needs a `git merge origin/main`. There is no rebase and no force push.
2. **Standing merge grant.** `.github/merge-grant.yml` has been in dotfiles since #2229 (2026-10-10). I told you earlier that I could not merge; that was wrong. I did not use the grant tonight, because you planned on my statement. Decide whether future runs may use it for this stack.
3. **Second-round reviews.** The fixes pushed after the first triage re-queued PR-Agent on the new heads. Anything it posts after 10:40Z is untriaged.
4. **Unchanged from before:** hive #523 merge and #505 (4.4.2) release, then comment the version on #515. Delete `~/Projects/ci-workflows` (the superseded local import).

## Parked

None.

## Done, unmerged

| PR | Item | Executable / raw LOC |
|---|---|---|
| #2940 | A1a: pool parser, allowlist and prices, `--pool embedded`, `Probe`, `Result` | 184 / 337 |
| #2941 | A1b: `Route` and the differential test (523 cases); lesson-516 | 293 / 414 |
| #2942 | A1c: the `dotf pr route` command | 128 / 167 |
| #2943 | A2: `dotf pr preflight` and its differential test (about 120 cases) | 225 / 349 |

Tickets filed: #2928 (nine script defects, closed by A5) and #2929 (what leaves dotfiles: an ADR question, Backlog).

## Evidence

On each final tip, in the PR bodies:
- `go build ./... && go vet ./...` ok;
- `go test ./... -count=1` exit 0, 48 packages ok;
- `go test -race ./internal/prreview` ok;
- bats route 35/35 and preflight 10/10.

CI is green on every head. Triage comments, with each finding applied or declined with a reason:
- #2940: issuecomment-6337639387
- #2941: 6337639803
- #2942: 6337640182
- #2943: 6338627220

Defects found and fixed during the run:
1. On Windows, the differential tests ran the bash script, and jq's CRLF made the script reject its own allowlist ids. The tests now skip on Windows with that reason. In CI they fail rather than skip when the script's tools are missing.
2. `Probe` forwarded `x-api-key` across redirects when a caller passed its own client.
3. The embedded-pool check skipped when its source was missing.
4. The route's output and exit contract was misdescribed in `tasks.md`.

## Budget

- 6 iterations, about 3h wall clock, 2 items.
- No dollar figure exists for this run.

## Host

`caffeinate -i -t 28800` ran from 07:35Z. The window was covered, and reviews arrived through 10:10Z.

## Next action

Merge #2940.
