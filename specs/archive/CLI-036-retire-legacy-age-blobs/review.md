---
spec: "CLI-036-retire-legacy-age-blobs"
verdict: "PASS"
reviewed_sha: "792b181ae89b15d790404be4311b4b6e3761c1a7"
reviewer: "nan/mimo-v2.5"
date: "2026-09-26"
---

## Adversarial review

**Scope**: CLI-036-retire-legacy-age-blobs (spec + 3 commits, base `dba85a23a79446d9783ad206c5b7efc206fae1b6`)
**Sources**: `specs/CLI-036-retire-legacy-age-blobs/{proposal,tasks,verification,features}.json`, `git diff dba85a23...HEAD`, `go test`, `bats`, `golangci-lint`

### Spec and task alignment

All six acceptance criteria are addressed in the diff:

- **AC1** — `TestCommittedAgeBlobsAreClaimed` (new, `cli/internal/secrets/committed_blobs_test.go`) globs `sensitive/*.secret.age`, checks each against age-backed registry entries, fails with 31 names before `git rm`, green after. The 31 blobs are removed from HEAD. **Verified:** test passes, `git ls-files` returns only `id_ed25519.secret.age`.
- **AC2** — `TestCheckSecrets_BwBackedEntriesAreNotAgeAsserted` updated: the unclaimed `chatgpt.api-key.secret.age` is now a FAIL naming it, and the output carries no `#971`. The old WARN path (`reportUnreferencedBlobs`) is deleted. **Verified:** test passes, 3 failures (AGEMISS + OFFLINE + orphan).
- **AC3** — `TestCheckSecrets_FixPrunesUnclaimedMirrorBlob` (new): without `--fix`, 2 FAILs and no disk change; with `--fix`, 0 FAILs and 2 `[FIX]` lines; a second run finds nothing to prune. **Verified:** test passes.
- **AC4** — `TestCheckSecrets_FixRefuses` (new, 4 rows): blob still in checkout, no DR escrow, no registry, mirror==checkout. Each row: files survive, blob still FAILs. The "files that are not blobs" case (`env-mapping.conf`, `README.md`, `dr/`) is asserted in AC3's test (they survive `--fix`). **Verified:** all 4 rows pass.
- **AC5** — ADR-028 amendment appended (26 lines): the escrow is the floor; per-secret blobs exist only for `age-offline`; #971 superseded. `pruneOrReportOrphans` carries the corrected comment. **Verified:** reads correctly, no stale references.
- **AC6** — Manual verification on msi recorded in `verification.md`: 32 pruned lines (31 blobs + 1 `.tmp`), second run clean, `dotf secrets drift` reports 0 findings.

All implementation tasks (`[x]`) in `tasks.md` have corresponding diff evidence. The two unchecked closing tasks (`go build/vet/test/lint` and PR body) are closing-phase items, not implementation gaps — both verified passing in this review.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | security | `pruneOrReportOrphans` checks `pathExists` then calls `os.Remove` — a TOCTOU window exists. If another process creates a symlink at the blob path between the check and the remove, `os.Remove` would remove the symlink target. Practically unreachable: doctor runs once, the mirror is `~/.dotfiles/sensitive/`, and no concurrent writer is expected. | code read of `pruneOrReportOrphans` | `TestCheckSecrets_FixPrunesUnclaimedMirrorBlob` (covers the happy path, not the race) | — (surface only; do not gate) |
| Minor | THEORETICAL | edge-case | `secretBlobCandidates` uses `os.ReadDir` which does not follow symlinks, and `IsRegular()` rejects symlinks. If a symlink with a `.secret.age` suffix exists in the mirror, it would be picked up as a candidate and then `os.Remove` would remove the symlink itself, not its target. Correct behavior for a symlink, but undocumented. | code read | `TestCheckSecrets_FixPrunesUnclaimedMirrorBlob` (no symlink in fixture) | — (document if desired) |
| Minor | THEORETICAL | scope | `TestCheckSecrets_FixRefuses` does not include a row for `*.secret.age.tmp.*` still in the checkout — AC4 only says "the file is not `*.secret.age` / `*.secret.age.tmp.*`" (for non-blob files), and the blob-in-checkout row tests the `*.secret.age` case. A `.tmp.*` file in the checkout would be an orphan (never claimed) and would be pruned — this is correct but untested. | code read | UNTESTED | tests (add row if desired) |

All findings are THEORETICAL and Minor. No Blockers or Majors found.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 6 ACs met; negative paths (refusal, idempotency, non-blob survival) covered and tested. |
| Verification       | A | 5 feature-level verifications in `features.json`, all `state: passing`; named test functions in evidence; mutation test (escrow gate dropped) recorded. |
| Scope              | A | Diff matches proposal exactly: 31 blob removals, doctor fix logic, ADR amendment, 3 consumer moves (rollout script, CI, READMEs), lesson. No creep. |
| Reliability        | B | Error paths handled (os.Remove failure reported, refusal messages actionable). Minor: TOCTOU window, undocumented symlink edge case. |
| Maintainability    | A | `pruneOrReportOrphans` 30 lines, `secretBlobCandidates` 16 lines. Clear naming. Comment explains WHY (ADR-028 §5, #802, #971). |
| Handoff-readiness  | A | ADR amendment committed, lesson 308 captured, runbook/READMEs corrected, tasks.md and verification.md complete. |

**Rubric aggregate:** All A or B → **PASS**.

### Verdict
PASS

### Recommended next steps

- **(Spec/verification)** The unchecked closing task in `tasks.md` (`go build ./... && go vet ./... && go test ./...`, `golangci-lint run`, `GOOS=windows go vet ./...`) was verified passing in this review session. Tick it or record it in `verification.md`.
- **(Spec)** The PR body should close #938, #971, and #802 and carry a `## Review triage` section per Standing Order #8. Record in `tasks.md`.
- **(Optional, tests)** Add a `TestCheckSecrets_FixRefuses` row for a `.secret.age.tmp.*` file still in the checkout to exercise the `.tmp.*` path under the "blob in checkout" refusal.
- **`dotf spec archive` is advisable** once the two closing tasks are ticked. No contract-set changes are required.
