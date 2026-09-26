---
spec: "GUARD-005-review-verdict-provenance"
verdict: "PASS WITH GAPS"
reviewed_sha: "81d0b5193e1f18285bb92d5298749eaa013cb6f1"
reviewer: "nan/mimo-v2.5"
date: "2026-09-25"
---

## Adversarial review

**Scope**: GUARD-005-review-verdict-provenance (`git diff bb9b99b89828f081cfde0ef0b6d8f973fe62b641...HEAD`)
**Sources**: `specs/GUARD-005-review-verdict-provenance/{proposal,tasks,features,verification}.md`, diff of 8 files (+680 lines), `go test ./... -count=1`

### Spec and task alignment

All 9 acceptance criteria are ticked in `tasks.md` and all implementation tasks are checked. Each AC maps to at least one named test or a documented end-to-end run:

| AC | Covered by | Status |
|----|------------|--------|
| AC1 (sidecar written before reviewer starts) | End-to-end in `verification.md` | Verified (e2e) |
| AC2 (no-verdict → refused, naming that cause) | `TestProvenanceCatchesAReviewThatWroteNothing` (f1) | PASS |
| AC3 (sha mismatch → refused, contrasting claim vs measurement) | `TestProvenanceCatchesAVerdictOnAnotherSha` (f3) | PASS |
| AC4 (different pool member → refused) | `TestProvenanceCatchesADifferentPoolMember` (f4) | PASS |
| AC5 (no sidecar → guard not asserted) | `TestProvenanceIsSilentWithoutASidecar` (f5) | PASS |
| AC6 (unparseable sidecar → loud error) | `TestUnparseableSidecarIsLoud` (f6) | PASS |
| AC7 (first review → empty digest) | `TestWriteReviewRequestRecordsTheDigestOfWhatWasThere/with_no_previous_review` (f7 subtest) | PASS |
| AC8 (write failure → warning, not blocking) | `spec.go:217-219` warns; no test | **UNTESTED** |
| AC9 (end-to-end against real binary) | `verification.md` console evidence | Verified (e2e) |

All 7 `features.json` verifiers execute and exit 0 against the installed test suite. Full `go test ./... -count=1` passes across all packages (14 packages, 0 failures).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|-------------|
| Minor | THEORETICAL | digest-check | The digest comparison treats "identical content across rounds" as "no verdict written". A reviewer that genuinely produces byte-identical content to a previous round would be refused. The error message ("the reviewer wrote no verdict") is misleading in that case. The risk is negligible because two rounds producing the same file is extremely unlikely, and the fix (`--force-without-review`) is already named in the error. | Code read of `checkReviewProvenance` — the `req.ReviewDigestBefore != "" && req.ReviewDigestBefore == fileDigest(...)` condition. | `TestProvenanceCatchesAReviewThatWroteNothing` exercises the "content unchanged" path (matching digest), but does not separately test the "content intentionally identical" variant. | code (if desired: add a second digest field for the after-state, or accept the false positive as documented) |
| Minor | THEORETICAL | test-coverage | AC8 ("sidecar write failure warns and lets the review launch") is documented in `spec.go:217-219` and in `verification.md` decisions, but has no unit test. The warning path is reached only when `os.WriteFile` fails (e.g. read-only spec dir), and no test injects that fault. | `spec.go:217-219` PrintErrf on WriteReviewRequest failure; grep of `*_test.go` shows no coverage of the error path. | UNTESTED | tests (add a test that makes specDir read-only and verifies the warning is emitted and the function returns nil) |
| Minor | THEORETICAL | sidecar-interop | The on-disk `review-request.json` contains two fields (`base_sha`, `contract_digests`) not defined in the `ReviewRequest` struct. `json.Unmarshal` silently ignores them, so this is harmless today — but it indicates the launcher may be writing fields the consumer does not read, which is a latent drift surface. | `cat specs/GUARD-005-review-verdict-provenance/review-request.json` shows 6 fields; `ReviewRequest` struct has 4. | N/A (observation) | code (if these fields are intentional, add them to the struct; if not, verify the launcher does not write them) |

No Blockers. No REAL Majors. The three findings are all THEORETICAL Minors that do not affect correctness of the core guard behavior.

### Evaluator rubric

| Dimension | Grade | Rationale |
|-----------|-------|-----------|
| Correctness        | A | All 9 ACs met; digest-first ordering prevents misleading error messages; absent sidecar degrades gracefully; damaged sidecar fails loud. |
| Verification       | B | 7 named tests covering all refusal paths + the happy path + the absent/damaged sidecar cases, plus e2e in verification.md. AC8 (write failure warning) has no test — drops from A. |
| Scope              | A | Diff is exactly the sidecar type, its read/write, the provenance check, the launcher wiring, and the spec artifacts. No unrelated changes. |
| Reliability        | A | Error paths handled for missing sidecar, damaged sidecar, empty fields, write failure. Sidecar write failure degrades to "not asserted" rather than blocking launch. |
| Maintainability    | A | Functions ≤40 lines, clear naming, thorough comments explaining the "why" for every non-obvious choice. The ordering comment in `checkReviewGate` is exemplary. |
| Handoff-readiness  | A | Proposal, tasks, verification, and features.json all complete and accurate. No promotion candidates needed per verification.md decisions. |

### Verdict

**PASS WITH GAPS**

Three THEORETICAL Minors (digest false-positive on identical content; no test for the write-failure warning path; extra sidecar fields from launcher). None are Blockers or REAL Majors. The core guard behavior — detecting a no-verdict run, a sha mismatch, and a wrong-pool-member signature — is thoroughly tested and correct.

### Recommended next steps

1. **[tests, Minor]** Add a test for the sidecar write failure path (AC8): make `specDir` read-only, call `WriteReviewRequest`, verify the error is returned (the launcher handles it with a warning). This closes the UNTESTED gap.
2. **[code, Minor]** Optionally add `base_sha` and `contract_digests` to the `ReviewRequest` struct if the launcher intends to write them, or verify the launcher does not produce them. The silent ignore is safe but obscures a potential drift.
3. **[spec, Minor — no action required]** The digest false-positive on identical content is documented, extremely unlikely, and has an explicit escape (`--force-without-review`). No fix needed unless the probability assessment changes.

`dotf spec archive` is **advisable** in the current state: no Blockers, no REAL Majors, all core ACs verified, and the gaps are tracked. The two code-level items (1, 2) are follow-ups, not archive gates.
