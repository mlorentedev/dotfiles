---
id: "BUG-032-windows-guard-001"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-07-14"
issue: "mlorentedev/dotfiles#691"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, guard-001, memory-sink, powershell, doctor, fresh-machine, security]
template_version: "1.0"
---

# BUG-032-windows-guard-001

Install the GUARD-001 memory-sink dispatcher on Windows (the setup twin
`install-git-hooks.sh` always flagged as a follow-up) and fix the POSIX-only
absolute-path check that made `dotf doctor` false-FAIL the agy AGY_APP_DATA on
Windows.

## Why

Two Windows-side gaps in the memory-sink guard and doctor (audit C20 + the
GUARD-001 Windows gap):

- **GUARD-001 never installs on Windows.** `setup-windows.ps1` has zero
  `core.hooksPath` / git-hooks deploy references (grep-confirmed). AGENTS.md's
  "MEMORY SINGLE-SINK (GUARD-001)" advertises a machine-wide pre-commit guard that
  rejects `MEMORY.md`/`memory/` outside the vault — on Windows it simply does not
  exist, so an agent can commit memory artifacts into any repo. The Linux setup
  deploys the dispatcher and wires the global hook via
  `scripts/install-git-hooks.sh`; there is no Windows equivalent, and
  `install-git-hooks.sh` itself notes "a Windows setup twin is a tracked
  follow-up." Empirically on a real box: `dotf doctor` reports
  `[FAIL] dispatcher not found at C:\Users\<u>\.dotfiles\git-hooks — run dotfiles
  setup to deploy git-hooks/`, and `core.hooksPath` is unset.
- **C20 — doctor false-FAIL on a valid Windows path.** `checks_deploy.go:350`
  gates `AGY_APP_DATA` with `strings.HasPrefix(agyData, "/")`. An absolute Windows
  path (`C:\Users\…\.gemini\antigravity-cli`) does not start with `/`, so whenever
  agy is on PATH on Windows `dotf doctor` reports "AGY_APP_DATA is relative or
  unset" — the same POSIX-only assumption the #551 `contractOS` work removed
  elsewhere in this package.

## What

- **`scripts/install-git-hooks.ps1`** — the PowerShell twin of
  `install-git-hooks.sh`. Two functions plus an entry point, matching the shell
  contract exactly:
  - `Deploy-GitHooks <src> <dest>`: clean-mirror the dispatcher tree into
    `<dest>` (remove-then-copy, so a hook removed upstream never lingers — a stale
    security hook is worse than none). Same safety guards: refuse a `<dest>` that
    is not a `*\git-hooks` path, refuse `$HOME\git-hooks` / a drive root, and no-op
    when `src` and `dest` resolve to the same directory (#695 self-empty guard).
  - `Set-GlobalHooksPath <target>`: wire global `core.hooksPath` to `<target>`
    **only when unset**; an unrelated pre-existing value is preserved and warned
    (machine-wide blast radius), an already-correct value is a no-op. Mirrors the
    `dotf doctor` `checkGuardHooks` contract.
  - `Install-GitHooks [src] [dotfilesDir]`: deploy + wire, defaulting `src` to the
    repo's `git-hooks/` and `dotfilesDir` to `~/.dotfiles`.
- **`setup-windows.ps1`** sources `install-git-hooks.ps1` and calls
  `Install-GitHooks`, non-fatal (a warning on failure, like the Linux side), so a
  fresh Windows box ends with the dispatcher at `~/.dotfiles\git-hooks` and
  `core.hooksPath` wired. The wired path is `Join-Path $DotfilesDest 'git-hooks'`,
  which round-trips byte-for-byte through `git config` and equals the Go
  `filepath.Join(cfg.DotfilesDir, "git-hooks")` the doctor compares against
  (verified empirically), so `dotf doctor` then PASSES on Windows.
- **C20 fix (Go).** `checks_deploy.go`: `strings.HasPrefix(agyData, "/")` ->
  `filepath.IsAbs(agyData)`, so an absolute Windows path passes.
- **Tests.** Pester for `install-git-hooks.ps1` (clean-mirror deploys the
  dispatcher; re-deploy prunes a stale hook; wire sets an unset hooksPath;
  preserves an unrelated value; refuses an unsafe dest) driven against fixtures
  under an isolated temp dir with a throwaway `--file` git config — no real
  `~/.gitconfig` mutation. Go unit test for the `filepath.IsAbs` branch.

## Out of scope

- **Porting the guard install to Go.** Bootstrap stays shell (ADR-020 C7): the
  deployed `dotf` release carries no source tree, so it cannot self-deploy
  `git-hooks/`; the setup bootstrap places them and `dotf doctor` verifies. The
  `.ps1` twin mirrors the existing `.sh`; this is not a new-tooling case.
- **The dispatcher hooks themselves.** They are POSIX and already run under
  Git-for-Windows `sh`; unchanged.
- **The other fresh-machine bugs** (#690, #688) — their own issues.

## Risks / open questions

- **Path-match (resolved).** `git config --global core.hooksPath` stores a
  backslash Windows path and `--get` returns it unchanged (verified with a
  throwaway config file), so the PS-wired `~/.dotfiles\git-hooks` equals the
  doctor's `filepath.Join(cfg.DotfilesDir, "git-hooks")` and `current == target`
  holds. No forward/back-slash normalization drift.
- **Machine-wide blast radius.** Global `core.hooksPath` affects every repo. The
  wire is unset-only + preserve-existing + warn, identical to the Linux twin and
  the doctor `--fix`; never clobbered.
- **Idempotency.** Clean-mirror + unset-only wire are safe to re-run each setup.
- **Windows CI.** The `test-windows` Pester job runs `setup-windows.ps1` e2e on a
  runner; the new install runs there. PSScriptAnalyzer must be extended to cover
  the new script (currently only 4 files; tracked in #692) — meanwhile it is
  linted locally.

## Acceptance criteria

- [ ] `scripts/install-git-hooks.ps1` `Deploy-GitHooks` clean-mirrors `git-hooks/`
      to a `*\git-hooks` dest and refuses unsafe dests.
- [ ] `Set-GlobalHooksPath` wires an unset `core.hooksPath`, no-ops when already
      correct, preserves+warns on an unrelated value.
- [ ] `setup-windows.ps1` deploys the dispatcher to `~/.dotfiles\git-hooks` and
      wires `core.hooksPath`; afterward `dotf doctor` GUARD section PASSES on
      Windows (verified end-to-end on a real box).
- [ ] `dotf doctor` reports AGY_APP_DATA absolute for a Windows path
      (`filepath.IsAbs`), no false-FAIL.
- [ ] Pester green for `install-git-hooks.ps1`; Go test green for the IsAbs branch;
      `go build/vet/test ./...` + PSScriptAnalyzer (changed files) clean; ASCII-only.

## References

- GH issue: [#691](https://github.com/mlorentedev/dotfiles/issues/691)
- Linux twin: `scripts/install-git-hooks.sh` (deploy + wire, the contract mirrored)
- GUARD-001 history: #398 (dispatcher), #415 (verifier), #418 (wiring), #695 (self-empty guard)
- ADR-020 C7 (bootstrap stays shell); ADR-025 (cross-machine paths)
- Sibling fresh-machine bugs: #690, #688; PSSA coverage gap: #692
