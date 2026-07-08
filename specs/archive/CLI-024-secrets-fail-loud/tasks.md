---
tags: [spec, tasks, secrets, cli, go, hardening]
created: "2026-06-26"
---

# Tasks - CLI-024-secrets-fail-loud

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch from main: `fix/secrets-fail-loud-resolution`
- [x] `proposal.md` complete; acceptance criteria testable

## Implementation

- [x] **A1** — `EnvFor` rejects an empty resolved value (env or file) fail-fast;
  `show` inherits it. `fieldFromItem` errors on a typed field against a non-login
  item (`bwItem.Login` → pointer). Tests: `TestEnvFor_BwEmptyValue_FailsFast`,
  `TestFieldFromItem_NonLoginItem_Errors`, `TestSecretsShow_EmptyValue_Errors`.
- [x] **A2** — `ErrSecretAbsent` sentinel; `AgeDecrypt` reports a missing file as
  absent. `render.Result` splits `Missing` (quiet) from `Unresolved []UnresolvedVar`
  (loud, carries the error); `resolve` returns the error (no swallow); the command
  prints the specific error per var and adds `--strict` (non-zero on real failure).
  Tests: `TestRender_AbsentSecret_QuietMissing`,
  `TestRender_UnresolvedDecryptError_LeftIntactWithError`,
  `TestRender_EmptyValue_Unresolved`, `TestSecretsRender_Strict_NonZeroOnFailure`.
- [x] **A3** — `resolveOnly` errors when an explicit `--only` resolves to zero
  secrets. Test: `TestResolveOnly_EmptyTokens_Errors`.
- [x] **A4** — #610 (`stripBackendAuth`) merged independently; the child-env leak is
  closed on main.

## Closing

- [x] Every AC covered by ≥1 test; existing render test updated to the new contract
- [x] `features.json` carries non-vacuous verification commands
- [x] `go test ./internal/secrets ./internal/cmd` green; `go vet` + `gofmt` clean
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec + #612
