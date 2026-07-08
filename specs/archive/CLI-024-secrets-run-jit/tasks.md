---
tags: [spec, tasks, secrets, jit]
created: "2026-06-25"
---

# Tasks - CLI-024-secrets-run-jit

> TDD order. One task = one focused commit. Scope = Phase 1a (the primitive) only.

## Setup

- [x] Branch `feat/secrets-run-jit` from `main`; #493 self-assigned (→ In Progress)
- [x] #493 commented with the ADR-028 reconciliation + the 1a/1b split + the opencode/agy blast-radius finding
- [x] `proposal.md` complete

## Implementation

- [ ] **RED**: `internal/secrets/secrets_test.go` — `ParseMapping` (VAR=file, @VAR=file>dest, comments/blanks, ~ expansion); `Loader.EnvFor` with a fake Decryptor (env secrets → KEY=VALUE, `--only` filter, file secrets → 0600 dest + Var=path).
- [ ] **GREEN**: `internal/secrets/secrets.go` — `Entry` type + `ParseMapping`.
- [ ] **GREEN**: `internal/secrets/resolve.go` — `Decryptor` seam (default shells `age --decrypt --identity`), `Loader.EnvFor`.
- [ ] **GREEN**: `internal/cmd/secrets.go` — `dotf secrets run [--only …] -- <cmd>`; `--` via `ArgsLenAtDash`; child exec with merged env + stdio; exit-code propagation. Split the exec into a testable `runChild`.
- [ ] **GREEN**: wire `newSecretsCmd()` into `root.go`.
- [ ] `go test ./internal/secrets/... ./internal/cmd/... && go build ./...` green.

## Closing

- [ ] `verification.md` with evidence (test output + a `dotf secrets run -- printenv` smoke proving the value is absent from the parent shell)
- [ ] PR opened referencing this spec + #493 (no auto-merge); body states it does NOT remove the ambient export (1b)
- [ ] 1b (retire ambient export + migrate opencode/agy) tracked for the next PR
