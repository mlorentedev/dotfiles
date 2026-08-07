---
tags: [spec, tasks, guard-001, powershell, doctor]
created: "2026-07-14"
---

# Tasks - BUG-032-windows-guard-001

> TDD order. One task = one focused commit.

## Setup

- [x] Branch created from main: `fix/windows-guard-001-memory-sink`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Path-match risk resolved empirically before coding (git config round-trip)

## Implementation

- [x] Go: failing test for the C20 branch — an OS-absolute AGY_APP_DATA must PASS
      (`TestCheckAntigravity_AbsolutePathAccepted`, OS-aware); proved red under
      `HasPrefix("/")`, green under `filepath.IsAbs`
- [x] Go: `checks_deploy.go` `strings.HasPrefix(agyData,"/")` -> `filepath.IsAbs`
- [x] `scripts/install-git-hooks.ps1`: `Deploy-GitHooks` (clean mirror + safety
      guards), `Set-GlobalHooksPath` (wire-when-unset, preserve, no-op),
      `Install-GitHooks` (deploy+wire); dot-source-safe entry guard
- [x] Pester: `tests/install-git-hooks.Tests.ps1` (mirror/prune/refuse-unsafe;
      wire/no-op/preserve via throwaway GIT_CONFIG_GLOBAL) — 8/8
- [x] `setup-windows.ps1`: source + `Install-GitHooks -DotfilesDir $DotfilesDest`
      (non-fatal), as a new GUARD-001 section
- [x] Empirical e2e on this Windows box: install + `dotf doctor` GUARD = all ok

## Closing

- [x] Every acceptance criterion covered by a test or the e2e run
- [x] `go build`/`vet`/`test ./...` clean (golangci-lint: CI)
- [x] PSScriptAnalyzer clean on changed files (PSUseSingularNouns suppressed with
      justification for the inherently-plural GitHooks nouns); ASCII-only
- [x] No unrelated changes (pre-existing setup-windows.ps1 em-dashes untouched, #692)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

> `features.json` omitted, matching the BUG-029/030/031 precedent.
