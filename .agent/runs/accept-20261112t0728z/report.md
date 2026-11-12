# Run report: accept-20261112t0728z

**First line for the owner:** Windows acceptance of v0.80.0 is blocked on PLAT-001c's `vcredist` catalog entry. A clean Arm64 guest cannot run mise (lesson 561). The entry is unblocked now that 0.80.0 is pinned, and it belongs with the #2438 pins PR (feat-herdr-explorer-binding has been told).

- Armed 07:28Z, ended about 11:45Z. Reason: `goal-blocked`. The rest of the objective needs a catalog entry this run does not own, and its paths are outside write_paths.
- A session pause kept the Windows guest busy from 07:30Z to 11:34Z, past my slot and over feat-herdr-explorer-binding's 08:00–10:30Z slot. I apologised and handed the guest back.

## Items
| Item | Result |
|---|---|
| #3274 | closed: all six fleet PRs merged; the leftover `pr-agent.yml` comment moved to #1836 (owner, `.github/`) |
| #3363 | open, Windows only: macOS (`mise ls` clean) and from-zero Linux (pyyaml via postinstall, doctor Python ok) proven |
| #3367 | open, Windows only: Linux cache-gc first run plus not-due on the second, no-Docker skip; macOS plan, doctor Disk space, launchd background |
| #3385 | filed: the declared checkout was detached twice by an unknown actor, so self-deploy skipped; options A/B/C for the owner |

## Evidence
- `dotf vm linux --ref main`: converged, the second run changed nothing (`0 changed, 15 ok, 1 opaque, 7 skipped, 0 failed`).
- `dotf vm windows --ref main`: tools step FAIL after 45m25s, `mise-v2026.9.13-windows-arm64.exe does not run on windows/arm64`.
- v0.80.0 darwin-arm64 asset: checksum OK; `converge --plan --only cache-gc` reports not due; doctor `[Disk space]` ok.
- msi: unreachable over SSH (timeout), not measured.

## Housekeeping
Removed the `docs-adr032-credit-premise` worktree and hive's `fix-402-pool-refusal` worktree, both merged.
