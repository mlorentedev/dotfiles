---
tags: [spec, tasks, secrets, cli, go, bitwarden]
created: "2026-06-26"
---

# Tasks - CLI-024-secrets-bw-writer

> TDD order. One task = one focused commit.

## Setup

- [x] Branch from main: `feat/secrets-bw-writer`
- [x] `proposal.md` complete; acceptance criteria testable

## Implementation

- [x] `BWWriter` interface (`SetField`), the write analog of `BWReader`.
- [x] `setItemField(itemJSON, field, value)` — pure read-modify-write over a generic
  map: typed login (created if absent), notes, custom field update/append; preserves
  every other key. `itemID` + `setCustomField` helpers.
- [x] `BWPut` production writer (shell-out: `bw get item` → setItemField → base64 →
  `bw edit item <id>`; absent item → clear error; value via stdin, never a temp file).
- [x] Tests: `TestSetItemField_*` (typed login + siblings, login-created, notes,
  custom update, custom append, bad JSON), `TestItemID`, `TestBWWriter_MockRoundTrip`.

## Closing

- [x] Every AC covered by ≥1 test (BWPut shell-out is live-smoke, like BWGet)
- [x] `features.json` carries non-vacuous verification commands
- [x] `go test ./internal/secrets` green; `go vet` + `gofmt` clean; `go build ./...` ok
- [x] Additive (no existing behaviour changed); `registry.yaml` untouched
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec + #612 C2
