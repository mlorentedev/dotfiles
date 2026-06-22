---
tags: [spec, verification]
created: "2026-06-21"
---

# Verification - CLI-018-dotf-doctor-windows-repoint

## Evidence (PR-A)

- [x] **Orca hook check behaves** → `checks_orca.go` `checkOrcaHook`; `TestCheckOrcaHook` table covers skip-when-absent, `timeoutSec` ≥30 PASS / <30 FAIL / any-too-low FAIL, `HttpWebRequest` PASS / `Invoke-WebRequest` FAIL, and both-files combined.
- [x] **Registered in the full sweep, not `--quick`** → `doctor.go` calls `checkOrcaHook` inside the `!opts.Quick` block after `checkAntigravity`.
- [x] **Cross-OS** → no `GOOS` gate; SKIPs when neither Orca file is present (the off-Windows / no-Orca state).

## Evidence (PR-B0 — §4 residual port, build-only)

- [x] **§4 residual ported** → `checkProfileFiles` (`checks_profile.go`): `.claude/CLAUDE.md`
  + `.gemini/AGY.md` existence (cross-OS, recovers Linux coverage the CLI-012
  consolidation dropped) and the Windows-only `$PROFILE`. Registered after
  `checkSymlinks` in the `!opts.Quick` sweep.
- [x] **`$PROFILE` resolved by candidate paths, not a pwsh shell-out** →
  `{Documents, OneDrive\Documents} × {PowerShell, WindowsPowerShell}` /
  `Microsoft.PowerShell_profile.ps1`. Pure-Go + deterministic (fits the framework's
  temp-tree test model) and covers the OneDrive-redirected Documents common on
  corporate boxes, which a naïve `~\Documents` check would false-FAIL.
- [x] **Table test green** → `TestCheckProfileFiles` (7 rows): posix both-present
  (profile SKIP) + CLAUDE.md/AGY.md missing fails; windows pwsh / WinPS / OneDrive
  pass + profile-missing fail. `go test ./internal/doctor/...` ok; gofmt + vet +
  golangci-lint clean.
- [x] **No coverage lost on PR-B's deletion** → `healthcheck.ps1` §4's three
  deployed-file checks now live in `dotf doctor`; PR-B can delete the `.ps1`.

## Evidence (PR-B — delete the `.ps1` twins + repoint all callers)

- [x] **Retired scripts deleted** → `git rm scripts/healthcheck.ps1 scripts/doctor.ps1
  tests/healthcheck-ps1.bats`. `dotf doctor` is now the only Windows diagnostic.
- [x] **All production callers repointed** →
  - `setup-windows.ps1`: dropped both deploy blocks (no longer copies the `.ps1`
    to `ScriptsDir`) and replaced the 8c/8d invoke blocks with a single non-fatal
    `dotf doctor` post-setup block (does NOT alter `$LASTEXITCODE`).
  - `.github/workflows/ci.yml`: `test-windows` runs `dotf doctor` (PATH prepended
    with `~/.local/bin`) instead of `healthcheck.ps1`.
  - `powershell/profile.ps1`: `hc` wraps `dotf doctor @args`.
  - `scripts/claude-session-start.ps1`: SessionStart drift surfaces via
    `dotf doctor --quick` (mirrors the Linux hook).
  - `README.md`: dropped the `healthcheck.ps1` tree line.
- [x] **Guard tests added** (`tests/setup-windows.bats`) →
  - `CLI-018: healthcheck.ps1 and doctor.ps1 are deleted` (file-absence).
  - `CLI-018: no production caller references healthcheck.ps1 or doctor.ps1`
    (greps the 6 production files for the `(healthcheck|doctor)\.ps1` token).
    Retired-script *mentions* in comments were reworded to "doctor + healthcheck
    shell scripts" so the guard catches live references, not documentation.
- [x] **Parity + structural bats updated** → `setup-linux.bats` (both OSes run
  `dotf doctor`; SessionStart drift via `--quick`), `powershell-profile.bats`
  (`hc` wraps `dotf doctor`, no `healthcheck.ps1`), `opencode.bats` + `pi-config.bats`
  (the old `healthcheck.ps1` assertions are now comments — behaviour lives in
  `go test`).
- [x] **No-silent-drop flagged** → `doctor.ps1 -Fix` auto-invoked `profile-heal.ps1`
  (BUG-020). `dotf doctor --fix` (`runHeals`) does NOT — it only runs
  `claude-mem-heal.sh`. `profile-heal.ps1` stays deployed and the BUG-021 setup
  preflight still points at it (manual recovery intact); the doctor auto-heal port
  is tracked in **#531**. `tests/profile-heal-ps1.bats` swaps the `doctor.ps1 -Fix`
  assertion for an explanatory comment referencing #531.

## Test status

- `go -C cli test ./internal/doctor/...` → **ok** (4.9s), incl. `TestCheckOrcaHook`.
- `go -C cli vet ./internal/doctor/...` → clean. `gofmt -l` clean for the 3 touched files.
- `go -C cli test ./...` → `internal/spec` + `internal/vault` `TestEmbeddedTemplatesMatchVault` **fail locally** — this is the **pre-existing #461 drift** (embedded templates vs vault SSOT), unrelated to this change (doctor-only) and a SKIP on CI (no vault). `internal/doctor` is green.

## Decisions made during implementation

- **Cross-OS, not `GOOS`-gated.** The Orca check skips on file-absence, which is equivalent to a Windows gate (the files only exist where Orca is installed) but simpler and faithful to the `.ps1` (which also skips when absent).
- **`timeoutSec` via regex, not full JSON parse** — mirrors the PowerShell check exactly (flag *any* hook timeout < 30), and avoids coupling to orca.json's nested schema.
- **§4 junction (BUG-012) not ported** — the `.ps1` itself marks it *secondary*, superseded by BUG-014 (already in `dotf doctor`'s `checkClaudeMem`); claude-mem-heal owns the repair. §4 deployed-file residual deferred to PR-B.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? no — the cross-OS-parity lesson is already captured (CLI-020).
- [ ] ADR-worthy? no — executes ADR-020/021.
- [ ] New pattern? no.

## Archive checklist (after PR-B merges)

- [ ] `proposal.md` `status: archived`
- [ ] Folder → `specs/archive/CLI-018-dotf-doctor-windows-repoint/`
- [ ] Close #380 (PR-B link)
