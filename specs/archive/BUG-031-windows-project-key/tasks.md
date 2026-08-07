---
tags: [spec, tasks, templates]
created: "2026-07-14"
---

# Tasks - BUG-031-windows-project-key

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `fix/windows-twin-memory-parity`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] Go: failing test — `dotf mem project-key 'C:\Users\me\p'` prints `C--Users-me-p`
      and `/home/me/p` prints `-home-me-p` (`TestMemProjectKey`, cli/internal/cmd/mem_test.go)
- [x] Go: implement `mem project-key <path>` subcommand -> `memlink.ClaudeProjectKey`, make it pass
- [x] `setup-windows.ps1`: replace the inline encoder with `Get-ClaudeProjectKey`
      (dotf-first + corrected fallback, in utils.ps1)
- [x] `knowledge-crystallize.ps1`: source utils.ps1; delete local buggy `Get-EncodedPath`;
      `Get-MemoryFilePath` uses `Get-ClaudeProjectKey`; fix the misleading comment
- [x] `knowledge-crystallize.ps1`: `Get-DecodedPath` regex `^([A-Za-z])-` -> `^([A-Za-z])--`;
      Stage-2 scan uses pure `Get-ClaudeProjectKeyEncoded` (no per-dir subprocess)
- [x] `vault-maintenance-weekly.ps1`: `--all` -> `-All`
- [x] Pester guard: pure encoder == `dotf mem project-key` == expected Go key
      (`tests/claude-project-key.Tests.ps1`, 10/10)

## Closing

- [x] Every acceptance criterion covered by >=1 test
- [x] `go build`/`vet`/`test ./...` clean (golangci-lint: run in CI)
- [x] PSScriptAnalyzer clean on changed files; ASCII-only `.ps1`; Pester green
- [x] No unrelated changes in the diff (no scope creep; pre-existing #692 gaps ticketed, not inlined)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

> `features.json` omitted, matching the current precedent (BUG-029, BUG-030 ship
> proposal/tasks/verification only). Acceptance criteria map to Go/Pester tests
> named in `verification.md`.
