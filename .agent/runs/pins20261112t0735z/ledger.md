# Run pins20261112t0735z ledger

Armed 2026-11-12T07:35Z by manu, expires 19:35Z. merge: standing-grant, except PR B (never merged by the run: it changes what every machine installs and touches ci.yml). The owner authorised the wider write_paths (tests, ADR-036, doctor checks, ci.yml).
Advisor: the native advisor answered before arming (rebase and run the suite first, no `--dotf`, PR B owner-landed, bats guard `last=` stays 0.79.0 because 0.80.0 carries the table).
Coordination: feat-vm-linux-exec-3c (no guest, no mise.go tonight); feat-worktree-one-root-3e (v0.80.0 acceptance, has the guest until ~08:45Z, no overlap on my files); fix-issue-491 (ts-bridge: Windows from 15:00Z, Linux guest before). My guest slot: ~08:45Z-10:30Z, and not before 15:00Z after it.

| # | item | state | note |
|---|---|---|---|
| 1 | #2438 claim | picked up 07:5xZ | assigned, board In Progress, comment posted |
| 1 | feat/mise-npm-pins-v2 5fb56675 | pushed | rebased on main b00a4642 (0.80.0); a force-push to the old name was denied, so the work is under a new name and the old branch is left as it was |
| 1 | PR B content | committed | four pins + four catalog deletions + vcredist + doctor fallback and `pinSource` label + ADR-036 amendment + guards: versions-conf test 11, copilot-config test 11, ci-npm-cli-cache (pi only, key without packages.json) + ci.yml cache step + TestCheckVersionMatch fixture |
| 1 | pre-guest verification | green | `go test ./...`, `GOOS=windows go vet ./...`, golangci-lint 0 issues, 45 bats ok; mutation-checked: a bw path back in the cache, and a copilot catalog entry |
| 2 | vcredist before mise (#3396) | draft PR #3399 | parked `scope-violation`: tests/packages-json.bats test 3 needs one jq clause (`or (.source.winget_arch \| type == "object")`), a file outside write_paths; the clause is in the PR body and on #3396 for the owner. Go order test passes |
| 2 | guest run on feat/mise-npm-pins | running | started 11:34Z, still in the first converge at 11:43Z |
| 2 | full bats sweep | running | 396 of ~1200 ok at 11:42Z; packages-json test 3 expected red on PR B (same cause as #3399) |
| 2 | guest ref | verified | `dotf vm windows --ref` runs `git archive` in the cwd's repo (cli/internal/cmd/vm.go:174), so the run measures the local `feat/mise-npm-pins` 5fb56675 (0.80.0), not the remote 0.79.0 |
| 2 | full bats sweep | done | 1805 ok / 11 not ok. 2 pre-existing on main-based tree (harness-doctrine-records "committed records match", roster-consistency-real "live vault's roster": both read the real vault). 1 known: packages-json 925, same cause as #3399. 8 are PR B expectation updates in tests outside write_paths: opencode.bats (2), setup-linux.bats (3), setup-windows.bats (3); each assertion read, all expect opencode/copilot/yarn as catalog npm tools. setup-linux 1242/1243 fail loudly ("packages.json owns no npm tool, so the check would be vacuous"), not vacuously: they need a different fixture or a skip when the catalog has no npm tool |
| 2 | PR B | to open as draft, parked `scope-violation` | cannot be green under this envelope; the owner's steps (the 8 test edits plus the #3399 clause) go in the PR body. The objective was "prepared and evidenced", not landed |
| 2 | guest run, 0.80.0, ref feat/mise-npm-pins | failed: tools, one tool | 11:34Z to 12:25Z (46 min tools step). bw, copilot, opencode and yarn pass the running-at-pin check; `bats` does not (#2346). Second converge not reached, so `changed=0` is not shown. Log stayed in the guest. Guest stopped 12:25Z |
| 2 | PR B | draft #3403, parked `scope-violation` | Refs #2438; owner steps (8 tests + #3399 clause) listed in the body. Evidence comments posted on #3329 and #3328 |
