# Run cli026-20261102T0551Z

Status: armed 2026-11-02T05:51Z, expires 17:51Z. The verdict is written here at disarm.

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
