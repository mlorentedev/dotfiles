---
tags: [spec, tasks, secrets, cli, go]
created: "2026-06-26"
---

# Tasks - CLI-024-secrets-verify

> TDD order. One task = one focused commit.

## Setup

- [x] Branch from main: `feat/secrets-verify`
- [x] `proposal.md` complete; acceptance criteria testable

## Implementation

- [x] **AC1** — `Loader.Verify(entry)` resolves via the backend resolver without
  materializing files or returning the value; nil on a non-empty secret,
  `ErrSecretAbsent` (wrapped) on absent, a real error otherwise. Tests:
  `TestLoader_Verify_Classification`, `TestLoader_Verify_FileSecret_NotMaterialized`,
  `TestLoader_Verify_Bw`.
- [x] **AC2/AC3** — `newSecretsVerifyCmd`: verify all or `[id...]` (via the `--only`
  selector), report OK/MISSING/FAILED per var+backend, no values, exit non-zero on any
  FAILED (`--require-all` also on MISSING). Registered in `newSecretsCmd`. Tests:
  `TestSecretsVerify_ReportsStatuses_NoValues`, `TestSecretsVerify_ScopesById`,
  `TestSecretsVerify_UnknownId_Errors`.

## Closing

- [x] Every AC covered by ≥1 test
- [x] `features.json` carries non-vacuous verification commands
- [x] `go test ./internal/secrets ./internal/cmd` green; `go vet` + `gofmt` clean
- [x] No existing behaviour changed (verify is additive)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec + #612 C1
