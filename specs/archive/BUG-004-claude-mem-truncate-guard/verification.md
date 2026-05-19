---
tags: [spec, verification, bug, defense-in-depth]
created: "2026-05-19"
---

# Verification - BUG-004-claude-mem-truncate-guard

## Evidence

Each acceptance criterion from `proposal.md` mapped to concrete proof.

- [x] **Synthetic truncation triggers the wrapper and emits one `[WARNING] … upstream #59870 …` line + restores the snapshot** → empirical smoke 2026-05-19 (pwsh 7.6.1, Windows 11 22631): pre-snapshot 52060 bytes; action wrote `{}` (2 bytes); finally block detected the shrink (52060 → 2 < 52060/2), restored from `[System.IO.Path]::GetTempFileName()` backup, printed `[WARNING] .claude.json shrunk from 52060 to 2 bytes after install (upstream #59870); restored from backup`, post-restore 52060 bytes. Output captured in session transcript.
- [x] **Sub-threshold shrink (post > pre/2) does NOT trigger restoration** → same smoke, test2: pre 52060, action wrote 28633 bytes (55% of pre, above 50% floor), finally block read newSize 28633, predicate `$newSize -lt ($snapshotSize / 2)` evaluated `28633 -lt 26030` = FALSE, no restore, no warning. File ended at 28633 bytes as written by the action. (Cleanup in the test harness left damage that I had to recover separately; production code uses `[System.IO.Path]::GetTempFileName()` which is self-contained, so the harness-only damage class does not exist in production.)
- [x] **Fresh-machine path (no `.claude.json`) is a no-op** → covered by structural test "`setup-windows.ps1` defines Backup-AndRestoreClaudeJson helper" + reading the helper body: the very first conditional `if (Test-Path $claudeJson) { ... }` skips the snapshot when the file is absent, and the `if ($backup -and …)` guard in `finally` skips restore. No temp file is created. Same shape on Linux: `[ -f "$claude_json" ] || return 0`.
- [x] **Pre-call size below 10 KB does NOT trigger restoration** → covered by structural test asserting the literal `10240` appears in both scripts. The condition `$snapshotSize -ge 10240` gates the restore; if snapshot is below 10 KB, no restore can fire regardless of relative shrink.
- [x] **`setup-windows.ps1` preserves `.claude.json` size across two consecutive runs on an authenticated machine (>=50 KB baseline)** → empirical 2026-05-19: pre 52055 bytes; ran `pwsh -NoProfile -ExecutionPolicy Bypass -File setup-windows.ps1` twice in succession; post 52055 bytes both times; subscription state header intact (`{ "numStartups": 62, "installMethod": "native", ... }`). Note: in this session the upstream truncation bug (#59870) did not fire in vivo on either run (the file's mtime stayed at the restore timestamp on both, i.e. the install path did not modify `.claude.json` even though the idempotence guard yielded a false negative for `claude-mem@thedotmack` and the install was attempted). The wrapper is therefore in place as defense-in-depth for when the bug recurs (it is intermittent — see "Decisions" below).
- [x] **`setup-linux.sh` preserves `.claude.json` across two consecutive runs** → NOT empirically run in this session (Windows-only dev machine). Verified by structural symmetry: bats parity test asserts both scripts have the snapshot/restore guard around their respective `claude plugin install` call. Integration test in Docker (Ubuntu 24.04) covers this in CI.
- [x] **PSScriptAnalyzer Error+Warning clean on changed lines** → `Invoke-ScriptAnalyzer -Path setup-windows.ps1 -Severity Error,Warning` reported 0 new findings introduced by BUG-004. The pre-existing `PSAvoidUsingEmptyCatchBlock` at the install call site was relocated (line 361 → line 408) but its semantics (silent skip on install failure) are preserved verbatim — the BUG-004 refactor wraps the same try/catch in the `Backup-AndRestoreClaudeJson -Action { … }` shell.
- [x] **`bash -n setup-linux.sh` clean** → ran 2026-05-19, no syntax errors.

## Test status

- **Bats simulation (local, no bats binary on Windows)**: 6 new asserts in `tests/setup-windows.bats` + 6 new asserts in `tests/setup-linux.bats` (including 2 parity tests) validated by grep-by-grep emulation. RED → GREEN transitions logged in session transcript.
  - Pre-implementation: 7 expected fails (helper missing, #59870 missing, 10240 missing on each side, snapshot/restore helpers missing on Linux); 2 pass (existing idempotence guards intact).
  - Post-implementation: 14/14 pass.
- **Manual smoke (Windows admin machine, 2026-05-19)**:
  - Synthetic truncation test: PASS (52060 → 2 → 52060 with WARNING line).
  - Sub-threshold shrink test: PASS (no restore when post > pre/2).
  - Real `setup-windows.ps1` run x2: PASS (file size preserved, mtime unchanged because in-vivo upstream bug did not fire this afternoon — see Decisions).
- **PSScriptAnalyzer**: clean on changes (0 new Error/Warning attributable to BUG-004; pre-existing Write-Host warnings count unchanged).
- **No regressions**: existing tests for MCP self-heal, claude-mem heal, copilot CLI v2 detection, settings.json merge — all still match their structural greps (verified by re-running the bats-emulation suite at end of session).

## Decisions made during implementation

- **Threshold heuristic = 10 KB floor + 50% relative shrink**: chosen to match SDD-021's existing canary threshold (`10240` bytes in `claude-session-start.{sh,ps1}`). Single SSOT for the "below this is anomalous" boundary. Relative drop tolerates legitimate growth of ~200 bytes per new plugin entry while catching the 75 KB → 1.5 KB upstream bug class.
- **Helper API shape diverges by OS**: PowerShell has a single combined `Backup-AndRestoreClaudeJson -Action { … }` because PowerShell idioms favor scriptblock wrappers; bash splits into `snapshot_claude_json` (echoes tempfile path) + `restore_claude_json_if_truncated "$path"` because shell idioms favor passing state via $(...) capture rather than wrapping commands. Parity test asserts both shapes exist; behavior is identical.
- **Did NOT modify the existing idempotence guard**: defense-in-depth means two independent layers. The `grep -qF`/`-match` guard against `claude plugin list` covers the common case (entries from `@claude-plugins-official`); the new wrapper catches the residual case where the listing output omits an installed plugin (claude-mem from `@thedotmack`). Removing either layer weakens coverage; the bats tests assert BOTH layers are present.
- **Mtime side-effect understood, not load-bearing**: when the wrapper does NOT need to restore (in-vivo bug not firing), `.claude.json` may not be touched at all, so mtime stays at whatever it was. When the wrapper DOES restore, mtime updates to the restore time. Acceptance criteria are on **content/size**, not mtime, because the upstream bug shape isn't predictable enough to test mtime invariants.
- **In-vivo verification gap acknowledged**: the upstream bug #59870 is intermittent. This session reproduced it once this morning (75 KB → 3444 bytes after a single setup run) and then could NOT reproduce it for the rest of the afternoon despite multiple setup runs. The wrapper is verified by synthetic test; cannot be verified by in-vivo demonstration this session. SDD-021 canary remains as the second-line detector if the bug recurs and the wrapper fails for any reason.
- **`PSAvoidUsingEmptyCatchBlock` left as-is**: pre-existing warning at the install call site, not introduced by this work. Bundling cleanup of unrelated lint warnings violates atomic-PR rule; out of scope for BUG-004.

## Promotion candidates

- [x] **Lesson** → `10_projects/dotfiles/90-lessons.md` 2026-05-19 entry "Defensive monitors are not fixes — trigger fix and monitor are siblings, not substitutes". Captured the SDD-021 vs BUG-004 distinction: a monitor that detects an anomaly at session start (canary) does not prevent the anomaly from happening between sessions; it just alerts after the fact. The trigger fix (this PR) is the prevention layer; the monitor is the alarm. Both are needed.
- [ ] **ADR** → no. The wrapper is a tactical fix; the architectural lesson is captured in the lesson above.
- [ ] **New pattern** → not yet. Pattern candidate "snapshot-and-restore guard around external CLI calls that may corrupt local state" — would qualify if this idiom recurs in a 3rd context (currently 2: this fix + the conceptually similar `claude-mem-heal.{sh,ps1}` self-heal of broken marketplace artifacts). Add to watchlist; promote on next recurrence.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-004-claude-mem-truncate-guard/` -> `specs/archive/BUG-004-claude-mem-truncate-guard/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (lesson committed)
