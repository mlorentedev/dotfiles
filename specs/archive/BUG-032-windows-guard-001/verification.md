---
tags: [spec, verification, guard-001, powershell, doctor, fresh-machine, security]
created: "2026-07-14"
---

# Verification - BUG-032-windows-guard-001

## Evidence

- [x] `Deploy-GitHooks` clean-mirrors to a `*\git-hooks` dest and refuses unsafe
      dests -> `tests/install-git-hooks.Tests.ps1` (mirror/prune/refuse-non-git-hooks/
      refuse-drive-root/refuse-no-pre-commit), 8/8 Pester.
- [x] `Set-GlobalHooksPath` wires an unset `core.hooksPath`, no-ops when correct,
      preserves+warns on an unrelated value -> same suite, isolated via a throwaway
      `GIT_CONFIG_GLOBAL`.
- [x] `setup-windows.ps1` deploys the dispatcher to `~/.dotfiles\git-hooks` and
      wires `core.hooksPath`; `dotf doctor` GUARD section then PASSES on Windows ->
      **verified end-to-end on this box**: with a temp `DOTFILES_DIR` +
      `GIT_CONFIG_GLOBAL`, `Install-GitHooks` wired
      `...\.dotfiles\git-hooks`, the dispatcher `pre-commit` was present, and
      `dotf doctor` reported `[GUARD memory-sink hooks] (1 checks, all ok)`.
- [x] `dotf doctor` reports AGY_APP_DATA absolute for a Windows path ->
      `TestCheckAntigravity_AbsolutePathAccepted` (OS-aware): proven red under the
      old `HasPrefix("/")` (FAILed `C:\Users\me\...`), green under `filepath.IsAbs`.
- [x] `go build`/`vet`/`test ./...` clean; Pester 8/8; PSScriptAnalyzer clean on
      the changed `.ps1`; all changed `.ps1` ASCII-only.

## Test status

- Go: `go build/vet/test ./...` -> ok; `TestCheckAntigravity_AbsolutePathAccepted`
  passes (and fails under the reverted `HasPrefix`, confirming it guards the bug).
- Pester (Windows, this box): `install-git-hooks.Tests.ps1` -> Passed=8 Failed=0.
- End-to-end (this box): install + `dotf doctor` agree — the PS-wired path equals
  the Go `filepath.Join(cfg.DotfilesDir,"git-hooks")` target, so the guard shows
  "all ok" rather than a false unwired/absent FAIL.
- PSScriptAnalyzer: PASS on `install-git-hooks.ps1`, `setup-windows.ps1`,
  `install-git-hooks.Tests.ps1` (repo settings, Error+Warning).

## Decisions made during implementation

- **Verify the path-match empirically BEFORE coding.** The one real risk was that
  git normalizes a backslash `core.hooksPath` and breaks the doctor's exact
  `current == target` compare. A throwaway `git config --file` round-trip proved
  the backslash path is stored+returned verbatim, so replicating the path in
  PowerShell (mirroring the Linux twin) is safe — no need to route the wiring
  through `dotf doctor --fix`.
- **A `.ps1` twin of `install-git-hooks.sh`, not a Go port.** Bootstrap stays shell
  (ADR-020 C7): the deployed `dotf` carries no source tree, so setup must place the
  dispatcher. Factored into `scripts/install-git-hooks.ps1` (not inlined in
  setup-windows.ps1) for Pester-testability + symmetry with the `.sh`.
- **Split the pure IsAbs test OS-aware.** `filepath.IsAbs` is OS-dependent
  (`C:\...` is absolute only on Windows); the test uses the current OS's absolute
  form so it is green on both the ubuntu and windows Go runners while the
  windows-latest run exercises the exact regression.
- **`PSUseSingularNouns` suppressed with justification** rather than renaming:
  "GitHooks" is inherently plural and mirrors `install-git-hooks.sh` / the
  `git-hooks/` dir. (Not CI-caught today — install-git-hooks.ps1 is outside the
  hardcoded lint list, #692 — but fixed so a coverage fix later stays green.)

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? yes (small) — "when a Windows twin must match a
      Go-computed path via `git config`, verify the storage round-trip empirically
      before assuming string equality." Capture on merge if it recurs; the
      proposal + this file already record it.
- [ ] ADR-worthy? no — applies ADR-020 C7, does not change it.
- [ ] New vault pattern? no — repo-local.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/BUG-032-windows-guard-001/`
- [ ] Bitácora ticket (#691) closed with PR link (ADR-018)
