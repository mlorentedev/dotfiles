# Run bugs-20261102t0548z

**Verdict:** ended `queue-empty` at 2026-11-02T09:31Z after 2 iterations / 3h40m: 2 done, 0 parked, 2 merged, 0 unmerged. Nothing irreversible beyond the two squash merges below.

## Waiting on you
- Merged under the grant: #3057 (`a57d175`, closes #2784) and #3074 (`e65228e`, refs #2896).
- #2896 stays open for one line in the `pr-review-triage` skill pointing at `triage.standing`: a vault edit plus recompile; `harness/**` was frozen for this run.
- #3007 needs your A/B decision (base-aware hygiene with strict/merge-queue, or merge-time lesson numbering). Not attempted.
- Release #3025 (0.79.0) is still open and yours.

## Parked
None.

## Skipped (not attempted)
| Item | Reason |
|---|---|
| #3048 #3049 #3033 | claimed by run night-20261102T0555Z (feat-pr-declare-unreviewed) |
| #2957 | in run plat005's paths (cli/internal/converge) |
| #3050 | claimed by feat-worktree-one-root |
| #3007 | needs an owner design choice |
| #2851 | depended on CLI-026 (another session); that session opened #3070 for it |

## Evidence index
| Item | Command | Exit |
|---|---|---|
| #2784 | `go test ./internal/cmd -run TestForge -count=1`; guard mutated -> new test FAILs | 0 / 1 (mutation) |
| #2784 | `golangci-lint run ./...` and `GOOS=windows golangci-lint run ./...` | 0 issues each |
| #2784 | CI on `a57d175`, `dotf pr land 3057` | green, `[MERGED]` |
| #2896 | new test fails before fix (`reason="github-actions reviewed, never triaged"`), passes after | 1 / 0 |
| #2896 | `bats -f "every writer of the triage record" tests/pr-agent-config.bats` | 0 |
| #2896 | CI on `e65228e`, `dotf pr land 3074` | green, `[MERGED]` |

Known local-only failure: `TestEmbeddedTemplatesMatchVault` (internal/spec, internal/vault) from vault template drift, tracked in #3050; CI has no vault.

## Budget
2 of 10 iterations, 2 of 6 items, 3h40m of 12h. No dollar figure exists for this run.

## Host
Wake-lock `caffeinate -ims -t 43500` held on AC power from 05:50Z. A usage-limit pause stopped the session between about 06:10Z and 07:11Z; the run resumed on its own.

## Knowledge
Lesson 536 (`docs/lessons/lesson-536-a-default-read-from-the-cwd-follows-the-shell-not-the-change.md`) via #3057.

## Next action
Decide #3007 (A or B), and merge #3025.
