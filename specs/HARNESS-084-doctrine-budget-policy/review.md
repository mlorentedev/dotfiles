---
spec: "HARNESS-084-doctrine-budget-policy"
verdict: "PASS WITH GAPS"
reviewed_sha: "ff48994ae989c114cca33ba2af4053b5f2fe1158"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-10-01"
---

## Adversarial review

**Scope**: HARNESS-084-doctrine-budget-policy
**Sources**: `specs/HARNESS-084-doctrine-budget-policy/proposal.md`, `specs/HARNESS-084-doctrine-budget-policy/tasks.md`, `specs/HARNESS-084-doctrine-budget-policy/verification.md`, `git diff 4d0ffa99f7612797add2eb9d945634ecb376cf8d...HEAD`

### Spec and task alignment
- **AC1-AC3**: Met. The `full-only` regions are correctly filtered out from the compact payload and retained in the full payload. The markers themselves are stripped out.
- **AC4**: Met. `render_region` properly handles `render_region_raw`'s exit code, no longer swallowing the failure if a record is missing.
- **AC5**: Met. The cap is verified to be respected.
- **AC6**: Met. Tests pass and `shellcheck` is clean.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | Correctness | **Same-line marker drops doctrine**: Marker parsing matches whole lines. In the compact render, `awk`'s `/.../ { next }` discards the entire line. In the full render, `grep -v` excludes the line entirely. If a marker shares a line with doctrine content, the text is silently dropped from both surfaces. | Code inspection of `scripts/compile-harness.sh` `render_region` (`grep -v`) and `render_region_compact` (`awk`). | UNTESTED | code + tests |
| Major | THEORETICAL | Correctness | **Cross-file marker bleeding**: `render_region_compact` parses the concatenated stream of all records. An unclosed `full-only:begin` in one record can be "closed" by a stray `full-only:end` in a subsequent record, silently dropping all doctrine between them across multiple files and masking the unclosed region error. | Code inspection of `render_region_compact`'s `awk` loop which maintains global state across files. | UNTESTED | code + tests |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Criteria met on happy paths, but stream parsing edge cases (cross-file bleeding, same-line markers) exist. |
| Verification       | A | Verification evidence covers criteria with reproducible commands and outputs. |
| Scope              | A | Diff matches proposal exactly; no unrelated changes or creep. |
| Reliability        | B | Handled most error paths, but stream parsing is fragile to edge cases. |
| Maintainability    | A | Clear structure, no complex nesting or high cyclomatic complexity. |
| Handoff-readiness  | A | Spec is fully updated, decisions and lessons captured. |

### Verdict
PASS WITH GAPS

### Recommended next steps
- Fix `grep -v` in `render_region` to use `sed` (e.g. `sed 's/<!-- full-only:\(begin\|end\) -->//g'`) so that it strips the marker instead of dropping the entire line.
- Update `awk` parsing in `render_region_compact` to assert the markers are alone on a line, or split strings instead of skipping the entire line.
- Reset the `full_only` state in `render_region_compact` between files, or parse each file individually before concatenation, to prevent cross-file marker bleeding.
- Once these are applied, ticketed, or declined in `verification.md`, `dotf spec archive` is advisable.
