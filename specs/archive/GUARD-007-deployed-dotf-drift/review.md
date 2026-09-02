---
spec: "GUARD-007-deployed-dotf-drift"
verdict: "PASS"
reviewed_sha: "3a11d977d6c3de7cbb3f57a8063ec8c6a65d7695"
reviewer: "nan/mimo-v2.5"
date: "2026-09-01"
---

## Adversarial review

**Scope**: GUARD-007-deployed-dotf-drift (diff `6e0d180..HEAD`, 20 files, +894/-17)
**Sources**: `specs/GUARD-007-deployed-dotf-drift/{proposal,tasks,verification,features}.json` + full diff

### Spec and task alignment

All six acceptance criteria have corresponding implementation, test coverage, and `features.json` entries. Every task in `tasks.md` is ticked except the PR-open checkbox, which is expected at review time. The implementation maps 1:1 to the proposal's state table (8 states → 8 subtests). No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags found in any spec file.

The scope includes a release-please version bump (0.52.0 → 0.53.0) and a lesson (252) — both documented and appropriate for the commit.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | testing | `TestCheckDotfProvenanceUsesRootRelativePathspec` asserts the argv string contains `:(top)cli` but does not test the count's correctness when git is invoked from a subdirectory. The test proves the mechanism is wired; a separate test proving the *behavior* (count differs from `cli` without the prefix) would close the gap. | verification.md manual test shows 4-from-root / 0-from-subdir split; mock test only checks argument string | `TestCheckDotfProvenanceUsesRootRelativePathspec` (mechanism only) | tests |
| Minor | THEORETICAL | edge-case | `dotf version --commit` producing whitespace-only output (e.g. `\n \n`) would be TrimSpace'd to empty and classified as a source build with no stamp. If a build system injected a non-SHA whitespace value, the message would say "source build" when the truth is "malformed stamp". | code path: `strings.TrimSpace(raw)` → `stamp == ""` branch | UNTESTED (no test for whitespace-only stamp) | code + tests |

No Blocker or Major findings. Both minors are THEORETICAL: the first is a test-depth question with the implementation verified correct; the second is an edge case that would require a broken build pipeline to trigger, and the existing "source build" message is a reasonable misclassification that still warns.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 6 ACs verified with fail-first evidence; 8/8 state subtests pass; no observed defects |
| Verification       | A | Every AC has reproducible commands with outputs; fail-first transcript for the critical pathspec bug; all 6 features.json commands exit 0 |
| Scope              | A | Diff matches proposal exactly; only two documented side-changes (version bump, lesson) |
| Reliability        | B | All error paths handled (skip on missing repo, missing binary, missing object, non-ancestor); minor theoretical gap on whitespace-only stamp |
| Maintainability    | A | 140-line implementation, 258-line test file; functions ≤40 lines; comments explain WHY at every decision point; lesson 252 captures the cross-project insight |
| Handoff-readiness  | A | Spec complete, lesson captured, features.json present, PR checkbox open (expected) |

All dimensions B or above → rubric supports PASS.

### Verdict
PASS

### Recommended next steps (before archive)
- Open the PR referencing this spec folder (the sole unticked task).
- No code changes required for archive. The two minor findings are optional follow-ups:
  1. A test that asserts the count from a subdirectory CWD (proving behavior, not just mechanism).
  2. A test for whitespace-only `--commit` output (very low priority).
- `dotf spec archive` is advisable once the PR is merged and the reviewer pool re-checks `review.md` freshness.
