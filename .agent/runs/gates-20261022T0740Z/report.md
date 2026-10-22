# run gates-20261022T0740Z

**Ended queue-empty after 3 items, 07:40Z to 11:35Z.** 3 merged (#2920, #2937, #2952), 0 parked, 3 filed. Nothing irreversible is waiting on a human. All three merges are revertible squash commits.

## Waiting on you, irreversible first

1. **Two skip-sdd judgment calls, already landed: confirm or revert.** Both PRs crossed spec-gate's 50-LOC line by a few lines and went in under `skip-sdd`, with a rationale in each body. Active specs are at 11 against the limit of 10, so a spec would have needed your override.
   - #2920 (53 lines): a CI-gate defect fixed with the cheapest honest change.
   - #2937 (55 lines): additive `omitempty` output fields. No consumer golden-compares the shape.
2. **#2954: set `max_description_tokens` in `.pr_agent.toml`.** pr-agent reviewed #2937's 3,225-character body truncated and flagged the Knowledge and skip-sdd sections as unseen. The likely cause is pr-agent's default of 500 description tokens. `.pr_agent.toml` is on the human floor and was frozen in this run. The tool-017 session was told the number.

## Parked

None.

## Done

- **#2920** (`acfcbce2`, closes #2918; #2905 closed as its duplicate). `dotf spec audit` no longer fails a PR whose merge ref predates an archive on main. A zombie the default branch already archived, by either route (`specs/archive/`, `specs/archive/_abandoned/`), is now a WARN that says to merge the base. Lesson 514.
  - Proven on main after the merge: the test-windows job of CI run `39755955995` on `acfcbce2` passed its "Spec issue-state audit (CI-004 AC8)" step, and so did run `39759956974` on `3679ebcc`.
- **#2937** (`f5c9f79d`, closes #2873). `worktree list` and `worktree sweep` now say why the forge could not be asked: `pr_merged_unknown` in the JSON, `unknown` in the PR column with the reason, and `merge_unknown_skips` in the sweep. A forge error is never read as merged.
- **#2952** (`3b5d1d27`, closes #1644, GUARD-018). A `review: waived` waiver no longer archives a spec whose signed review, from either signer, says `verdict: FAIL`. The refusal names the re-review exit. An abandoned spec still archives through the recorded `--force-without-review --reason`, and its `review_bypass:` names the overridden FAIL. A waiver beside a PASS, or with no review, archives unchanged. 17 production lines.
  - Reviewed by pr-agent at `9aba0631`: two findings, one declined with the help output, one applied with the scan output (`active specs: 11, with a waiver: 0`). CodeRabbit did not review (free limit reached). The PR author did the triage.

## Filed (found while working)

- #2953: `dotf spec archive --help` does not yet say a waiver never overrides a signed FAIL. The file, `cmd/spec.go`, was outside this run's write paths.
- #2954: pr-agent reviews a truncated PR body (above).

## Scanned, not eligible (one question each)

- #1906, #1623: who owns the Gitea forge backend?
- #1511: which exit, A or B? The fix writes a reviewer artifact, and `cmd/spec.go` is out of scope.
- #2773, #1973, #886, #960: each changes a contract or a policy. Which one do you want specced first?
- #1473: a shell-hook change. Should it move to `dotf`?
- #1875: prreview. Which session owns it?
- #2778: `review_launch.go`. Which discoverability option, A or B?
- #2803: another session's PR is already open. Leave it with them?

## Evidence index

| Item | Command | Exit / result |
|---|---|---|
| #2920 | `go test ./internal/spec/ ./internal/cmd/ ./internal/doctor/` | ok |
| #2920 | live audit on `042954df`: 0.77.0 vs dev | exit 1 (FAIL GUARD-003) vs exit 0 (2 WARN) |
| #2920 | test-windows on main, run `39755955995` (`acfcbce2`) | "Spec issue-state audit" success |
| #2937 | `go test ./internal/worktree/ ./internal/lane/ ./internal/cmd/` | ok |
| #2937 | mutations: reason dropped; merged read without its error | tests fail, as intended |
| #2937 | live list and sweep, repo with no origin | reason in the table, the JSON and the sweep; worktree kept |
| #2937 | live list, dotfiles repo | 44 answered, 8 merged, the same as 0.77.0 |
| #2952 | new tests before the fix | 3 fail (FAIL archived, both slots; bypass record said "overrode: nothing") |
| #2952 | `go test ./internal/spec/ ./internal/cmd/` | ok |
| #2952 | CI on `9aba0631` | 24 success, 3 skipped |
| all | `go vet ./...`, `GOOS=windows go vet ./...`, golangci-lint (pin) | clean, 0 issues |
| all | `dotf pr triage-queue --pr N` before each land | exit 0 |

## Budget spent

3 items against the envelope's 4. No dollar figure exists for this run.

## Host-suspend line

Covered. `caffeinate -i -t 43200` was held from 07:40Z.

## Next action

Glance at the two skip-sdd calls (#2920, #2937), then take #2954's one-line `.pr_agent.toml` change.
