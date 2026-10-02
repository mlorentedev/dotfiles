---
spec: "HARNESS-084-doctrine-budget-policy"
verdict: "FAIL"
reviewed_sha: "7db329b73aff6ff7503097dfd33a100dae84d21e"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-10-01"
---
## Adversarial review

**Scope**: HARNESS-084-doctrine-budget-policy
**Sources**: `specs/HARNESS-084-doctrine-budget-policy/{proposal,tasks,verification}.md`, `git diff 4d0ffa99f7612797add2eb9d945634ecb376cf8d...HEAD`

### Spec and task alignment
- The implementation splits `render_region` and implements `render_region_compact` with `awk` to strip the `full-only` region.
- The `harness/enforced/no-auto-merge.md` file correctly wraps the exception in `full-only` markers.
- However, the risk identified in the proposal ("A missing end marker would swallow the rest of the record") is NOT mitigated. The guard only tests a well-formed fixture.
- There are undocumented side-changes regarding ASCII folding and preamble migration.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker  | REAL    | `compile-harness.sh` | `render_region_compact` silently swallows all remaining doctrine files if `<!-- full-only:end -->` is missing or misspelled. The `awk` script exits `0` instead of failing, silently deploying truncated doctrine. | Demonstrated locally: `echo -e "line1\n<!-- full-only:begin -->\nline2" \| awk '...'` exits 0. | UNTESTED | code + tests (add `END { if (full_only) exit 1 }` to `awk`, and a negative test in `tests/compile-harness.bats`) |
| Minor    | REAL    | Scope | The diff refactors ASCII folding into `fold_to_ascii` and adds `migrate_legacy_preamble` to replace em-dashes. This is a sensible fix for the cap policy but is completely undocumented in the spec. | `scripts/compile-harness.sh` | `@test "doctrine: the old em-dash preamble is migrated to ASCII..."` | spec (document the side-change in `proposal.md` or `tasks.md`) |
| Minor    | THEORETICAL | `render_region` | `render_region` reads raw output into a command substitution `out="$(render_region_raw "$@")"`, which strips all trailing newlines, and then adds exactly one back with `printf '%s\n'`. This diverges slightly from the raw source. | Bash command substitution behavior. | UNTESTED | — (surface only; no action required) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | D | Missing end marker silently truncates the payload (a defect directly contrary to the stated risk). |
| Verification       | C | Evidence proves happy paths but fails to test the negative case for the unclosed marker risk. |
| Scope              | C | Diff includes undocumented side-changes (ASCII refactoring, preamble migration). |
| Reliability        | C | Unhandled error path for unclosed `full-only` tags. |
| Maintainability    | B | Code is generally clean, well-commented, and structurally sound. |
| Handoff-readiness  | B | Spec updates are included, though missing the undocumented side-changes. |

### Verdict
FAIL

### Recommended next steps
- **code**: Update `render_region_compact`'s `awk` script with an `END { if (full_only) { print "Error: unclosed full-only marker" > "/dev/stderr"; exit 1 } }` block.
- **tests**: Add a negative test case in `tests/compile-harness.bats` to prove that a missing `<!-- full-only:end -->` marker fails the build and exits non-zero.
- **spec**: Document the `migrate_legacy_preamble` and `fold_to_ascii` changes in `proposal.md` or `tasks.md` to accurately reflect the implementation scope.
