---
spec: "CLI-078-secrets-layout-drift"
verdict: "FAIL"
reviewed_sha: "c7b912b760b05899ed15fb7da974437952662d53"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-22"
---
## Adversarial review

**Scope**: git diff c52e637cd2e014d5c897853f74841dbdc1061f66...HEAD
**Sources**: specs/CLI-078-secrets-layout-drift/{proposal,tasks,verification}.md

### Spec and task alignment
- AC1, AC2, AC3, AC6, AC7, AC8 are fully met.
- AC4 claims field resolution matches `fieldFromItem` exactly, but there is a subtle divergence for empty notes/usernames.
- AC5 claims deduplication per problem, but misses deduplication for missing fields.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker  | REAL    | correctness | `byName` map silently overwrites items with duplicate names, masking valid managed items behind personal items of the same name. This incorrectly reports valid items as misfiled/missing fields, blocking CI and risking destructive mutation in `reconcile`. | Bitwarden allows duplicate item names. `byName[it.Name] = it` keeps only the last decoded item. | UNTESTED | code + tests |
| Minor    | THEORETICAL | scope | Missing field findings are not deduplicated. Seven vars declaring the same missing field produce seven identical findings, violating AC5 (one finding per problem). | `LayoutDrift` lacks a `seenField` check in the `!hasField(it, d.Field)` branch. | UNTESTED | code + tests |
| Minor    | THEORETICAL | correctness | `hasField` behavior diverges from `fieldFromItem` for empty notes/usernames. `fieldFromItem` returns `("", nil)` (present), but `hasField` returns false (missing), leading to a false finding for intentionally empty fields. | `HasNotes` is `w.Notes != ""` in `decodeItems`, while `fieldFromItem` simply returns `it.Notes, nil`. | UNTESTED | code |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Substantial negative-path gap: duplicate item names cause false positives and math errors. |
| Verification       | B | Evidence covers criteria but missed the duplicate item edge case and missing field dedupe. |
| Scope              | B | Diff mostly matches, but AC5 (dedupe) is partially missed for fields. |
| Reliability        | B | Most error paths handled; read-only guarantees are structurally enforced. |
| Maintainability    | B | Clear naming and structure; logic is well isolated. |
| Handoff-readiness  | A | Spec updates included and decisions recorded in verification.md. |

### Verdict
FAIL

### Recommended next steps
- Fix the `byName` map to handle multiple items per name (e.g. `map[string][]ItemSummary`), and update `LayoutDrift` to consider a declaration satisfied if AT LEAST ONE matching item meets it.
- Add deduplication for missing field findings (e.g., `seenFieldMissing[d.Item + ":" + d.Field]`).
- Clarify in the spec or code whether the `hasField` divergence for empty notes/usernames is intended (and update AC4), or adjust `decodeItems` to distinguish absent from empty.
