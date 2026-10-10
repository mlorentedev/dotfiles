---
spec: "HARNESS-063-spec-gate-adjacency"
verdict: "PASS WITH GAPS"
reviewed_sha: "ca4685d22c115e55e75fc209c237274d1daccd99"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-10-10"
---

## Adversarial review

**Scope**: HARNESS-063-spec-gate-adjacency
**Sources**: `specs/HARNESS-063-spec-gate-adjacency/{proposal,tasks,verification}.md`, `.github/workflows/spec-gate.yml`, `scripts/check-spec-gate.sh`, `tests/spec-gate-adjacency.bats`, `scripts/spec-gate-pr.sh`

### Spec and task alignment
- **Criteria 1-4** are covered with concrete evidence.
- The advisory report is properly injected in CI and defensively handles missing credentials or feeds.
- The `gh issue list` jq parse explicitly handles null bodies and flattens newlines to preserve TSV formatting.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major    | THEORETICAL | matching | `git diff --name-only` wraps filenames containing spaces/special chars in double quotes unless disabled. The resulting quotes leak into `file_path` and `base`, causing `*"$base"*` to search for `"name.sh"`, which will silently fail to match the unquoted issue text. | Argued from bash logic (the output of `git diff --name-only` on `a b.sh` is `"a b.sh"`) | UNTESTED | code + tests |
| Minor    | REAL | reporting | **(Already Ticketed #2303)** Substring noise. Generic `*"$base"*` matching flags any file whose name is a substring of issue text. E.g., `ui` flags `building the ui`. | `verification.md` round 1 dispositions | UNTESTED | code |
| Minor    | REAL | reporting | **(Already Ticketed #2303)** Markdown injection in step summary. An issue body containing `\|` splits the `GITHUB_STEP_SUMMARY` table structure. | `verification.md` round 1 dispositions | UNTESTED | code |
| Minor    | REAL | reporting | **(Already Ticketed #2303)** Renamed files. `git diff --name-only` only outputs the *new* filename, so an issue referencing the *old* filename is missed. | `verification.md` round 1 dispositions | UNTESTED | code |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Criteria met on happy path; negative-path gaps identified (path quoting). |
| Verification       | A | Evidence proves each criterion with reproducible commands, mutation testing, and characterization tests. |
| Scope              | A | Diff matches proposal exactly; offline paths kept intact. |
| Reliability        | B | Gracefully degrades to silence on error. Path quoting issue is a silent theoretical failure. |
| Maintainability    | B | Matcher split makes logic clearer; `_report_adjacent_issues` at 39 lines is acceptable. |
| Handoff-readiness  | B | Spec updates included, lessons captured in `docs/lessons.md`. |

### Verdict
PASS WITH GAPS

### Recommended next steps
- Fix the `core.quotePath` bug in `git diff --name-only` by passing `-c core.quotePath=false` (easiest fix without changing to `-z` null-byte parsing) so filenames with spaces are matched correctly against the issue text. Add a named test for a path with spaces.
- The remaining findings are already ticketed under #2303.
