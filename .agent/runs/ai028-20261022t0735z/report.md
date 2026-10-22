# Run ai028-20261022t0735z

**ended queue-empty after 2 items / about 1.7h: 1 landed, 0 parked by this run, 0 unmerged.** Waiting on a human: the AC9 live run (msi or the work-lab box), #574's closure, and #2932's A/B. Nothing irreversible is waiting.

Envelope: `.agent/unattended.yml` at `0f0cbb42` (armed 07:35Z, expires 17:35Z).

## Waiting on you
- **Landed (irreversible):** #2922 (`b491851e`). hive's Windows bootstrap passes `--no-build-package cryptography` (`self_install.no_build`), and the hive pin moves to 4.4.2. AI-028 AC2 passes on the Arm64 guest.
- **Closed:** #2715. Its registrar went with #2876 (`git grep` evidence on the issue).
- **Filed:** #2932. The guest clock runs 2h behind, so certificates under 2 hours old fail. A: `guest-set-time` over QGA (recommended). B: `RealTimeIsUniversal` in autounattend.
- **#574:** comment recommending closure as superseded by hive ADR-019 and #2901. The owner decides.
- **#791 (AI-028)** stays Blocked on one step, AC9's live changed=0, which needs a machine holding the old trigger. Then `dotf spec review AI-028-hive-install-model-migration`.

## Parked
None by this run. Out of the envelope's scope (`setup-windows.ps1` frozen, `cli/internal/vm` not in write_paths), so not attempted:
- #2258: delete setup's duplicate Startup fallback. Measured on the guest and commented on the issue.
- #2932: the VM clock.
- #1297: a design decision, waiting on the owner since #791.

## Evidence index
| Item | Command | Exit |
|---|---|---|
| AC2 guest run, attempt 1 (07:10Z) | `dotf vm windows --ref 68ed76db --dotf <arm64 build>` | 1: npm `certificate is not yet valid` (#2932); hive rows identical |
| AC2 guest run, attempt 2 (07:52Z) | same | 0: `==> converged, and the second run changed nothing`; excerpt below |
| Windows Arm64 resolution | `uv pip compile --python-platform aarch64-pc-windows-msvc --only-binary cryptography` | 0: cryptography 46.0.3 (x64 stays 50.0.2) |
| #2922 unit and lint | `go test ./internal/tools/ ./internal/cmd/`, `go vet ./...`, `GOOS=windows go vet ./...`, `golangci-lint run ./...` | 0, 0 issues |
| #2922 bats | the bats files that reference packages.json (minus verify-setup) | 0: 273 tests |
| Mutations | flag removed from `selfInstall`; `no_build` removed from hive's entry | each fails its test |
| CI on `bf56b02e` | check-runs | 27 green |
| Triage | `dotf pr triage-queue --pr 2922` | 0 |
| Land | `dotf pr land 2922` | 0: `[MERGED] #2922 at bf56b02` |

## Budget
2 items, about 1.7h of wall clock. No dollar figure exists for this run.

## Host
`caffeinate -i -s` armed at 07:35Z and still alive at exit: the window was covered.

## Next action
Owner: power msi on (or use the work-lab box) and run `dotf converge --only hive-upgrade-retire` twice with a dotf release carrying #2876. That ticks AC9, then run `dotf spec review AI-028-hive-install-model-migration`.

## Evidence: AC2 guest run on Windows Arm64

Command: `dotf vm windows --ref 68ed76db --dotf <GOOS=windows GOARCH=arm64 build of 68ed76db>`, started 07:52Z, exit 0.
Matching lines from the 1909-line log (`grep -nE`; the number is the log line):

```
1:==> boot a throwaway overlay of clean-base (QEMU's output in /Users/manu/.local/state/dotfiles/vm/windows/qemu.log)
2:==> copy the ref's archive into C:\Users\Public\dotf
3:==> copy the dotf build /private/tmp/claude-501/-Users-manu-Projects-worktrees-dotfiles-chore-spec-triage-ai-028/f0562f64-49f0-47f0-8925-429c7bbaad46/scratchpad/dotf.exe in
4:==> first converge: install.ps1 from the ref
717:Downloading cryptography (2.8MiB)
725:hive self-upgrade: switched none -> 4.4.2.
726:hive 4.4.2 installed via hive self-upgrade 4.4.2 (hive-vault)
807:  [ OK ]   hive-upgrade-retire  664ms  no unattended hive upgrade is installed
897:hive 4.4.2 already installed; skipping
943:[SUCCESS] Registered MCP hive
954:1 changed, 0 ok, 0 opaque, 0 skipped, 0 failed
997:0 changed, 1 ok, 0 opaque, 0 skipped, 0 failed
1001:0 changed, 1 ok, 0 opaque, 0 skipped, 0 failed
1153:  (15 checks, all ok)
1156:  (1 checks, all ok)
1162:  (2 checks, all ok)
1187:  (2 checks, all ok)
1216:  (2 checks, all ok)
1234:  (1 checks, all ok)
1243:  (1 checks, all ok)
1266:  (1 checks, all ok)
1271:[hive daemon (the backend of every agent's `hive client`)]
1272:  (1 checks, all ok)
1283:  (1 checks, all ok)
1289:  (4 checks, all ok)
1292:  (2 checks, all ok)
1295:  (1 checks, all ok)
1298:  (2 checks, all ok)
1308:  (1 checks, all ok)
1337:  (1 checks, all ok)
1340:  (1 checks, all ok)
1343:  (1 checks, all ok)
1346:  (1 checks, all ok)
1375:  (4 checks, all ok)
1416:10 changed, 3 ok, 1 opaque, 6 skipped, 0 failed
1426:==> second converge: dotf converge
1444:  [ OK ]   hive-upgrade-retire  832ms  no unattended hive upgrade is installed
1484:hive 4.4.2 already installed; skipping
1540:0 changed, 1 ok, 0 opaque, 0 skipped, 0 failed
1581:0 changed, 1 ok, 0 opaque, 0 skipped, 0 failed
1585:0 changed, 1 ok, 0 opaque, 0 skipped, 0 failed
1636:  (15 checks, all ok)
1639:  (1 checks, all ok)
1642:  (2 checks, all ok)
1645:  (2 checks, all ok)
1670:  (2 checks, all ok)
1699:  (2 checks, all ok)
1717:  (1 checks, all ok)
1726:  (1 checks, all ok)
1749:  (1 checks, all ok)
1754:[hive daemon (the backend of every agent's `hive client`)]
1755:  (1 checks, all ok)
1766:  (1 checks, all ok)
1772:  (4 checks, all ok)
1775:  (2 checks, all ok)
1778:  (1 checks, all ok)
1781:  (2 checks, all ok)
1791:  (1 checks, all ok)
1820:  (1 checks, all ok)
1823:  (1 checks, all ok)
1826:  (1 checks, all ok)
1829:  (1 checks, all ok)
1858:  (4 checks, all ok)
1899:0 changed, 13 ok, 1 opaque, 5 skipped, 0 failed
1909:==> converged, and the second run changed nothing
```
