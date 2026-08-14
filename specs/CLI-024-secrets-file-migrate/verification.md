---
tags: [spec, verification, templates]
created: "2026-08-14"
---

# Verification - CLI-024-secrets-file-migrate

## Evidence

- [x] AC1 (file secrets migrate byte-exact) -> test `TestSecretsMigrate_FileSecret_ByteExact` + `TestSetBackendBW_FileSecret`; live: `dotf secrets migrate` run for `KUBECONFIG`, `GMAIL_BACKUP_CODE`, `CHATGPT_BACKUP_CODE`, `CHATGPT_RECOVERY_CODE`, `STRIPE_BACKUP_CODE` (all succeeded, all flipped to `backend: bw`)
- [x] AC2 (ZOHO_RECOVERY_CODE surfaces the real #962 blocker) -> live: `dotf secrets migrate ZOHO_RECOVERY_CODE --dry-run` now reaches the `bw get item "zoho": More than one result was found` error (#962) — before this PR it failed earlier, at `migrateGuard`'s blanket file-secret rejection; removing that guard is what lets the pre-existing zoho ambiguity surface for this command, correctly, not a new bug
- [x] AC3 (byte-exact end-to-end test) -> test `TestSecretsMigrate_FileSecret_ByteExact` (multi-line fixture WITH a trailing newline, like `age -d` appends; asserts the bw-written value is exactly the fixture, neither trimmed nor appended — a fixture with no trailing newline would make `TrimRight` a no-op regardless of `isFile`, so the trailing newline is what makes the `isFile` branching regression-detectable)
- [x] AC4 (stale #612 M3/M6 comment fixed) -> `registry_write.go` doc comments on `SetBackendBW` and `assertMigratable` rewritten; `grep -rn 'M3/M6' cli/internal/secrets/registry_write.go` returns nothing
- [x] AC5 (verify 33/33 OK) -> `dotf secrets verify`: `33 ok, 0 missing, 0 failed` after the 5 migrations; `ZOHO_APP_PASSWORDS`/`ZOHO_RECOVERY_CODE` still `age` (OK, expected — blocked on #962)

## Test status

- Test suite: `cd cli && go build ./... && go vet ./... && go test ./... -count=1` -> all packages `ok`, 0 failures
- Lint: `golangci-lint run` -> `0 issues`
- Manual smoke test (live Bitwarden vault, this operator's unlocked session):
  - `dotf secrets migrate <id> --yes` for the 5 non-Zoho file secrets — each created or updated its declared bw item/field and flipped the registry
  - SHA-256 parity spot-check (plaintext never printed) on `KUBECONFIG` (multi-line, dedicated item) and `CHATGPT_RECOVERY_CODE` (custom field on an item shared with `CHATGPT_BACKUP_CODE`): age-side hash == bw-side hash on both
  - `dotf secrets migrate ZOHO_RECOVERY_CODE --dry-run` confirmed to still fail with the pre-existing, unchanged error
- No regressions in existing test suite: yes — full suite green before and after, including the pre-existing env-secret migrate tests

## Decisions made during implementation

- `STRIPE_BACKUP_CODE`'s registry entry carried an unresolved inline `# GROUP?: own item, or a field on stripe-api-key's item` comment predating this spec. Resolved via user confirmation: ship as-declared (own dedicated bw item) — zero registry change beyond removing the stale comment, migrates in this PR. Consolidating onto `stripe-api-key` is left as a future follow-up, not part of this spec.
- `BWTarget`'s third return value (`isFile`) was already correctly computed for file secrets but discarded (`_`) at both call sites in `secrets_migrate.go` — no new field was needed, just threading the existing value through to `ageValue`, `applySet`, and the parity gate's `normalizeValue` call.

## Adversarial review (nan/deepseek-v4-flash, PASS WITH GAPS → resolved)

`specs/CLI-024-secrets-file-migrate/review.md` (reviewed sha `d284b83`) found one Major, REAL finding: `TestSecretsMigrate_FileSecret_ByteExact`'s original fixture had no trailing newline, making `strings.TrimRight` a no-op regardless of whether `isFile` was actually threaded — a regression reverting either the `ageValue` or parity-gate `isFile` branch to the old hardcoded trim would have passed silently. Verified empirically (a standalone Go snippet confirmed `TrimRight` is identity on the no-newline fixture, non-identity with one) then fixed: the fixture now carries a trailing newline. Re-verified by two mutation tests — hardcoding `isFile=false` at each of the two call sites in turn and confirming the test fails both times, then reverting — before re-running the full suite (all green) and pushing the fix.

Two Minor findings were also resolved: `proposal.md`'s `status:` was still `draft` (now `verifying`), and AC2's wording implied "no error changed" when the actual, correct behavior is that the file-secret guard's removal is what lets the pre-existing #962 zoho ambiguity surface for `ZOHO_RECOVERY_CODE` for the first time — reworded in `proposal.md`, `tasks.md`, and here to state that precisely.

Two Question-class findings did not require a code change: `checkBwSources` already validates a file secret's `bw.field` is non-empty, confirming the relaxed `assertMigratable` guard cannot admit an unresolvable file secret; and the pre-existing `migrate --split (C9)` error-message reference to a flag that does not yet exist is unrelated to this PR (tracked under #612 C9 already).

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? No — mechanical extension of an already-established design (byte-exact vs. trimmed contract was already codified in `normalizeValue`), not a new pattern.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? No — implementation detail within ADR-028's existing scope.
- [ ] New pattern candidate for `00_meta/patterns/`? No — single-repo, single-occurrence.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-024-secrets-file-migrate/` -> `specs/archive/CLI-024-secrets-file-migrate/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
