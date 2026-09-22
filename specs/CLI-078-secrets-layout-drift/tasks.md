---
tags: [spec, tasks, templates]
created: "2026-09-21"
---

# Tasks - CLI-078-secrets-layout-drift

## Setup

- [x] Branch created from main: `feat/secrets-layout-drift`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] Open questions recorded with the measurement that settled each

## Implementation

- [x] [AC6] [AC7] Metadata projection at the producer (`bwserve_list.go`): a decode
      target that cannot name a value, plus the test that plants secrets in a
      payload and proves none survives
- [x] [P] [AC1] [AC3] Folder comparison, and the taxonomy moved to the names the
      live vault carries (`Dotfiles/apps`, `Dotfiles/infra`) across
      `validBWFolders`, `planeFolder`, 26 registry lines and 7 fixtures
- [x] [AC2] Dormant declarations walked — the gap that hid three absent items
- [x] [AC4] `hasField` mirrors `fieldFromItem`, pinned by a test over all four
      resolution paths
- [x] [AC5] One finding per problem, deduped for both missing and misfiled
- [x] [AC8] `dotf secrets drift`, read-only, non-zero on findings
- [x] Derive the ratified-taxonomy error message from the set instead of repeating
      it — it named "(apps, infra)" and went on saying so after the set moved

## Closing

- [x] Every AC covered by at least one test
- [x] Every AC has a `features.json` entry with a non-vacuous command; 8/8 run green
- [x] `go build` / `go vet` / `GOOS=windows go vet` clean
- [x] `golangci-lint run` at the pinned v2.12.2 — 0 issues
- [x] Whole module green
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#1597, merged; post-merge fixes in #1600)
