# Run identity-20261022T0736Z

Armed 07:36Z, window to 15:30Z, disarmed about 11:20Z: queue exhausted for the declared filter. Items used: 3 of 4 landed, 1 parked.

## Landed
| PR | Merge | Closes |
|---|---|---|
| #2847 | `4a70ad7d` | binding floor raised to 0.77.0 (HERDR-001 shipped, spec archived after a round 3 PASS) |
| #2925 | `f358e4c` | #1236, #1547: `dotf env identity [set <ID>] [--replace]`, doctor row "Machine identity", ADR-032 section 8 amendment, lesson 515 |
| #2946 | `e708355` | #2204: doctor row "GitHub token scopes", `forge.RequiredGHScopes`, both runbooks request every scope at login |
| #2949 | `494f3c0` | #1909: a 404 from a repository other than the home one is "not verified", never FAIL |

## Parked
- #1881 (doctor: handoff lost a thread): A/B comment posted. **The board Status was not set to Blocked**: GraphQL rate limit exceeded, so the park signal is the comment and the removed assignee only.

## Review dispositions
- #2925: one REAL pr-agent finding (blank `machine.id` refused without `--replace`), fixed in `f358e4c`.
- #2946: three CodeRabbit findings applied (`--hostname github.com`, request status before headers, env token not refreshed); one pr-agent question declined with reason.
- #2949: pr-agent asked for command output in the body (applied).
- All triage was by the PR author, disclosed on each PR. The fix commits of #2925 and #2946 were not re-reviewed by a second reviewer beyond the automatic re-run.

## Notes
- pr-agent runs queued serially (up to 9 pending): `review` took 40 to 60 minutes per push. CodeRabbit was rate-limited on most heads.
- `knowledge-gate` fails on an unclosed code fence in the PR body, naming the line; a body edit re-triggers it, a rerun of the old job does not.
- A heredoc-heavy `cat > file <<EOF` combined with a `go test` in one Bash call hung and wrote nothing; use the Write tool for files.

## Owner
- Keyboard checks on #2814 (open, close from inside, stranger yazi untouched), then close it.
- After a release carrying #2925: `dotf env identity set macmini` on this Mac, so dispatch stops denying non-local pools.
- After a release carrying #2946: `gh auth refresh -h github.com -s read:project` where doctor warns.
- Decide #1881 (A recommended).
- Stale worktrees: feat-herdr-explorer-binding, fix-herdr-toggle-focused-explorer, chore-autopilot-herdr-20261021; local branch `feat/herdr-explorer-binding`.
