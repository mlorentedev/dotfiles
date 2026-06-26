---
tags: [spec, verification, secrets, cli, go, hardening]
created: "2026-06-26"
---

# Verification - CLI-024-secrets-fail-loud

## Evidence

- [x] **AC1** (empty → fail fast; non-login item → error) — PASS:
  `TestEnvFor_BwEmptyValue_FailsFast`, `TestFieldFromItem_NonLoginItem_Errors`,
  `TestSecretsShow_EmptyValue_Errors`.
- [x] **AC2** (render surfaces the specific error; absent quiet; `--strict` non-zero)
  — PASS: `TestRender_AbsentSecret_QuietMissing` (Missing, exit 0, intact),
  `TestRender_UnresolvedDecryptError_LeftIntactWithError` (Unresolved carries the
  real "no identity matched" error), `TestRender_EmptyValue_Unresolved`,
  `TestSecretsRender_Strict_NonZeroOnFailure` (default exit 0, `--strict` errors).
- [x] **AC3** (explicit `--only` → zero = error) — PASS:
  `TestResolveOnly_EmptyTokens_Errors` (`,` and `  ,  `).
- [x] **AC4** (suites green, vet/fmt clean, existing tests updated) — see below.

## Test status

- `go test ./internal/secrets/ ./internal/cmd/ -count=1` → **ok** (both). Targeted
  fail-loud run: all 11 new/updated cases PASS.
- `go vet ./...` → clean. `go build ./...` → clean. `gofmt -l` on staged (LF) blobs
  → clean for every changed file.
- Updated existing test: `TestRender_UnresolvedDecryptError_LeftIntact` →
  `...WithError` (asserts the surfaced error + `[]UnresolvedVar` shape).
- No regressions: full `go test ./...` green **except** `internal/mem`
  `TestVaultHealth/...clean_stub`, which **fails identically on clean main**
  (environment-dependent vault-health stub on this Windows box) — untouched here.

## Decisions made during implementation

- **Absent vs misconfig is classified at the resolver, not by parsing stderr.**
  `AgeDecrypt` `os.Stat`s the file and returns `ErrSecretAbsent` when missing; the
  injected fake decryptor can simulate either case. bw errors are all "real" (a
  registry-declared bw secret absent from the vault *is* a misconfiguration).
- **Empty is a hard error for both shapes** (env and file). None of the registry
  secrets are legitimately empty; an explicit `allowEmpty` flag is the future path if
  one ever is — never a silent default.
- **render default stays non-fatal** (setup completes when a secret is genuinely
  absent); only `--strict` is fatal, and only on real failures (not absence).
  Misconfig is always loud on stderr regardless of `--strict`.
- **`bwItem.Login` became a pointer** so "no login block" is distinguishable from "empty
  password" — the former errors here, the latter is caught by the empty-value guard.
- A4 (`stripBackendAuth`, #610) merged independently while this was in flight.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? **no** — the "secrets must fail loud" principle is
  captured in the audit issue #612 + this spec.
- [ ] ADR-worthy? **no** — implements ADR-028; no new architectural decision.
- [ ] New cross-project pattern? **no** — repo-specific.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/CLI-024-secrets-fail-loud/`
- [ ] #612 Phase A items A1–A3 checked off; PR linked
- [ ] Promotions above executed (if any)
