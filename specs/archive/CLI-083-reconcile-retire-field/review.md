---
spec: "CLI-083-reconcile-retire-field"
verdict: "PASS"
reviewed_sha: "2fc33d35605a84f75546a351bfeba179512ceb85"
reviewer: "nan/mimo-v2.5"
date: "2026-09-24"
---

## Adversarial review

**Scope**: CLI-083-reconcile-retire-field (full diff from `ee533070f4af65b2340907d3ba07ece30535bdee` to HEAD)
**Sources**: `specs/CLI-083-reconcile-retire-field/{proposal,tasks,verification,features}.json` + `git diff ee533070f4af65b2340907d3ba07ece30535bdee...HEAD` (14 files, +727/-56)

### Spec and task alignment

All 5 acceptance criteria are implemented and covered by named tests. Every task in `tasks.md` is ticked except the adversarial review and PR (correct — this is the review). `features.json` has 5 verifier commands, all verified passing in this session:

- f1: `TestRegistryValidatesRetiredItems` + `TestRegistryValidatesRetiredFields` — 2 PASS
- f2: `TestPlanRetiredFields` + `TestApplyDeletesARetiredField` + `TestReconcileDeletesARetiredField` — 3 PASS
- f3: `TestBWFromOnAMultiVarSecret` + `TestPlanMultiVarFromPlansOnce` — 2 PASS
- f4: `TestScopeUploadDedupesNames` — 1 PASS
- f5: `TestPlanRetiredFields` + `TestReconcileDeletesARetiredField` — 2 PASS

The runbook update (`docs/runbooks/guide-secrets-governance.md` step 9) accurately reflects the new `delete-field` operation, including the DR-escrow recovery caveat.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | plan output | `PlanRetiredItems` blocked message uses `ri.label()` for field entries, which formats as "Stripe/backup-codes" in the "cannot delete" detail. The underlying issue is item-name ambiguity (2 items named Stripe), not the field. An operator reading the blocked output may briefly wonder why a field name is mentioned in an item-level ambiguity message. The `PlanNote.Item` field holds just "Stripe", and the remedy ("rename or remove the duplicate items") is clear, so this is a clarity nit, not a correctness defect. | `reconcile_dedupe.go:104`: `ri.label()` in the blocked detail format string; the `PlanNote.Item` is set to `ri.Item` (not `ri.label()`) at line 102, so the structured field is correct. | `TestPlanRetiredFields` — the assertion `p.Blocked[0].Item != "Hetzner"` verifies the `Item` field is correct; the detail substring check does not exercise the label format. | tests (add an assertion on the full blocked detail string for a field entry, or accept the current format as informative) |

**Note on AC5 (no value in output):** All test vault values carry `PLANTED` and the tests assert no `PLANTED` substring reaches the output. The mutation battery (21/21 killed) independently verifies this by attempting to expose values through each code path. No gap found.

### Evaluator rubric

| Dimension | Grade | Rationale (one line) |
|-----------|-------|----------------------|
| Correctness        | A | All 5 ACs verified, negative paths covered, no observed defects |
| Verification       | A | 21/21 mutants killed, 5 verifier commands pass, PLANTED leak detection, command-level end-to-end test |
| Scope              | A | Diff matches proposal exactly; 4 commits of clean, focused work; no creep |
| Reliability        | B | Error paths handled (blank field, blank item, ambiguous name, self-destination); idempotent (re-plan after apply reports gone); one THEORETICAL clarity nit on blocked message |
| Maintainability    | A | Clean struct additions, extracted `checkRetiredEntry`, `bwFields()` replaces `bwDestField()` with clear multi-var semantics, all functions ≤40 lines |
| Handoff-readiness  | A | Spec complete, runbook updated (CONVERGE step 9), verification.md filled, promotion candidates evaluated and declined with reason |

### Verdict
PASS

### Recommended next steps
- Disposition the 1 Minor finding in `verification.md` — apply the blocked-detail assertion fix to `TestPlanRetiredFields` as a clarity improvement, or decline with reason (the current format is informative if slightly redundant).
- `dotf spec archive CLI-083-reconcile-retire-field` is **advisable** in the current state. All contract files are complete, the review is fresh, and no blockers or real majors exist.
