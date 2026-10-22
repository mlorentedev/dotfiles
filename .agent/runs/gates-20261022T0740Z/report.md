# run gates-20261022T0740Z

**In progress, 2 items so far, from 07:40Z.** 0 merged, 2 PRs open (#2920, #2937), 0 parked. Nothing irreversible has happened yet.

## Waiting on you, irreversible first

1. **Two skip-sdd calls, both over the 50-LOC spec-gate line, one each.** This is a judgment call you should see.
   - #2920 (53 lines): a CI-gate defect fixed with the cheapest honest change.
   - #2937 (55 lines): an additive, omitempty output field. No consumer golden-compares the shape.

   The active-spec WIP is at 12 against a limit of 10, so a spec could not be opened without your override.

## Parked

None.

## Done but unmerged

- **#2920** (closes #2918; #2905 is the same bug, filed by another session, to close as a duplicate on landing). With this fix, `dotf spec audit` no longer fails a PR whose merge ref predates an archive on main, so a zombie the default branch already archived (either route) is a WARN.
  - Triage: posted. pr-agent raised no inline findings.
  - Land: blocked by the shared GraphQL limit (exhausted until 08:32Z). The next step is one `dotf pr land 2920 --repo mlorentedev/dotfiles`.
- **#2937** (closes #2873). `worktree list` and `sweep` now name why the forge could not be asked: `pr_merged_unknown`, an `unknown` PR column, and `merge_unknown_skips`. Draft, CI pending.

## Evidence index

| Item | Command | Exit / result |
|---|---|---|
| #2920 | `go test ./internal/spec/ ./internal/cmd/ ./internal/doctor/` | ok |
| #2920 | live audit on `042954df`: 0.77.0 vs dev | exit 1 (FAIL GUARD-003) vs exit 0 (2 WARN) |
| #2937 | `go test ./internal/worktree/ ./internal/lane/ ./internal/cmd/` | ok |
| #2937 | mutations: reason dropped; merged read without its error | tests fail, as intended |
| #2937 | live list/sweep, no-origin repo | reason in the table, JSON and sweep; worktree kept |
| #2937 | live list, dotfiles repo | 44 answered; 8 merged, the same as 0.77.0 |
| both | `go vet ./...`, `GOOS=windows go vet ./...`, golangci-lint (pin) | clean, 0 issues |

## Budget spent

2 items, under the envelope's 4. No dollar figure exists for this run.

## Host-suspend line

Covered: `caffeinate -i -t 43200` held from 07:40Z.

## Next action

Land #2920 after 08:33Z. Then triage and land #2937.
