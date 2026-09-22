---
tags: [spec, tasks, templates]
created: "2026-09-22"
---

# Tasks - CLI-080-secrets-reconcile

## Setup

- [x] Branch created from main: `feat/secrets-reconcile` (rebased on #1600, whose inventory fixes this plans from)
- [x] `proposal.md` complete, acceptance criteria testable
- [x] Open questions resolved in `proposal.md` (source of values, rotation age, stale cache, apply-before-merge)

## Implementation

- [x] [AC7] `bw.from` in the registry schema, validated on live and dormant blocks
- [x] [AC1] [AC4] [AC5] Planner: drift findings → operations, blocked and deferred notes, satisfied records
- [x] [AC8] `MoveItem` on both backends through one pure `setItemFolder`; SetField and MoveItem share one read-modify-write per backend
- [x] [AC2] [AC3] [AC5] `ApplyReconcile`: every source read before the first write; folders resolved once per run
- [x] [AC1] [AC6] `dotf secrets reconcile`: sync before planning, `--apply` re-plans and fails unless empty
- [x] `GITHUB_PERSONAL_ACCESS_TOKEN` and `RELEASE_TOKEN` declare `from:` and flip to bw
- [x] `TestSetBackendBW_RealRegistry_OnlyTargetChanges` discovers its target instead of pinning one
- [x] Operator protocol in `docs/runbooks/guide-secrets-governance.md` (CONVERGE), stale folder names fixed

## Verification

- [x] `go build`, `go vet`, `GOOS=windows go vet`, `go test ./...`, pinned `golangci-lint` — clean
- [x] AC1–AC8 `features.json` commands executed — 8/8 pass
- [x] Mutation — 16 mutations across planner, apply and command; see `verification.md`
- [x] Live plan against the real store (read-only)
- [x] [AC9] `--apply` against the live store (operator-authorized), clean re-plan, both PATs HTTP 200, `RELEASE_TOKEN` synced to its CI consumer
- [x] `sync ci [SECRET_NAME...]` scoping, found necessary during the live sync (model defect ticketed as #1603)

## Wrap-up

- [x] PR opened referencing this spec folder (#1601)
- [ ] Independent adversarial review before archive
