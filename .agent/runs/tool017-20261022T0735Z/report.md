# Run tool017-20261022T0735Z

**Verdict:** run tool017-20261022T0735Z ended `queue-empty` (the remaining TOOL-017 work waits on the owner) after 2 items, about 07:35Z to 11:10Z. 2 done, 0 parked, 1 merged (#2935), 1 unmerged (#2924, the owner's). What waits on a human: the #2924 merge, which needs one `harness/` refresh first. Already irreversible: vault commit `5d6a49c5`.

## Waiting on you, irreversible first

- **Vault `5d6a49c5`** (pushed to master): the `pr-review-triage` skill source now has the REAL/refuted rule from #2924. Every machine that pulls the vault sees it. The rendered record on main matches it only once #2924 lands.
- **#2924** (AC4/AC6/AC8, the reproduction gate). It touches `harness/`, so the owner merges it. `test-macos` is red at `c3cf03a6` for one expected reason: the record's `generated_sha` names the old vault source (`tests/harness-generated-sha.bats:73`). Before merging, on its branch:
  `git -C "$VAULT_PATH" pull --ff-only && scripts/compile-harness.sh --refresh`
  That should change one line, `generated_sha: 8caa08e3c9c33dc0`. It is written up on the PR. Main is merged in, and the triage is posted. `dotf pr triage-queue --pr 2924` exits 0.

## Parked

None.

## Done

- **#2935** merged as `eb272d10`, closing #2926: `dotf pr reproduce --defect-when-passes`; the row quotes its command and escapes `|` for the table cell; `classify` lost a dead parameter; the #1059 replay is ticked with evidence rerun at `1719024f`; lesson 518. One REAL finding was refuted by a reproduction against GitHub's own GFM renderer. My first version of that check was wrong: it grepped for `<code>` and missed `<code class="notranslate">`. That is disclosed on the PR. `af9f24c4` (a comment) and `fbc5fef2` (main merged in, lesson renumbered 514 to 518) were not re-reviewed, and the PR says so.

## Filed or fed

- #2954 (a peer's issue: pr-agent reads a truncated PR body). I added #2924 and #2903 as two more instances. AC4 now prices each such finding as a reproduction.

## Evidence index

| Item | Command | Exit |
|---|---|---|
| #2935 | `cd cli && go test ./...` | 0 |
| #2935 | `golangci-lint run` | 0 (0 issues) |
| #2935 | `GOOS=windows go vet ./...` | 0 |
| #2935 | #1059 replay, `dotf pr reproduce --probe {flagless,multiline}.diff --defect-when-passes -- bats tests/ci-age-pin.bats` at `1719024f` | did-not-reproduce / reproduced |
| #2935 | `dotf pr land 2935` | 0, `[MERGED]` |
| #2924 | `go test ./internal/prtriage/ ./internal/reproduce/ ./internal/cmd/` after the main merge | 0 |
| #2924 | `dotf pr triage-queue --pr 2924` | 0 |
| repo | `dotf pr triage-queue` | 1: #2951 only, another session's PR (fix/git-fixture-quiet-gc-harness-vault) |

## Budget

2 of 4 items; about 3.5h of the 8h window. No dollar figure exists for this run. The GraphQL rate limit blocked `gh pr ready` from 07:40Z to 08:32Z. The pr-agent queue was about 10 runs deep around 09:45Z.

## Host

`caffeinate -i -t 28800` from 07:35Z. The window was covered, and every background poll returned on schedule.

## Next action

Owner: refresh the record on #2924 and merge it. Then run the TOOL-017 final verification (tasks.md lines 58-61), followed by `dotf spec review TOOL-017-deterministic-review-triage`.
