# Run pins20261112t0735z report

**Verdict:** run pins20261112t0735z ended `queue-empty` after 1 iteration over 5h20m (07:35Z to 12:55Z of a 12h envelope): 2 prepared, 2 parked (both `scope-violation`), 0 merged, 2 PRs unmerged. Waiting on you: the owner steps below. Nothing irreversible was done: no merge, no release, no push to main, and no remote change besides two draft PRs, three issue comments and one new issue.

## Waiting on you
1. **#3399** (draft): `vcredist` ahead of `mise` in `packages.json`. Apply one clause in `tests/packages-json.bats` test 3's jq, `or (.source.winget_arch | type == "object")`, then mark ready. Only that test fails in CI. Refs #3396.
2. **#3403** (draft, never merged by a run): the four npm CLIs move to mise. It changes what every machine installs and edits `ci.yml`, so it is yours to land. Before it can be green, nine tests in files the envelope froze need new expectations (listed in the PR body): `opencode.bats` 2, `setup-linux.bats` 3, `setup-windows.bats` 3, `verify-setup.bats` 1 (integration), `packages-json.bats` 1. It carries `vcredist` too, so whichever of #3399 and #3403 lands second rebases over one hunk.
3. **#3404** (new): `dotf vm windows --keep` leaves a failed run's converge log unreadable. Options A/B in the issue.

## Parked
- #3396 / #3399, `scope-violation`: the bats clause above. Cheapest unblock: one line in one test file.
- #2438 / #3403, `scope-violation`: the nine tests above. Cheapest unblock: widen the envelope to those four files and re-run, or edit them by hand.

## Done but unmerged
Spec AC4 evidence, ADR-036 amendment and doctor fallback are inside #3403. Evidence comments are on #3329 and #3328 (#3328 stays open: option 2 needs `converge/**`).

## Evidence index
- Guest, clean Win11 Arm64, dotf 0.80.0, ref `feat/mise-npm-pins` 5fb56675 (`git archive` of the local branch, `cli/internal/cmd/vm.go:174`): `dotf vm windows --ref feat/mise-npm-pins --keep` exit 1 after 46m13s in `tools`; only `bats` not running at its pin; bw, copilot, opencode, yarn pass. Second converge not reached, so `changed=0` is not shown. Output: the run's `pins-run1.out`/`.err`; the converge log stayed in the guest.
- `bats tests/*.bats` on the pins branch: 1805 ok / 11 not ok (2 fail on main too, 9 are PR B's open expectations, 1 of those is the #3399 cause).
- CI on #3403: `test` and `integration` red as described; `test-windows` green, setup step 400 s vs 228-309 s with the old cache.
- CI on #3399: all green except `test` (bats 925 only).
- Before the guest: `go test ./...`, `GOOS=windows go vet ./...` and `golangci-lint` 0 issues on the pins branch.

## Budget spent
1 iteration of 6, about 5h20m wall clock, 2 of 3 items. No dollar figure exists for this run.

## Host
Covered: caffeinate held the host awake, and the guest ran continuously 11:34Z to 12:25Z.

## Next action
Apply the bats clause on #3399 and mark it ready; then decide whether to widen a run to the four test files for #3403.
