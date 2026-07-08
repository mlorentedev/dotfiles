---
tags: [spec, verification, secrets, cli, go]
created: "2026-06-26"
---

# Verification - CLI-024-secrets-verify

## Evidence

- [x] **AC1** (`Loader.Verify`: no materialize, no leak, correct classification) — PASS:
  `TestLoader_Verify_Classification` (ok/absent/empty/error subtests),
  `TestLoader_Verify_FileSecret_NotMaterialized` (asserts the Dest file is NOT written),
  `TestLoader_Verify_Bw` (ok / empty / missing item).
- [x] **AC2** (`verify` reports statuses, no values, exit non-zero on FAILED) — PASS:
  `TestSecretsVerify_ReportsStatuses_NoValues` (OK/MISSING/FAILED all present, the
  secret value absent from output, exit non-zero).
- [x] **AC3** (`verify <id>` scopes; unknown id errors) — PASS:
  `TestSecretsVerify_ScopesById`, `TestSecretsVerify_UnknownId_Errors`.
- [x] **AC4** (suites green, vet/fmt clean, additive) — see below.

## Test status

- `go test ./internal/secrets/ ./internal/cmd/ -count=1` → **ok** (both). Targeted
  `-run Verify`: all 6 cases PASS.
- `go vet ./...` → clean. `go build ./...` → clean. `gofmt -l` on staged (LF) blobs →
  clean for every changed file.
- Additive only: no existing command/test changed (one new command + one new
  `Loader.Verify` method + new test file).
- Live smoke against the real registry deferred to the operator (it resolves real age/bw
  secrets — needs the age key / an unlocked Bitwarden); the unit tests cover the logic
  with fakes (the `Decryptor`/`BWReader` seams), no environment dependency.

## Decisions made during implementation

- **`Verify` reuses the resolver, not `EnvFor`** — so it gets identical resolution +
  empty-value semantics as `run`, but skips materialization (the side effect lives in
  `EnvFor`, not the resolver). No file written, value discarded → no leak.
- **MISSING is tolerated by default** (a machine need not hold every secret); only
  FAILED forces a non-zero exit. `--require-all` tightens MISSING into a failure for
  callers that want full provisioning asserted.
- **Scoping reuses `resolveOnly`** (the `--only` selector) so `verify <id>` and
  `run --only <id>` agree on what an id/var selects; an unknown id errors there.
- **bw errors are all FAILED** (a registry-declared bw secret absent from the vault is
  a misconfiguration, not "absent on this machine" — only age files can be MISSING).

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? **no**.
- [ ] ADR-worthy? **no** — implements ADR-028 / #612 Phase C.
- [ ] New cross-project pattern? **no**.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/CLI-024-secrets-verify/`
- [ ] #612 Phase C item C1 checked off; PR linked
- [ ] Promotions above executed (if any)
