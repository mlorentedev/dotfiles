# Run cli026-20261102T0551Z

**Verdict: queue-empty at 10:28Z 2026-11-02, all 3 items done.** Merged under the grant: 1 PR (#3070, closes #2851). Prepared for the owner: 2 PRs (#3066, then #3069), triaged and in draft. Parked: none.

Status: armed 2026-11-02T05:51Z, expires 17:51Z, disarmed 10:28Z.

## Items (the envelope's queue, in order)

1. **7c-records** (#495), owner-landed. The vault sources (skills and patterns) stop naming `compile-harness.sh`.
   `dotf harness refresh` carries the change into `harness/skills` and the targets (this one touches AGENTS.md).
   The schema descriptions name the Go verbs instead.
2. **7c-delete** (#495), owner-landed, stacked: its PR's base is the 7c-records branch. Its contents:
   - `git rm` the script, its bats and the parity oracles;
   - move `harness-doctrine-records.bats` and `harness-refresh-announce.bats` to `dotf harness refresh --vault`;
   - update the Go comments and docs;
   - a Go-side test for each of #2884's three cases.

   It closes #2884 and #1733. Lines that only say what the script replaced (ci.yml:333, setup-linux.sh:268) stay.
3. **#2851**, under the standing grant, but only if `tests/harness-deploy-parity.bats` still passes. If the oracle pins
   the old behaviour, it stacks on 7c-delete and becomes owner-landed.

## Rules

- Owner-landed PRs: once reviewed and triaged, run `gh pr ready --undo`. GitHub refuses to merge a draft, so none can land by mistake.
- The GitHub API budget is shared by ~25 sessions. A failed `gh` call counts as pending, never green; `dotf pr land` is the CI check.
- Commit vault edits by explicit path, or from a detached scratch worktree of the vault, straight to vault master.
- Peers: feat-plat003-close (cli/internal/spec, #3031 in `cmd/harness_refresh_test.go`: do not edit) and
  feat-pr-declare-unreviewed (worktree, forgewrite, lane, prreview, converge; holds #3033). Open PR #3040 edits
  `tests/setup-windows.bats`: rebase after it.
- Pushing the envelope: the spec-gate counts `.agent/unattended.yml` as production code (#2698). This envelope is
  kept under the threshold, and its prose lives here.

## Ledger

| time (UTC) | item | event | evidence |
|---|---|---|---|
| 05:51 | - | armed | `.agent/unattended.yml` |
| 07:12 | - | usage limit hit after arming (05:5x-07:11); wake-lock caffeinate -i -t 38400 started; the run starts now | `pmset -g`: sleep 0 |
| 07:23 | 7c-records | vault 345bb4c6 on master; PR #3066 opened (owner-landed) | harness check OK; bats 186/1 (real-vault test, fails on main too) |
| 07:34 | 7c-delete | PR #3069 opened, stacked on #3066 (owner-landed); script + 10 bats gone, 140+/4325- | go test ok (vault-template drift aside, #3050); lint 0; bats 345/346 (real-vault drift: insights, vault-doctor) |
| 07:43 | #2851 | PR #3070 opened (grant-eligible), gated on the deployed marker per advisor | go test harness/cmd ok; lint 0; bats parity+pipeline 33/33 |
| 09:15 | 7c | #3066 and #3069 reviewed (PR-Agent; CodeRabbit rate-limited), triaged, returned to draft; filed #3075 (stale insights/vault-doctor records) | review-attestation success on both; main d2c5c790 real-vault test not ok 2 on the same two files |
| 09:15 | #2851 | review findings applied: lesson 536, real evidence in body; new head re-queued for CI and review | go vet/test/lint output in PR body |
| 09:40 | #2851 | hygiene red: lesson 536 collided with one merged on main; renumbered to 537. The push gate skips re-review under 3 commits, so `/review` was requested | `lessons fmt --check` OK |
| 10:00 | #2851 | review found a real bug: a symlinked record file was removed on every run (Lstat). Fixed with a test that was red first | `bats exit=0`, 1..33, 0 not ok |
| 10:28 | #2851 | triaged, `dotf pr triage-queue --pr 3070` exit 0, landed `dotf pr land` -> c9b0020f; #2851 closed; worktree removed | `[MERGED] #3070 at 071dfa0` |

## Waiting on the owner

1. **Merge #3066** (7c-records: `harness/**`, `AGENTS.md`). It is in draft, so mark it ready first. Its mergeable state is clean.
2. **Then merge #3069** (7c-delete, closes #2884 and #1733). Once #3066 has merged, retarget it to `main` (or let GitHub retarget it) and mark it ready.
3. **Two comments still name the deleted script:** `.github/workflows/ci.yml:333` and `setup-linux.sh:268`. Both paths were frozen for this run. Correct them when you next touch those files, or as a CLI-026 closing item.
4. **#3075**: review the two vault edits to `insights` and `vault-doctor`, then refresh or revert them. Until then, the real-vault bats test is red on vault machines; CI stays green.

Verified at disarm: `git merge-tree --write-tree origin/main` exits 0 for both the #3066 and #3069 branches, so neither conflicts with today's `main`.

## Parked

None.

## Evidence index

- #3070 merged at `071dfa0` (`c9b0020f` on main); its body and the `## Review triage` comment carry the command output.
- #3066 and #3069: the `## Review triage` comments; `review-attestation` reads success on both.
- Repo-wide `dotf pr triage-queue`: exit 1. The only item is #3078 (`feat/vm-windows-cp`, another session), not this run's.

## Budget

- Items: 3 of 3 (max 3).
- Iterations: 4 (max 8).
- Consecutive failures: 0.
- No envelope widening.
- PR-Agent ran 5 passes in total, two of them requested with `/review`.
- CodeRabbit was rate-limited on every PR.

## Host

macmini on AC power with `sleep 0`, plus `caffeinate -i -t 38400`. The window was **not** covered from 05:5x to 07:11Z: the session hit its usage limit right after arming. Covered from 07:11Z to disarm.

## Next action

The owner merges #3066, then #3069. After that, the CLI-026 closing items: `verification.md`, `dotf spec review CLI-026-dotf-harness-engine`, archive.
