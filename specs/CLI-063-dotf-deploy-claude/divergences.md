---
id: "CLI-063-dotf-deploy-claude"
type: spec
status: implementing
created: "2026-10-04"
issue: "mlorentedev/dotfiles#1339"
tags: [spec, divergences]
template_version: "1.0"
---

# Twin divergences — Claude plugin sync and the `.claude.json` snapshot guard

**Method.** A static read of `setup-linux.sh` (`snapshot_claude_json`,
`restore_claude_json_if_truncated`, the plugin loop) and `setup-windows.ps1`
(`Backup-AndRestoreClaudeJson`, the `$plugins` loop) at `origin/main` `e4bedb26`. `pwsh` is not
installed on this machine, so the `.ps1` column comes from reading the script, not from running it.
Linux is the reference unless the row says otherwise.

## Divergences (increment 1)

| # | Behaviour | `.sh` | `.ps1` | Go (`dotf deploy`) |
|---|---|---|---|---|
| 1 | Plugin counted as added | Only when `claude plugin install` exits 0 | On every attempt, whether or not the install succeeded (#1491) | Only on success. A failed install is named on its own line and fails the run. **Recorded, not fixed in the `.ps1`**: the cutover deletes that loop. |
| 2 | Failed install visibility | Silent; the summary just shows a lower count | Silent | Loud: `failed claude-plugins <id>`, and a non-zero exit, which each setup already turns into a warning |
| 3 | `claude plugin list` fails | Read as an empty list (`\|\| true`), so every plugin is reinstalled | Same | Returned as an error before any install. A broken CLI is not evidence that nothing is installed. |
| 4 | "Lost more than half" | `new < snapshot/2` in integer arithmetic, so an odd-sized snapshot cut to exactly `floor(snapshot/2)` is not restored | `new < snapshot/2` in floating point (exact half) | `2*new < snapshot`, the exact half, matching the `.ps1` |
| 5 | Snapshot storage | A tempfile from `mktemp`, removed after the call | A tempfile from `GetTempFileName`, removed in `finally` | Held in memory. Nothing is left behind if the process is killed mid-call. |
| 6 | Restore write | `cp -f` over the file | `Copy-Item -Force` over the file | Rewritten in place, so the file keeps its own mode and ACL, as both twins do |
| 7 | Threshold source | Literal `10240` | Literal `10240` | `claude_json_min_bytes` from `session-start-config.json`, the value the session-start canary reads, falling back to `mem.ClaudeJSONMinBytes` |

## Not a divergence

- **Which calls are guarded.** All three paths wrap both the `plugin list` pre-fetch and every
  `plugin install` (BUG-004, BUG-011).
- **`.claude.json` location.** All three use `<home>/.claude/.claude.json`, and none of them honors
  `CLAUDE_CONFIG_DIR` there. The Linux twin reads `CLAUDE_CONFIG_DIR` only for the claude-mem
  cleanup, which is out of this spec's scope.
