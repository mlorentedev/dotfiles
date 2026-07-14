---
tags: [spec, verification, memlink, project-key, powershell, fresh-machine]
created: "2026-07-14"
---

# Verification - BUG-031-windows-project-key

## Evidence

- [x] `dotf mem project-key 'C:\Users\me\p'` prints `C--Users-me-p`; `/home/me/p`
      prints `-home-me-p` -> `TestMemProjectKey` (cli/internal/cmd/mem_test.go),
      verified end-to-end via `go run ./cmd/dotf mem project-key` (win path ->
      `C--Users-me-p`).
- [x] `setup-windows.ps1` and `knowledge-crystallize.ps1` compute the key via
      `dotf mem project-key` with a corrected fallback -> `Get-ClaudeProjectKey`
      (scripts/utils.ps1) called at setup-windows.ps1:874 and
      knowledge-crystallize.ps1 `Get-MemoryFilePath` / single-project branch.
- [x] `knowledge-crystallize.ps1 -All` discovery uses the double-dash decoder ->
      `Get-DecodedPath` regex `^([A-Za-z])--(.+)$`; Stage 2 uses the pure
      `Get-ClaudeProjectKeyEncoded` (no per-directory subprocess).
- [x] `vault-maintenance-weekly.ps1` invokes the crystallizer with `-All` (was the
      POSIX `--all` that bound positionally to `$ProjectDir`).
- [x] Pester guard: PS fallback key == `dotf mem project-key` == expected Go key ->
      `tests/claude-project-key.Tests.ps1`, 10/10 passing (incl. the dotf
      cross-check against the PR-built binary).
- [x] `go build`/`vet`/`test ./...` clean; PSScriptAnalyzer clean on the two
      CI-linted changed files; every changed `.ps1` ASCII-only.

## Test status

- Go: `go build ./... && go vet ./... && go test ./...` -> all packages `ok`
  (cli/internal/cmd, memlink, doctor, env, mem, ... green). `TestMemProjectKey`
  passes.
- Pester (Windows, this box): `Invoke-Pester tests/claude-project-key.Tests.ps1`
  -> `Tests Passed: 10, Failed: 0`. The dotf cross-check ran green against a
  PR-built `dotf` on PATH.
- PSScriptAnalyzer (repo settings, Error+Warning): PASS on `setup-windows.ps1`,
  `scripts/knowledge-crystallize.ps1`, `scripts/utils.ps1`,
  `scripts/vault-maintenance-weekly.ps1`, `tests/claude-project-key.Tests.ps1`.
- ASCII-only: all changed `.ps1` clean. (Pre-existing non-ASCII in
  `setup-windows.ps1` comments is untouched and out of scope -> tracked in #692.)
- No regressions in the existing suite.

## Decisions made during implementation

- **Approach B (SSOT via Go) over re-syncing three encoders** (user decision): the
  duplicated encoding datum was the root cause, so Go (`memlink.ClaudeProjectKey`)
  becomes the single source and the PowerShell twins call `dotf mem project-key`.
- **Split the PS helper in two** (`Get-ClaudeProjectKeyEncoded` pure +
  `Get-ClaudeProjectKey` dotf-first): the decoder's Stage-2 scan encodes every
  directory under `USERPROFILE`; shelling `dotf` per directory there would be a
  perf regression, so hot loops use the pure encoder and single-shot resolutions
  use the dotf-first wrapper. A Pester guard binds pure == dotf == Go.
- **Fallback is load-bearing, not defensive.** The Windows CI Pester runner
  installs the *released* dotf (no `mem project-key` yet, #734), so
  `Get-ClaudeProjectKey` degrades to the corrected pure encoder there. The guard's
  dotf cross-check probes support at discovery time and SKIPs (never fails) when
  absent (WIN-004 -Skip-at-discovery lesson).
- **Centralized the helper in `utils.ps1`** (already the cross-OS parity home and
  already sourced by setup-windows.ps1); `knowledge-crystallize.ps1` now sources it
  too, deleting its local buggy `Get-EncodedPath`.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? yes - "a shared datum re-implemented per-OS
      drifts silently; own it in the Go layer and have shells call it, guard-test
      the fallback for parity." Capture on merge.
- [ ] ADR-worthy? no - it applies ADR-020 (Go owns tooling logic), does not change it.
- [ ] New vault pattern? no - repo-local; the lesson above suffices.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-031-windows-project-key/` -> `specs/archive/BUG-031-windows-project-key/`
- [ ] Bitácora board ticket (#689) moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (the `docs/lessons.md` entry)
