---
tags: [spec, verification, claude-json, truncate-guard]
created: "2026-05-20"
---

# Verification - BUG-011-mcp-loop-claude-json-guard

## Evidence (per acceptance criterion)

- [x] **Linux helpers BEFORE MCP loop**: `snapshot_claude_json` defined at `setup-linux.sh:558` (relocated from line 606); first `claude mcp get` at `setup-linux.sh:619`. Lexical order verified via `grep -n` comparison.
- [x] **Linux MCP loop wrap**: `_snap=$(snapshot_claude_json)` at `setup-linux.sh:617`, before `claude mcp get` (l.619) and `claude mcp add` (l.628); `restore_claude_json_if_truncated "$_snap"` at l.622 (idempotence path) and l.635 (add path).
- [x] **Linux `claude plugin list` wrap**: `_snap=$(snapshot_claude_json)` at `setup-linux.sh:654` before `claude plugin list` at l.655; restore at l.656.
- [x] **Windows MCP loop wrap**: `Backup-AndRestoreClaudeJson -Action { ... }` wraps `claude mcp get` (`setup-windows.ps1:379-381`) and `claude mcp add` (`setup-windows.ps1:391-393`).
- [x] **Windows `claude plugin list` wrap**: `$installedPlugins = Backup-AndRestoreClaudeJson -Action { try { (& claude plugin list 2>$null) -join "\`n" } catch { "" } }` at `setup-windows.ps1:428-430`.
- [x] **Bats parity assertions**: 5 new asserts added to `tests/setup-linux.bats` (`# --- BUG-011: ...` block) + 3 new asserts to `tests/setup-windows.bats`. Manual grep-based equivalent of every assert verified PASS post-implementation.

## Test status

- **Manual grep verification (cross-OS, post-implementation)**:
  ```
  === Linux ===
  helpers before MCP loop: PASS
  mcp add snapshot: PASS
  mcp add restore: PASS
  plugin list snapshot: PASS
  plugin list restore: PASS
  plugin install snapshot (BUG-004 preserved): PASS
  === Windows ===
  mcp add wrap: PASS
  mcp get wrap: PASS
  plugin list wrap: PASS
  plugin install wrap (BUG-004 preserved): PASS
  ```
- **Bash syntax** (`bash -n setup-linux.sh`): OK.
- **PowerShell parse** (`[System.Management.Automation.Language.Parser]::ParseFile('setup-windows.ps1', ...)`): no parse errors (AST emitted, error array empty).
- **Empirical `$LASTEXITCODE` + stdout + closure propagation test** (isolated repro, `cmd /c exit N` as CLI stub through a `Backup-AndRestoreClaudeJson`-shaped wrapper):
  ```
  Case A (cmd /c exit 0): LASTEXITCODE after wrapper = 0
  Case B (cmd /c exit 5): LASTEXITCODE after wrapper = 5
  Case C (capture stdout + exit 7): captured='simulated-stdout', LASTEXITCODE = 7
  Case D (closure over $it): results = saw:alpha,saw:beta,saw:gamma
  VERDICT: PASS - wrapper preserves LASTEXITCODE, stdout, and closure semantics
  ```
  Confirms the Windows refactor is semantically safe: `$LASTEXITCODE` survives `& $Action`, captured stdout works (the `$mcpErr = Backup-AndRestoreClaudeJson { ... 2>&1 }` pattern), and `foreach` iteration variables are accessible inside the scriptblock (the `$srv.name` / `$srv.args` closure).
- **Full bats suite**: pending — `bats` not on local PATH; CI Docker integration job runs the full suite at PR open. Manual grep-based assertion checks (the same patterns the bats greps execute) all pass.
- **End-to-end empirical confirmation**: pending — would require running `setup-windows.ps1` on a machine with healthy ~75 KB `~/.claude/.claude.json`. NOT executed locally because that's the exact path the bug fires on; testing it on the user's live machine would risk the very re-login this PR prevents. Validation will come naturally from the next setup run on a clean target.

## Decisions made during implementation

- **Per-call wrap, not per-loop** (user Q1): chose finer granularity. Cost: ~9-10 extra snapshots per setup. Benefit: legitimate `mcp add` outputs (small file-size growth) never trigger spurious restore — the >50% shrink condition only fires inside its own iteration, against its own snapshot.
- **Include `claude plugin list`** (user Q2): wrapped the read-only pre-fetch too because BUG-004's own comment block warns "every `claude plugin install` writes to `.claude.json`" — empirically any CLI invocation goes through the same serializer.
- **Windows: separate scriptblock wraps per CLI call**, NOT one wrap around the foreach body. Reason: the iteration body contains `continue` statements; `continue` inside a scriptblock invoked via `& $Action` does NOT continue the OUTER foreach. Keeping each `claude <cmd>` in its own narrowly-scoped scriptblock preserves control flow.
- **`$LASTEXITCODE` propagation across scriptblock**: relied on PS's automatic-variable behavior — external commands update the global `$LASTEXITCODE` regardless of invocation scope. If empirical evidence shows this fails, fall back to capturing exit code explicitly inside the scriptblock and returning it.
- **Bats assertion grep window widened to `-B15`** (from initial `-B10`): comment headers + the idempotence-skip path push the snapshot ~11 lines above `claude mcp add`. The assertion's intent is "wrap is present", not "wrap is within 10 lines" — 15 is empirically generous, still tight enough to catch a missing wrap.

## Promotion candidates

- [x] **Lesson for `dotfiles/90-lessons.md`**: yes — "when guarding one CLI call site of a vulnerable upstream API, audit ALL call sites of the same CLI in the same PR". This is a sibling/refinement of the BUG-006 "Incident → guard pattern" lesson. Direct evidence: BUG-004 wrapped one call site (plugin install), BUG-011 had to wrap the remaining four (mcp get, mcp add, plugin list × 2 platforms).
- [ ] **ADR-worthy decision**: no — defense-in-depth wrap is a localized fix, not an architectural shift.
- [ ] **New pattern candidate for `00_meta/patterns/`**: no — already covered by `fix-small-debt.md` and the existing "Incident → guard" lesson; not a >1-project recurrence yet.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived` (post-merge).
- [ ] Folder moved: `specs/BUG-011-mcp-loop-claude-json-guard/` -> `specs/archive/BUG-011-mcp-loop-claude-json-guard/` (post-merge).
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link (post-merge).
- [ ] Lesson appended to `dotfiles/90-lessons.md` (post-merge, see Promotion candidates).
