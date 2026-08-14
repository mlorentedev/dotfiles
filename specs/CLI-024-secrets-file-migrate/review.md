---
spec: "CLI-024-secrets-file-migrate"
verdict: "PASS WITH GAPS"
reviewed_sha: "d284b83c0f41844972055f95c9d8ec620b254cff"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-14"
---

## Adversarial review

**Scope**: CLI-024-secrets-file-migrate / PR #965
**Sources**: specs/CLI-024-secrets-file-migrate/{proposal,tasks,verification}.md, PR #965 diff (commit d284b83), supporting source files

### Spec and task alignment

The spec (proposal.md) describes a clean, tightly scoped change:
- **AC1**: five file secrets migrate byte-exact age→bw, parity-gated, registry flipped
- **AC2**: ZOHO_RECOVERY_CODE still fails, surfacing the real #962 blocker (not a new error)
- **AC3**: dedicated byte-exact end-to-end test via `migrateExec`, no-trailing-newline fixture
- **AC4**: stale `#612 M3/M6` comment corrected
- **AC5**: `dotf secrets verify` returns 33/33

The code diff (one commit, d284b83) aligns with the spec structurally:
- `assertMigratable`/`migrateGuard` relaxed to allow file secrets
- `ageValue` threaded on `isFile`: no-trim for files, existing TrimRight for env
- `isFile` threaded into `applySet` and parity gate's `normalizeValue`
- stale comment fixed, spec files added, registry.yaml flipped for 5 secrets

However, the key acceptance criterion (AC3) is undermet by the test it produced (see findings). The code is correct — every path I traced is sound — but the verification artifact does not prove what the spec claims it proves.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | AC3 test | `TestSecretsMigrate_FileSecret_ByteExact` uses a fixture with NO trailing newline (`"apiVersion: v1\nclusters:\n...\nkind: Config"`). This makes the `isFile` toggle in `ageValue` (the trim/no-trim branch) and in the parity gate's `normalizeValue(back, isFile)` untestable: both `strings.TrimRight(value, "\r\n")` and `normalizeValue(back, false)` are identity operations on a string with no trailing `\r`/`\n`. A regression that reverted either `isFile` branch to the old hardcoded false would pass this test. The test's own docstring claims "a regression that trims OR one that appends is equally caught" — this is **false for the trim direction**. | TestSecretsMigrate_FileSecret_ByteExact (name, but fixture is wrong for the claim) | tests: change fixture to include a trailing newline (e.g. `"...kind: Config\n"`), or add a separate tamper-based file-secret parity-abort test via `TestSecretsMigrate_ParityMismatchAbortsBeforeFlip`-type pattern with `isFile=true` |
| Minor | REAL | AC2 wording | AC2 says ZOHO_RECOVERY_CODE "still fails with the pre-existing zoho item-name ambiguity error". Pre-PR, `migrate ZOHO_RECOVERY_CODE` failed at migrateGuard with `"ZOHO_RECOVERY_CODE" is a file secret — file migration is a follow-up`. Post-PR, the guard is removed and it fails at `bwReader.Field("zoho", ...)` → `More than one result was found`. The zoho ambiguity IS pre-existing (affects `secrets set` and `secrets migrate` for env-secret ZOHO_APP_PASSWORDS), but the migrate command's error for **this specific secret** changed. The change is correct behavior (surfaces the real #962 blocker), but the wording overstates the before-state. | — (AC2 verification in verification.md relies on live run; the diff itself proves the before/after error change by having removed the guard block) | spec: reword AC2/verification to say "surfaces the pre-existing #962 zoho ambiguity error" rather than "still fails with" |
| Minor | REAL | Spec hygiene | `proposal.md` frontmatter still `status: draft` — should be `verifying` (verification.md is filled, the feature is implemented). `tasks.md` box `[ ] PR opened referencing this spec folder` is unticked, yet PR #965 exists and references this spec. | Diff: proposal.md line `status: draft # draft \| implementing \| verifying \| archived`; tasks.md line `- [ ] PR opened...` | spec: set status → verifying, tick the PR box |
| Minor | THEORETICAL | Parity gate | The parity gate compares bw read-back vs age plaintext. If bw's store AND get both normalize identically (e.g., `bw edit` + `bw get` both strip trailing newline for the field type), parity would pass while bw diverges from the age plaintext. THEORETICAL — no evidence that the Bitwarden CLI behaves this way, and live SHA-256 spot-checks on KUBECONFIG + CHATGPT_RECOVERY_CODE counter-indicate it. The other 3 secrets (GMAIL_BACKUP_CODE, CHATGPT_BACKUP_CODE, STRIPE_BACKUP_CODE) were parity-gated at migrate time but not SHA-256 spot-checked; their single-line content makes a trailing-newline mismatch the only plausible divergence vector. | — | tests: add a test that demos the parity gate FILTERS the both-sides corruption case (difficult in fakeWriter — would need a tamper that corrupts identically on both store and read; not required before archive) |
| Question | REAL | assertMigratable relaxation | `checkBwSources` correctly validates that a file secret's bw target includes a `field` (line ~288: `if s.BW.Field == ""`). No multi-var file secret is representable (validation enforces env XOR file; `FileExpose` has one `Var`). The relaxed guard is safe. | diff: assertMigratable guard change in registry_write.go; checkBwSources already validates file-secret field | none required — resolved by examining checkBwSources |
| Question | REAL | Pre-existing debt | migateGuard's shared-age-source error says `use migrate --split (C9)`, but the migrate command has no `--split` flag (only `--dry-run`, `--yes`). Pre-existing, not introduced by this PR. | `secrets_migrate.go` line 136 vs flags (lines for dryRun/assumeYes only) | vault/00_meta/... — pattern/ticket for the pre-existing inconsistency |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B  | Code is correct — every path (ageValue isFile branch, parity gate normalizeValue, assertMigratable relaxation, checkBwSources field validation, unique-age guard) is soundly wired. No code defect. |
| Verification       | C  | AC3's test does not exercise the isFile branch (no-trailing-newline fixture makes both trim and no-trim paths equivalent). Parity gate isFile threading untested for file secrets. Live verification claimed but no repo-captured output. |
| Scope              | A  | Single logical change: relax guard, thread isFile, fix comment, flip registry. No scope creep. Spec additions clean. |
| Reliability        | B  | Parity gate guards write correctness. Rollback documented (keep .secret.age files, flip backend back). Idempotent. Both-sides-corruption risk is theoretical; live SHA-256 check covers 2/5. |
| Maintainability    | B  | Minimal diff (6 files touched + 3 spec files). Doc comments well-rewritten. Test docstring contains a factual error ("a regression that trims... is equally caught"). |
| Handoff-readiness  | B- | Two small hygiene gaps (status still draft, tasks box unticked). Spec folder is in the PR. Next-actor actions are clear (archive after review). |

