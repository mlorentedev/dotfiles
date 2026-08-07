---
id: "BUG-027-spec-gate-fail-closed"
type: spec
status: implementing
created: "2026-07-09"
tags: [spec, tasks]
template_version: "1.0"
---

# Tasks — BUG-027-spec-gate-fail-closed

- [x] T1 (C3). Compute the diff before the loop; fail closed (exit 2) with an
  actionable message on a git error; feed the loop from the captured output.
- [x] T2 (C25.1). Remove the `*generated*` glob from `_excluded`; document the
  explicit-allowlist policy.
- [x] T3 (C25.2). Add `_is_dependency_bot` + `SDD_PR_AUTHOR`; gate the
  `dependencies` skip on a bot author; wire `SDD_PR_AUTHOR` in `spec-gate.yml`.
- [x] T4 (C25.3). Accumulate active-spec LOC; require `>= SPEC_FLOOR` (10) to
  count as a spec touch; surface it in `--explain`.
- [x] T5. Update the changed-contract test (substantive spec) and add 5 new
  tests: fail-closed, generated-counts, bot/non-bot dependencies, sub-floor
  alibi.
- [x] T6. Local verification: `bash -n`, full `tests/check-spec-gate.bats`.
- [ ] T7. CI green (`lint` shellcheck, `test` bats, `spec-gate` self-run).
