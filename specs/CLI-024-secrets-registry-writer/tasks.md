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

- [x] `SetBackendBW(data, id)` — line surgery: flip `backend:` to bw, drop `age:`, keep
  the pre-declared `bw:` block in place; touches only those two lines. Guarded to the
  single scalar env-var shape WITH a declared bw: target; idempotent; re-validated via
  `ParseRegistry`. Helpers: `secretBlock`, `assertMigratable`, `leadingSpaces`.
- [x] Tests: `TestSetBackendBW_FlipsOnlyTargetBlock` (preservation of comment
  alignment + blank line + other block), `TestSetBackendBW_RealRegistry_OnlyTargetChanges`
  (golden: exactly one line dropped ⇒ every line outside the target block content-
  identical against the real registry.yaml), `TestSetBackendBW_Idempotent`,
  `TestSetBackendBW_Guards` (unknown id / multi-var / file / no declared bw: target).

## Closing

- [x] Every AC covered by ≥1 test
- [x] `features.json` carries non-vacuous verification commands
- [x] `go test ./internal/secrets` green; `go vet` + `gofmt` clean; `go build ./...` ok
- [x] No command wired yet (this PR is the primitive; `migrate` consumes it later)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec + #612 C2
