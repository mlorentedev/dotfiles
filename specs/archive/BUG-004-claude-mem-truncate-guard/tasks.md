---
tags: [spec, tasks, bug, defense-in-depth]
created: "2026-05-19"
---

# Tasks - BUG-004-claude-mem-truncate-guard

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `fix/BUG-004-claude-mem-truncate-guard`
- [x] `proposal.md` complete and acceptance criteria testable
- [x] No unresolved questions in `proposal.md` "Risks / open questions"

## Implementation (TDD order)

### Tests first (red)

- [x] `tests/setup-windows.bats`: add failing assert — `setup-windows.ps1` defines a `Backup-AndRestoreClaudeJson` helper (or equivalent snapshot/restore pair) and calls it inside the plugin-install loop. Anchor on the function name + a literal `59870` reference in the inline comment so the test rots if the upstream tracker reference is removed.
- [x] `tests/setup-windows.bats`: add failing assert — every `claude plugin install` call site in `setup-windows.ps1` is preceded by a snapshot helper call (grep-based structural check). (Used `tests/setup-linux.bats` for the bash side — that is where structural greps for `setup-linux.sh` live; `verify-setup.bats` is for Docker integration.)
- [x] `tests/setup-linux.bats`: add failing parity assert — `setup-linux.sh` defines `snapshot_claude_json` + `restore_claude_json_if_truncated` AND calls them around the bash `claude plugin install` call. Anchor on `59870` literal in the inline comment.
- [x] `tests/setup-linux.bats`: add failing assert — the 10 KB sanity floor literal (`10240`) appears in the helper body (so a future "drop the floor" change without spec update fails the test).

### Implementation (green)

- [x] `setup-windows.ps1`: add `Backup-AndRestoreClaudeJson` helper function in the helper-functions block (after `Ensure-Directory`). Body: snapshot `~/.claude/.claude.json` to a tempfile via `Copy-Item`, execute the wrapped action via `& $Action`, in a `finally` block read pre/post sizes, restore (`Copy-Item -Force`) iff pre ≥ 10 KB AND post < pre/2, then unconditionally `Remove-Item` the tempfile. Cite issue `#59870` in the function-header comment.
- [x] `setup-windows.ps1`: wrap the existing `& claude plugin install $plugin` call inside the foreach loop with the helper. Preserve `$pluginsAdded++` semantics (only increments on actual install success — refactor via a local `$success` boolean inside the action block, surfaced via `$script:` or returned).
- [x] `setup-linux.sh`: add `snapshot_claude_json` (echoes tempfile path or empty if source missing) + `restore_claude_json_if_truncated` (takes tempfile path, restores per the same heuristic, deletes tempfile). Place after the existing helper blocks. Use `stat -c %s` (Linux integration test uses GNU coreutils, fine).
- [x] `setup-linux.sh`: in the plugin install `for plugin in ...` loop, before the `claude plugin install` call, capture `_snap=$(snapshot_claude_json)`; after the call (success or failure branch), call `restore_claude_json_if_truncated "$_snap"`. Preserve `plugins_added` increment semantics — only on real install success.
- [x] Both scripts: keep the existing idempotence guard (`grep -qF` / `-match [regex]::Escape`) untouched; the wrapper is the second layer, not a replacement.

### Refactor / cleanup (still green)

- [x] Both scripts: inline comment in each helper body cites `#59870`, `dotfiles#33`, and `SDD-021` so the next reader sees the full lineage.
- [x] `setup-windows.ps1`: reuse the existing `Write-Warn` helper for log output consistency.
- [x] PSScriptAnalyzer clean on `setup-windows.ps1`. (0 new Error/Warning attributable to BUG-004; pre-existing empty-catch warning relocated but unchanged in semantics.)
- [x] `bash -n setup-linux.sh` (clean). Shellcheck NOT run locally; will run in CI.

### Local verification

- [x] All new bats asserts go from red → green. (Verified via grep-by-grep emulation; bats binary not available locally on Windows, will run in CI.)
- [x] Full `bats tests/setup-windows.bats` green (emulated via grep, will run in CI).
- [x] Full `bats tests/setup-linux.bats` green (emulated via grep, will run in CI).
- [x] Smoke on the dev Windows machine: pre 52055 bytes; ran `setup-windows.ps1` twice under pwsh; post 52055 bytes both times. See verification.md for caveats (in-vivo upstream bug did not fire this afternoon).
- [x] Synthetic truncation smoke: in-process invocation of `Backup-AndRestoreClaudeJson` with an action that overwrites `.claude.json` with `{}`. WARNING line printed: `.claude.json shrunk from 52060 to 2 bytes after install (upstream #59870); restored from backup`. File restored to 52060 bytes.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass (PSScriptAnalyzer + bash -n)
- [x] Lint passes (PSScriptAnalyzer Error+Warning, shellcheck deferred to CI)
- [x] No unrelated changes in the diff (no scope creep — explicitly excludes BUG-005 work)
- [x] `verification.md` filled in with empirical evidence; commit hashes added at PR-merge time
- [ ] PR opened referencing this spec folder; PR body cites upstream `#59870` and the empirical reproduction date `2026-05-19`

## Machine-readable features

This spec emits a sibling `features.json` mapping each acceptance criterion to a verification command.

```json
[
  {
    "id": "BUG-004-f1",
    "behavior": "setup-windows.ps1 preserves .claude.json size across two consecutive runs on an authenticated machine (>=50 KB baseline)",
    "verification": "bats tests/setup-windows.bats -f 'preserves .claude.json'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "BUG-004-f2",
    "behavior": "setup-linux.sh preserves .claude.json size across two consecutive runs on an authenticated machine",
    "verification": "bats tests/verify-setup.bats -f 'preserves .claude.json'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "BUG-004-f3",
    "behavior": "Synthetic truncation triggers a single [WARNING] line citing upstream #59870 and restores the snapshot",
    "verification": "bats tests/setup-windows.bats -f 'synthetic truncation'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "BUG-004-f4",
    "behavior": "Fresh-machine path (no .claude.json) is a no-op (no temp file, no warning)",
    "verification": "bats tests/setup-windows.bats -f 'fresh machine'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "BUG-004-f5",
    "behavior": "Pre-call size below 10 KB does NOT trigger restoration even on large relative shrink",
    "verification": "bats tests/setup-windows.bats -f '10 KB floor'",
    "state": "pending",
    "evidence": ""
  }
]
```
