---
tags: [spec, tasks, secrets, cli, go, bitwarden]
created: "2026-06-26"
---

# Tasks - CLI-024-secrets-set

> TDD order. One task = one focused commit.

## Setup

- [x] Branch from main: `feat/secrets-set`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] No open questions left in `proposal.md`

## Implementation

- [ ] Sentinels in `bw.go`: `ErrBWItemNotFound` (wrap when `bw get item` reports
  not-found), `ErrBWFieldNotFound` (wrap `fieldFromItem`'s field-missing errors). Tests
  first.
- [ ] `BWCreator` seam + `BWPut.CreateItem(item, field, value)` (minimal login template
  → `setItemField` RMW → base64 → `bw create item`). Reuses `setItemField` (DRY).
- [ ] `set` command core (`runSet`): resolve item+field from the secret's `bw:` block,
  disambiguate `[var]`, idempotency read-back, write/create/dry-run, empty refusal,
  per-shape newline normalization. Inject `bwWriter` (composite) like `bwReader`.
- [ ] Value input: stdin when piped; hidden TTY prompt via `golang.org/x/term`
  otherwise (the hidden-prompt branch is live/manual, like `BWPut`'s shell-out).
- [ ] Wire `newSecretsSetCmd` into `newSecretsCmd`; update the `secrets` long help.
- [ ] Tests: idempotent no-op, write-on-change (env trim / file exact), disambiguation,
  create gated (`--yes` / no-`--yes` / locked), empty refused, `--dry-run` inert.

## Closing

- [ ] Every AC covered by ≥1 test (hidden TTY prompt is live-smoke, like `BWPut`)
- [ ] `features.json` carries non-vacuous verification commands
- [ ] `go test ./internal/...` green; `go vet` + `gofmt` clean; `go build ./...` ok
- [ ] Additive (no existing behaviour changed); `registry.yaml` untouched
- [ ] `go mod tidy` run; `golang.org/x/term` recorded in go.mod/go.sum
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec + #612 C3
