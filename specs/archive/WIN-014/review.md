---
spec: "WIN-014"
verdict: "PASS WITH GAPS"
reviewed_sha: "a8e11e29851ad2c17b17e6c8ed8d7edaba44ebc6"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-29"
---

## Adversarial review

**Scope**: WIN-014
**Sources**: `specs/WIN-014/{proposal,tasks,verification}.md`, `git diff b15ad970f98c1ec6de1a2e51defc944cbbb0280a...HEAD`

### Spec and task alignment
- AC1: `dotf harness mirror` correctly respects the `--repo` flag and overrides the repository selection. Both setup scripts now pass this argument, ensuring reliable execution outside the deployment directory.
- AC2: Mirror operations atomic replace destination files and cleanly resolve Windows read-only locks by temporarily setting `0o200` write permissions to the destination before `os.Rename`. `os.Rename` correctly preserves the source attributes (e.g. `0o444`) on Windows when substituting the target file.
- AC3: Both `setup-linux.sh` and `setup-windows.ps1` execute mirror explicitly and print a warning if `dotf` is missing.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | Scope | Review base `b15ad9...` expands the diff scope to 92 files instead of the intended 6, mixing in unrelated secrets and doctor changes. | `git diff` output includes `secrets_curate.go`, `CHANGELOG.md`, etc. | UNTESTED | spec |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All acceptance criteria verified, permissions are preserved on Windows via `os.Rename`. |
| Verification       | A | Robust BATS and Go regression tests exercise the read-only replacement and setup fallback behaviors. |
| Scope              | C | Diff mixes significant unrelated changes due to an outdated review base, as acknowledged in `verification.md`. |
| Reliability        | A | Safe atomic rename operation restores permissions upon failure; avoids incomplete writes. |
| Maintainability    | A | Focused additions, readable code, properly mocked BATS scripts. |
| Handoff-readiness  | A | Spec is fully updated, and lessons are evaluated. |

### Verdict
PASS WITH GAPS

### Recommended next steps
- The `PASS WITH GAPS` is strictly due to the C grade in Scope caused by the outdated review base. 
- As noted in `verification.md`, regenerate the independent review request from the then-current `git merge-base origin/main HEAD` and re-request a review to clear the scope inflation before archive.
- `dotf spec archive` is advisable after addressing the scope mismatch.
