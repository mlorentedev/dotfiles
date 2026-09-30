---
spec: "HARNESS-111"
verdict: "FAIL"
reviewed_sha: "0a584d4ebf173fec0e02d77d70134b7712a3bb3b"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-30"
---

## Adversarial review

**Scope**: HARNESS-111
**Sources**: `specs/HARNESS-111/{proposal,tasks,verification}.md`, `specs/HARNESS-111/features.json`, `git diff a76ea184ea20f8f54903df15c7d558074f22e241...HEAD`

### Spec and task alignment
- `proposal.md` and `features.json` (AC3) explicitly require that characters altering the lexicon (accents, section signs) must NOT be folded.
- `proposal.md` and `features.json` (AC6) explicitly require that the cap warning reports BOTH units.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker  | REAL    | normalisation | Code intentionally overrides spec AC3 by folding lexicon-altering characters. | `fold_to_ascii` implements `s/\xc2\xa7/Section /g` and `s/\xc3\xa1/a/g`, and the code comment explicitly justifies it, contradicting `proposal.md` AC3. `features.json` falsely claims they survive. | `@test "doctrine: a capped surface is folded to pure ASCII, marker and preamble included"` | code |
| Major    | REAL    | cap warning | The cap warning when a file is over the limit only prints one unit ("%s characters"), conflating chars/bytes instead of reporting both as required by AC6. | Lines 1317-1324 print `WARN %s: the GENERATED doctrine alone is %s characters` and `%s is %s characters`. | UNTESTED | code + tests |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | D | Implementation actively contradicts AC3 and fails to meet AC6. |
| Verification       | C | Evidence in `features.json` for AC3 makes claims about surviving characters that the code itself actively replaces. |
| Scope              | D | Diff materially diverges from proposal by folding characters the proposal explicitly declared out of bounds. |
| Reliability        | B | `sed` hex escapes are robust and `if` tests prevent `set -e` aborts. |
| Maintainability    | B | Clear naming, fixed translation tables, and complexity is within limits. |
| Handoff-readiness  | C | Implementation overrides the spec without updating the contract files (spec is stale). |

### Verdict
FAIL

### Recommended next steps
- Remove lexicon-altering characters (`\xc2\xa7`, `\xc3\xa1`, etc.) from `fold_to_ascii` in `scripts/compile-harness.sh` to comply with AC3, OR update the contract files (`proposal.md`, `features.json`) to waive it.
- Modify the warnings at the end of `deploy_doctrine` to explicitly print both the `gen_chars`/`gen_bytes` and `chars`/`bytes` values when the cap is exceeded, instead of only `%s characters`, to comply with AC6.
- Add a named regression test asserting the cap warning format includes both units when triggered.
