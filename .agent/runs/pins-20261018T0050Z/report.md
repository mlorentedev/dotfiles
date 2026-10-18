# Run pins-20261018T0050Z

**run pins-20261018T0050Z — ended queue-empty after 2 iterations / about 2h30 (00:50Z to 03:20Z) — 2 done, 0 parked, 2 merged, 0 unmerged.**

Nothing waits on a human from this run. Both changes landed and are reversible with a revert; nothing was released or published. They reach installed boxes only through the next release.

## Waiting on you — irreversible first

- **Landed:**
  - #2735 → `b99729e3` (closes #2711). Landed under the standing grant: `## Review triage` comment issuecomment-6244639286, CI green on head `6fbd5b1f`, `dotf pr triage-queue --pr 2735` exit 0.
  - #2742 → `d18ccf0f` (closes #2677). Merged by the owner's account at 03:19:50Z, one minute after the `## Review triage` comment (issuecomment-6245118257) and with CI green on head `9bea91af`. `dotf pr land` was not the path that merged it.
- **Release.** Neither fix reaches a box until a release carries it.

## Parked, with the question restated

None.

## Done but unmerged

None.

## Evidence index

| Item | Verification | Exit |
|---|---|---|
| #2711 / #2735 | `go build/vet/test ./...`, `golangci-lint run` (0 issues), `GOOS=windows go vet ./...`; mutations 6/6 killed | 0 |
| | Live plan on the Mac from the `wintarget` worktree. The installed 0.74.0 planned `mise config to write`. The fix printed the "converging the declared checkout" note and planned what main plans | 0 |
| #2677 / #2742 | `go build/vet/test ./...`, `golangci-lint run` (0 issues), `GOOS=windows go vet ./...`. Mutations 4/5 killed; the survivor (dropping `p.ConfigChanged`) is equivalent and costs one extra `mise ls`, disclosed in the PR | 0 |
| | Isolated mise (four `MISE_*_DIR`s in a scratch dir): `mise prune --yes --tools http:zsh-autosuggestions` removes the tool, and the listing comes back `{}` | 0 |
| | Converge transcripts `20261017T081222Z` (no prune at all) and `20261017T184406Z` (prunes `gh` only), both failing on `http:zsh-autosuggestions`: the stale list | n/a |

The full outputs are in each PR body.

## Deviations and gaps, disclosed

- **Neither of #2677's candidates was the cause.** The cause was a third one: the orphan list is taken before the config write that changes it (lesson 478).
- **No live reproduction of #2677.** pr-agent asked for a check on the real box; declined with a reason in the triage. #2667 merged and main pins `zsh-autosuggestions`, so reproducing it would uninstall a tool the machine wants.
- **Main was red, not because of this run.** From #2729 (`e6eb0f76`) until #2746 (`ed1a807a`), HARNESS-041 failed on `.agent/` and `test`, `test-macos` and `test-windows` failed on every PR. The fix was in `tests/**`, outside the envelope, so I reported it to dotfiles-20 and held #2742. #2746, the owner's, fixed it.
- **A denied push.** The rebase plus `--force-with-lease` of #2742 was denied. After the owner's go-ahead the branch was updated with a merge from main (REST `update-branch`, `9bea91af`), with no force.
- **One reviewer per PR.** CodeRabbit posted only "Review limit reached" notices; pr-agent was the reviewer on both.
- **Power outage.** The background tasks died, including the wake-lock. It was relaunched (`caffeinate -i -t 34000`); the gap was minutes, and no work ran during it.
- **A denied reproduction.** The first mise reproduction (with `rm -rf` and `env -i`) was refused. It was redone with `mktemp -d` in the session scratchpad and exported variables.
- **A test written in the wrong worktree.** The #2677 test began in the #2711 worktree. It moved by patch, and the #2711 worktree was restored before its push.
- **This report's push** goes through the skip-sdd escape. The run branch carries only `.agent/`, and it is never merged (#2746 removed `.agent/` from main).

## Knowledge

- Lesson 478: a list taken before the write that changes it is stale by construction (#2742).
- Lesson 474, amended 2026-10-18: converge now defaults to the declared checkout (#2735).
- ADR: none.
- Runbook: none. `cli/README.md` describes the new converge default.

## Debt filed

- #2744: the dry-run plan does not name the tools that a mise config write drops. Option (A) is recommended: diff the deployed config against the rendered one.
