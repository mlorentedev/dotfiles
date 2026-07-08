---
tags: [spec, verification]
created: "2026-06-18"
---

# Verification - WIN-006-windows-dotf-install

## Evidence

- [x] **AC1 — fetch/verify/install** → `scripts/install-dotf.ps1`. **Real-release smoke** (this machine): `.\scripts\install-dotf.ps1` → `dotf 0.6.0 installed to C:\Users\mlorente\.local\bin\dotf.exe`; `dotf version` → `dotf version 0.6.0`; `dotf env path VAULT_PATH` → `C:\Users\mlorente\Projects\Workspace\knowledge` (machine.json resolved). User-space `~/.local/bin`, no admin.
- [x] **AC2 — setup wiring + lint** → `setup-windows.ps1` dot-sources + `Install-Dotf` (non-fatal) before the hive/`dotf env` block. `Parser::ParseFile` clean on both files; `Invoke-ScriptAnalyzer -Settings .PSScriptAnalyzerSettings.psd1 -Severity Error,Warning` = **0** on both.
- [x] **AC3 — bats + guard** → `bats tests/install-dotf-ps1.bats` → **11/11**. Dot-source guard: `pwsh -c '. .\scripts\install-dotf.ps1'` defines `Install-Dotf` **without** auto-installing (run-guard `$MyInvocation.InvocationName -ne '.'`).

## Test status

- `install-dotf.ps1` parse OK; PSSA (CI settings) = 0 warnings/errors.
- `setup-windows.ps1` parse OK; PSSA (CI settings) = 0.
- `bats tests/install-dotf-ps1.bats` → 11 ok / 0 failed.
- Real binary smoke: dotf 0.6.0 installed + functional (`version`, `env path`).

## Decisions made during implementation

- **No `file://` behavioral bats.** PowerShell `Invoke-WebRequest` has no `file://` scheme, so the `install-dotf.sh` fixture-driven download tests can't be mirrored in bats. Coverage is the structural/convention bats (repo .ps1 convention) + the real-release smoke + the fact that the verify/extract logic is a line-for-line mirror of the bats-tested `.sh`.
- **`Install-Dotf` never throws** — returns `$true`/`$false` so setup wires it `if (-not (Install-Dotf)) { Write-Warn }`, the analogue of `install_dotf || log_warning`. EA=Stop is set inside the function (not leaked to the dot-sourcing caller).
- **Run-guard, not auto-run-on-source.** Dot-sourcing (`. install-dotf.ps1`) only defines functions; executing (`.\install-dotf.ps1`, `irm | iex`) installs — so setup dot-sources + calls explicitly, and the one-liner still works.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? **maybe** — "a release binary that goreleaser already builds is worthless until each OS's setup actually downloads it; the gap reads as 'needs Go' but it's just a missing fetch step."

## Archive checklist

- [ ] PR merged, closes #451.
- [ ] Deploy verified: a clean `setup-windows.ps1` run installs dotf with no manual step.
- [ ] `proposal.md` `status: archived`; folder → `specs/archive/WIN-006-windows-dotf-install/`.
- [ ] Bitácora #451 → Done.
