# run tool031-20261102T0550Z — ended queue-empty after 6 iterations / 3.7h — 3 done, 2 parked, 1 skipped (held), 2 merged, 0 unmerged

Waiting on a human: one decision on #2893 and two on parked tickets. Nothing is irreversible: the two merges below can be reverted.

## Waiting on you

**Landed**

- #3061 `7c056408`: test that fails when prreview reaches a forge outside GHRunner. Part of #1005, which stays open.
- #3071 `ee93a2b4`: `ai/nan/README.md` says the pool draws by declared weight. Closed #235.

**Decisions:** the three parks below.

## Parked and skipped

1. **#2893, the queue's first item: skipped, held; no park comment, board unchanged.** Reason: lease held. `feat-pr-declare-unreviewed` holds the lease until 2026-11-03T03:09Z, past this run's expiry; #2928 is closed and that worktree is detached and clean, so the holder looks idle, but I did not bypass it with `--share-issue`. Filed as #3076.
   - (A) Run `dotf worktree done` on that worktree, then rerun this item. No new risk.
   - (B) Authorize `--share-issue` for it. Same result; the holder stays.
   - Shape settled with the advisor, ready to build: `dotf pr preflight --pool embedded|<path>`, exactly one of `--pool` or `--model`/`--fallbacks` (else exit 2). The chain is the `openai/` `pr_agent` members, weight descending, pool order breaking ties: mimo, deepseek, glm, which equals today's chain. Document order is out because the pool says order is provenance. A pool with no NaN member exits 2, never 1. Fix `preflight.go:105` to compare with the derived primary, and make `remedy` name the pool, not dotfiles' files, when the source is a pool. Pin the slice with a test.
2. **#257: parked, board Blocked.** Reason: scope (`scripts/**` is frozen) and the plan predates ADR-028. Options are on the ticket: (A) rewrite around ADR-028 with the restore as a `dotf secrets` step, (B) close and record the chain in the existing guides. Unblock: pick one.
3. **#256: parked, board Blocked.** Reason: scope (`**/AGENTS.md` is frozen, `docs/wontfix-ledger.md` is outside write_paths) and the shape is undecided. Options on the ticket: (A) one ledger file, (B) one file per concept. Unblock: pick one and widen the scope.

## Done but unmerged

None.

## Also done

- #1005: `dotf pr guard` inventory posted as a comment. `Backend` versus inputs stays open there.
- #252: stale, comment posted (the PATH claim is now true and the table exists). Not closed; it can be.
- Skipped, not mine: #3053 and #3070 show in the triage queue; both belong to other runs.
- #3061's second commit, which applied PR-Agent's findings, was not re-reviewed; the triage comment says so. CodeRabbit reviewed neither PR (limit).

## Evidence index

| Item | Command | Exit | Output |
|---|---|---|---|
| #3061 | `go test ./internal/prreview/ -count=1`, `golangci-lint run` Linux and Windows | 0 | PR body; CI on `4217e138` |
| #3061 | `dotf pr land 3061` | 0 | `[MERGED] #3061 at 4217e13` |
| #3071 | `dotf pr land 3071` | 0 | `[MERGED] #3071 at 92c7248` |
| #235 | `gh api .../issues/235 --jq .state` | 0 | `closed` |
| #2893 | `dotf worktree add ... --issue 2893` | 1 | refused, three times: 06:0xZ, 07:1xZ, 09:30Z |
| #256 #257 | `gh project item-list` | 0 | `status: Blocked` |

## Budget

6 of 10 iterations, 6 of 6 items, 3.7h of the 11.75h window. Most of it was the shared CI queue. No dollar figure exists for this run.

## Host

`caffeinate` held throughout, so the window was covered. One usage-limit stall happened between about 06:10Z and 07:11Z; no work was lost.

## Next action

Decide #2893: run `dotf worktree done` on `feat-pr-declare-unreviewed` and rerun that item.
