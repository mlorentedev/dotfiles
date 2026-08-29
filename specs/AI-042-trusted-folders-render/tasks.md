---
tags: [spec, tasks]
created: "2026-08-29"
---

# Tasks - AI-042-trusted-folders-render

## Setup

- [x] Branch: `feat/trusted-folders-render` (worktree `dotfiles-wt-trusted`), stacked on `fix/deploy-manifest-v2` (#1369) until it merges
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left in `proposal.md`

## Implementation

- [x] [P] [AC1] Failing tests `TestExpandPaths_RendersTheDeclaredForm`, `TestExpandPaths_RejectsWhatItCannotRender`
- [x] [AC1] `expandPaths` (JSON-aware, `native` | `slash`), `Config.Paths`, `ParseManifest` validation, `ManifestVersion` 3 + frozen field set
- [x] [P] [AC2] Failing test `TestDeploy_PathsComposeWithMergeAndReportInSync`
- [x] [AC2] Expansion in `load()` so merge, replace and PlanConfig all see the rendered source
- [x] [P] [AC3] `ai/copilot/config.json` and `ai/agy/settings.json` templated; manifest `copilot-config` gains `paths: native`, new `agy-settings` (replace, `paths: slash`); both setups stop copying `ai/agy/settings.json`
- [x] [AC3] bats: no user/home literal under `ai/**/*.json` (every file, one `refute_grep` each); templates present; manifest shape; the copy is gone from both setups
- [x] [AC4] Box: `dotf deploy` → Copilot native, agy slash, `firstLaunchAt` kept; second run in sync; doctor green; `copilot -p` + `agy` smoke
- [x] Mutation checks: native form without `FromSlash` fails AC1; expansion skipped in `load()` fails AC2

## Verification

- [x] Go loop: build, vet (host, `GOOS=linux`), test, golangci-lint (0 issues)
- [x] bats: copilot-config 14/14, antigravity, setup-linux/setup-windows subsets 40/40; shellcheck 0; `setup-windows.ps1` parse 0 errors, ASCII delta 0, CRLF intact
- [x] `verification.md` records the evidence; `features.json` per AC
