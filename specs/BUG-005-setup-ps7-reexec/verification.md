---
tags: [spec, verification, bug, windows, powershell]
created: "2026-05-19"
---

# Verification - BUG-005-setup-ps7-reexec

## Evidence

- [x] **PS 5.1 + pwsh installed → re-execs under pwsh, no AsHashtable warning** → empirical 2026-05-19 (Windows 11 22631.6931, pwsh 7.6.1): `PowerShell -NoProfile -ExecutionPolicy Bypass -File setup-windows.ps1` first line of output: `[INFO] Windows PowerShell 5.1.22621.6931 detected; re-executing under pwsh (C:\Users\mlorente\AppData\Local\Microsoft\WindowsApps\pwsh.exe) for full feature compatibility (BUG-005)`. Setup proceeded normally; merged `~/.claude/settings.json` (1687 bytes, mtime updated, 14 plugins + preserved user customizations); NO `[WARNING] Claude settings template is not valid JSON ... AsHashtable` line in output.
- [x] **PS 5.1 + pwsh NOT installed → fail loud + exit 1** → not directly testable (pwsh IS installed on the dev machine). Verified structurally by reading the preamble: the `else` branch unconditionally writes `[ERROR]` lines (`Get-Command pwsh -ErrorAction SilentlyContinue` returns `$null`, falsy) and `exit 1`. Bats assert `setup-windows.ps1 fails loud with winget hint when pwsh missing (BUG-005)` greps for both `winget install Microsoft.PowerShell` and `exit 1`, locking the branch's contract.
- [x] **pwsh 7+ direct → preamble no-op, existing behavior** → empirical 2026-05-19: `pwsh -NoProfile -File setup-windows.ps1` (used during BUG-004 smoke earlier same day) emitted ZERO `[INFO] Windows PowerShell ... detected` lines. The `if ($PSVersionTable.PSVersion.Major -lt 7)` predicate evaluated false, the entire preamble block was skipped, and the script ran identically to its pre-BUG-005 state.
- [x] **bats grep asserts**: 5 new asserts in `tests/setup-windows.bats` covering version check, re-exec branch, fail-loud branch, BUG-005 cite, and negative parity (Linux `setup-linux.sh` clean of `PSVersion`). All 5 RED before implementation, all 5 GREEN after — verified via grep-by-grep emulation; full bats run in CI.
- [x] **PSScriptAnalyzer clean on changes**: 4 new `PSAvoidUsingWriteHost` warnings introduced by the preamble's `Write-Host` calls. Style-consistent with the rest of the script (`Write-Info`/`Write-Success`/`Write-Warn`/`Write-Err` are all `Write-Host` wrappers, see lines 45-48 of the original file). The preamble cannot use the wrapper functions because the wrappers are defined below the preamble's runtime position. Zero `Error` severity findings. Net change: +4 Write-Host warnings, all matching existing convention; no new rule classes triggered.
- [x] **No Linux changes**: `git diff main..HEAD --stat -- setup-linux.sh` returns empty. Verified `setup-linux.sh` byte-identical to main; negative-parity bats assert locks this in.

## Test status

- **Bats grep emulation (local, no bats binary)**: 5/5 GREEN after implementation. RED → GREEN transitions logged in session transcript.
- **Manual smoke (Windows admin machine, 2026-05-19)**:
  - Under Windows PowerShell 5.1.22621.6931: re-exec line appears as first output, setup proceeds under pwsh, settings.json merged (1687 bytes), no AsHashtable warning. **PASS.**
  - Under pwsh 7.6.1 directly: preamble no-op, no `[INFO] ... detected` lines. **PASS.**
  - Without pwsh installed: not directly testable; structural verification only.
- **PSScriptAnalyzer**: 4 new Write-Host warnings, style-consistent with existing script. Zero Errors. Zero new rule classes.
- **No regressions**: existing tests for MCP self-heal, claude-mem-heal, copilot v2 detection, settings.json merge — all still match their structural greps (Linux side untouched; Windows side preamble inserted ABOVE all existing logic so no relocations).

## Decisions made during implementation

- **Auto re-exec instead of in-place PS 5.1 compatibility**: Two options on the table for fixing the AsHashtable issue — (a) backfill PS 5.1 compatibility into `Merge-ClaudeSettings` via a `PSCustomObject → hashtable` recursive helper, or (b) require PS 7+ and re-exec. Chose (b) because: (1) the helper-rewrite would add ~30 lines of subtle recursive logic vs ~15 lines of preamble, (2) PS 5.1 has many other latent shortcomings that will bite future helpers (e.g. ternary operator, null-conditional `?.`), and locking the script to PS 7+ at the front door makes downstream code free to assume modern features, (3) Microsoft is sunsetting Windows PowerShell 5.1 in favor of pwsh as the supported PowerShell, so the trend is in our favor.
- **Write-Host instead of Write-Info wrapper**: The wrapper functions are defined LATER in the script (around line 76 after this PR). The preamble must run before any function definition because its purpose is to short-circuit execution. Using `Write-Host` directly is structurally required; the resulting PSScriptAnalyzer warnings are consistent with the existing style (the wrappers themselves use Write-Host).
- **`@args` instead of `$PSBoundParameters`**: current `param()` is empty (no named params), so positional `@args` suffices. If named params get added later, the inline comment in the preamble specifies that `$PSBoundParameters` is the correct forwarding mechanism. Documented.
- **Don't auto-install pwsh**: leaves the install action explicit to the user. Auto-install would (a) require admin rights mid-script, (b) require winget present (not guaranteed on older Windows builds), (c) make the failure mode "the script tried to install something" instead of "the script told me what to install". The actionable error is the better UX.

## Promotion candidates

- [ ] **Lesson** → no. The BUG-005 fix is mechanical; no non-obvious insight that wouldn't be obvious from reading the code.
- [ ] **ADR** → no. Pwsh-7-required for setup-windows.ps1 could be ADR-worthy IF the team grows. For a single-user dotfiles repo, the inline comment in the preamble suffices.
- [ ] **New pattern** → no. "Auto-reexec under newer runtime when current is incompatible" is a known idiom (shebangs do this on Unix); not novel enough to promote.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-005-setup-ps7-reexec/` → `specs/archive/BUG-005-setup-ps7-reexec/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (none)
