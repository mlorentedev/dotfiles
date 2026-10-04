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
| 8 | Which `.claude.json` is guarded | `$HOME/.claude/.claude.json`, hardcoded | `$USERPROFILE\.claude\.claude.json`, hardcoded | `$CLAUDE_CONFIG_DIR/.claude.json`, with the directory resolved through the env contract (default `~/.claude` on both OSes). The child `claude` is run with that same `CLAUDE_CONFIG_DIR`, so the guard and the CLI agree on the file by construction. The twins are right only while the rc files export the default; Claude Code writes `~/.claude.json` when the variable is unset. Found by PR-Agent on #1992. |
| 9 | "Already installed" test | `grep -qF` for the id anywhere in the list output | `-match [regex]::Escape($plugin)` on the output, also a substring match | The id must equal a whole whitespace-separated token. A declared `lsp@m` is not "present" just because `gopls-lsp@m` is installed. |

## Divergences (increment 2)

Same method, at `origin/main` `614ae4ed`: `setup-linux.sh` lines 983-1046, `setup-windows.ps1`
lines 522-600.

| # | Behaviour | `.sh` | `.ps1` | Go (`dotf deploy`) |
|---|---|---|---|---|
| 10 | Gate on the whole loop | `claude`, `npx` and `jq` | `claude` and `npx` | `claude` only. `npx` is now `sequential-thinking`'s declared `prerequisite_binary`, the one server that needs it; `jq` was the shell's JSON reader. Without npx the other servers still register. |
| 11 | `prerequisite_command` | Run before the server's `get` | Run before the server's `get` | **Not run** (#1993). Installing a tool is not a registration, and `--upgrade` on every run is not idempotent. hive must reach the catalog before the cutover deletes the twins. |
| 12 | Missing prerequisite binary | Warn, skip, counted as skipped | Same | `skipped claude-mcp <name> (<bin> not on PATH)`. Not a failure: the server cannot run here, and that is not the deploy's fault. |
| 13 | HIVE-118 migration when `uv` is absent | The stale entry is removed, then the loop skips hive, so hive ends unregistered | Same | The prerequisite is checked first, so the stale entry stays. Neither works without `uv`; Go does not destroy what it cannot replace. |
| 14 | HIVE-118 remove fails | Silent; the loop then finds hive registered and skips it, so the stale entry stays | Same | `failed claude-mcp hive`, and the run fails |
| 14b | HIVE-118 remove succeeds, re-add fails | "Migrating" is already logged; the add failure is warned; hive ends unregistered | Same | Same end state, but `migrated` is printed only once the re-add lands, so the run says `failed claude-mcp hive` and nothing claims a replacement. Found by PR-Agent on #1994. |
| 15 | Failed `mcp add` | Warned with the CLI's output, counted, exit 0 | Same | Named on its own line and fails the run, as plugins do (row 2). The plugin step still runs. |
| 16 | Where the list is read | `$DOTFILES_DIR/mcp-servers.json`, the deploy dir copy | `$DotfilesDir\mcp-servers.json` | The repo root, like every `ai/deploy.json` source |
| 17 | Reading the file | `jq` with `// ""` defaults; an unknown field is ignored | `ConvertFrom-Json`; an unknown field is ignored | Strict: an unknown field, an empty name or args, a duplicate, or a transport other than stdio/http fails the step. A test loads the repo's own file. |

## Divergences (increment 3)

Same method, plus a measurement: the Go merge deployed onto a copy of msi's own
`~/.claude/settings.json`.

| # | Behaviour | `.sh` | `.ps1` | Go (`dotf deploy claude-settings`) |
|---|---|---|---|---|
| 18 | `permissions.deny` on an existing box | Never written; only a fresh bootstrap gets it | Same | Unioned in. msi had 0 of the template's 17 deny rules, including the secrets defenses (`dotf secrets show`, `env`, `printenv`). Same defect class as `outputStyle`, closed by the same change. |
| 19 | `attribution` | Replaced whole, so a stale subkey cannot reinstate a trailer | Same | Merged: the template's three subkeys win, a subkey only the box has survives. Accepted: the template names every subkey Claude Code documents, and the values that hide the trailer are the template's. |
| 20 | A template key the policy does not name | Silently skipped on an existing box (the allow-list) | Same | Written. The template is the allow-list now; `hooks` is kept out of it by test. |
| 21 | `$schema` | Not in the policy, so never written on an existing box | Same | Written. Harmless: an editor hint. |
| 22 | `permissions.allow` order | `unique` sorts the whole list | `Select-Object -Unique` keeps first-seen order | Box order kept, new entries appended. Membership is what Claude reads, and neither path rewrites the other's output: after the jq merge, Go reports `in sync`. |
| 23 | Destination | `$HOME/.claude/settings.json` | `$USERPROFILE\.claude\settings.json` | `{CLAUDE_CONFIG_DIR}/settings.json`, as row 8 |
| 24 | File formatting on first write | `jq` keeps key order | `ConvertTo-Json` keeps hashtable order | Keys sorted, two-space indent. One-time, cosmetic. |

## Not a divergence

- **Which calls are guarded.** All three paths wrap both the `plugin list` pre-fetch and every
  `plugin install` (BUG-004, BUG-011), and every `mcp get`, `mcp add` and `mcp remove`.
- **Args splitting.** All three split `args` on whitespace with no quoting.
- **MCP before plugins.** Both twins register servers first; so does Go.
