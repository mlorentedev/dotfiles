---
tags: [spec, tasks, secrets, cli, go]
created: "2026-06-26"
---

# Tasks - CLI-024-secrets-registry-writer

> TDD order. One task = one focused commit.

## Setup

- [x] Branch from main: `feat/secrets-registry-writer`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Probe: yaml.v3 Node round-trip rejected empirically (collapses blank lines +
  comment alignment) → surgical text approach chosen

## Implementation

- [x] `SetBackendBW(data, id, item, field)` — line surgery: flip `backend:` to bw,
  drop `age:`, insert `bw: { item, field }`; touches only those lines. Guarded to the
  single scalar env-var shape; idempotent; re-validated via `ParseRegistry`. Helpers:
  `secretBlock`, `assertSingleScalarEnv`, `leadingSpaces`, `isInlineMapping`.
- [x] Tests: `TestSetBackendBW_FlipsOnlyTargetBlock` (preservation of comment
  alignment + blank line + other block), `TestSetBackendBW_RealRegistry_OnlyTargetChanges`
  (golden: net-zero lines ⇒ every line outside the target block byte-identical against
  the real registry.yaml), `TestSetBackendBW_Idempotent`, `TestSetBackendBW_Guards`
  (unknown id / multi-var / file / empty item / empty field).

## Closing

- [x] Every AC covered by ≥1 test
- [x] `features.json` carries non-vacuous verification commands
- [x] `go test ./internal/secrets` green; `go vet` + `gofmt` clean; `go build ./...` ok
- [x] No command wired yet (this PR is the primitive; `migrate` consumes it later)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec + #612 C2
