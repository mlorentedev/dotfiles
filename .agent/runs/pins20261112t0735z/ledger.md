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