### Verdict

**PASS WITH GAPS**

One C (Verification) — the AC3 test fixture is structurally insufficient to prove the isFile branching that is the core of this change. No D (no code defect found).

Gaps to close before archive:

1. **Fix the AC3 test fixture** — add a trailing newline to the fixture string. `kubeconfig := "...kind: Config\n"`. This makes both the `ageValue` no-trim branch and the parity gate's `normalizeValue(back, true)` behavior regression-detectable: if a future change reverts either to the old hardcoded `false`, the parity gate would compare trimmed-back ("...Config") against raw-value ("...Config\n") and abort the migration — the test's `t.Fatal(err)` would catch it. Alternatively, add a separate `TestSecretsMigrate_FileSecret_ParityAbortsOnMismatch` analogous to the env-side parity test.

2. **Spec hygiene** — set `proposal.md` frontmatter `status: verifying`; tick the `[ ] PR opened` box in `tasks.md`.

3. **AC2 wording** — consider whether `verification.md` should note that the error surfaced is the correct #962 blocker (not "pre-existing for this command").

### Recommended next steps (before archive)

1. Fix `TestSecretsMigrate_FileSecret_ByteExact` fixture to include a trailing newline (the simplest one-line fix that retroactively proves the isFile branching).
2. Optionally add `TestSecretsMigrate_FileSecret_ParityMismatchAbortsBeforeFlip` using a `tamper` on the fakeWriter (analogous to the env path's existing test).
3. Tidy spec frontmatter status and tasks checklist.
4. **`dotf spec archive CLI-024-secrets-file-migrate` is NOT advisable in current state** — the verification gap in AC3 means the core behavioral change (isFile branching) would not be caught by existing tests on regression. Fix the fixture first, re-run the suite, then archive.