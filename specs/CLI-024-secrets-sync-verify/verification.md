---
tags: [spec, verification, secrets, sync, cli, go]
created: "2026-06-26"
---

# Verification - CLI-024-secrets-sync-verify

## Evidence

- [x] **AC1** (dead token aborts before upload) — PASS:
  `TestSecretsSyncCi_DeadTokenAbortsBeforeUpload` — a failing validator returns an error
  containing "liveness check", the fakeSetter is empty (nothing uploaded), and the
  validator was called exactly once.
- [x] **AC2** (live token validates then uploads) — PASS:
  `TestSecretsSyncCi_LiveTokenUploads` — value uploaded, output carries `verified RELEASE_TOKEN`.
- [x] **AC3** (escape hatch + opt-in) — PASS:
  `TestSecretsSyncCi_SkipVerifyBypassesValidation` (`--skip-verify` → no validator call,
  still uploads) and `TestSecretsSyncCi_UnmarkedEntrySkipsValidation` (unmarked entries
  never call the validator).
- [x] **AC4** (schema + suite) — PASS: registry parses `validate`; `SetBackendBW` golden
  against the real `registry.yaml` still passes with the two added `validate:` lines.

## Test status

- `cd cli && go test ./internal/secrets/ ./internal/cmd/ -count=1` → **ok**.
- `go vet ./...` clean; `go build ./...` clean; `gofmt -l internal/secrets internal/cmd` empty.

## Decisions made during implementation

- **Pre-upload gate, not post-upload.** "Don't upload broken credentials" means the dead
  token must never reach Actions — so the liveness pass runs over all marked entries
  *before* the upload loop and aborts the whole sync on the first failure.
- **Opt-in per entry, GitHub-only.** Liveness validation does not generalize across
  providers (each is a bespoke probe, or none). A registry marker keeps it explicit and
  avoids a per-provider tar pit; unmarked secrets are untouched.
- **Authenticate AS the token under test.** `GHTokenValidate` strips inherited
  `GH_TOKEN`/`GITHUB_TOKEN` before injecting the probed value, so the check can't pass on
  the ambient `gh auth`.
- **No "skip identical" guard.** Actions secrets are write-only (no readback to compare),
  and a re-upload is idempotent and harmless — the incident was a *dead* token, not a
  duplicate one.
