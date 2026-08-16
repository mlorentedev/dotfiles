---
spec: "BUG-086-verify-partial-registry"
verdict: "PASS"
reviewed_sha: "e033302489a9e446efa8e38253cb7aa5a7b8a590"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-16"
---

## Adversarial review

**Scope**: BUG-086-verify-partial-registry
**Sources**:
- `specs/BUG-086-verify-partial-registry/proposal.md` — 8 acceptance criteria, explicit non-goals, risks documented
- `specs/BUG-086-verify-partial-registry/tasks.md` — all 8 implementation tasks ticked; verification box ticked
- `specs/BUG-086-verify-partial-registry/verification.md` — named tests per AC, mutation check observed failing, build/vet/test results
- `cli/internal/secrets/registry.go` — `ParseRegistryPartial`, `SecretDefect`, `validateSecret` at HEAD
- `cli/internal/cmd/secrets.go` — `loadRegistryPartial`, `scopeVerify`, verify's RunE at HEAD
- `cli/internal/secrets/registry_test.go` — new and pre-existing validation tests
- `cli/internal/cmd/secrets_registry_test.go` — new verify tests
- `718c895` — the implementing commit (merge-ancestor of HEAD)
- `git diff 718c895~1..718c895` — full implementation diff

### Spec and task alignment

All 8 implementation tasks are ticked `[x]`. The diff covers exactly what the tasks describe:
- [AC6] `validate` → `validateSecret` split ✓ (registry.go)
- [AC1] `ParseRegistryPartial` with per-secret validation, defects returned ✓
- [AC6] `ParseRegistry` reimplemented on top of it ✓
- [AC5] Defective secrets excluded from returned registry ✓
- [AC4] id/vars registered only after validation ✓
- [AC1] verify reports each defect as FAILED row ✓
- [AC2, AC3] `scopeVerify` matches args against defects and registry ✓
- Tests, including mutation check ✓

No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in any spec artifact. The proposal's frontmatter does not declare `review: waived`, so this review is in-scope.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|-------------|
| Minor | REAL | maintainability | `validateSecret` is 72 lines, nearly 2× the AGENTS.md guideline of ≤40 lines per function. The function is a sequential validation list (each check is independent and short), but the length exceeds the stated threshold without inline justification. | Code read at `cli/internal/secrets/registry.go:231-302`. `scopeVerify` also edges over at 41 lines. | `TestParseRegistry_DefectiveSecretsNeverReachEntries` exercises all internal paths transitively; no test *directly* asserts anything about function length. | code — either inline a comment explaining why length is justified (sequential checks), or extract one or two of the longer blocks (e.g. bw.folder validation, file mode validation) into named helpers. |
| Minor | THEORETICAL | clarity | `scopeVerify` double-counts a token that is BOTH a defect id and a valid var name (e.g. a malformed secret with `id: FOO` and a well-formed secret with `expose: {env: FOO}`). The token produces both a defect row and a healthy var resolution, which is arguably correct (AC8), but the behavior is not documented in the function's comment. | Code read at `scopeVerify` (secrets.go:225-258), specifically the `known || !isDefect` branch. No existing test exercises this specific cross-over. | UNTESTED — no named test exercises the case where a defect id collides with a well-formed var name. | spec — document the expected behaviour in `proposal.md`'s AC8 or in a new subsection; tests — optionally add a regression test if the collision is considered within scope. |
| Minor | SPECULATIVE | performance | `ParseRegistryPartial` builds a `kept` slice and copies each validated secret with `kept = append(kept, *s)`, which copies the entire struct (including all string fields, slices, maps). For very large registries (hundreds of secrets), this allocates O(n) memory beyond the original YAML decode. The original strict door did no copying because it returned instantly on the first defect. | Code read at registry.go `ParseRegistryPartial` loop. No reproduction or observed failure. | UNTESTED — no benchmark exists for this path. | — (surface only; no action needed unless a registry exceeds ~500 secrets) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | All 8 ACs verified; negative paths covered by named tests; mutation tests confirm the exact bug is caught on reintroduction; `go test ./...` green on all packages. |
| Verification | A | Each AC maps to a named `Test*` function; verification.md proves the mutation guard was observed failing; build/vet/entire-test-suite run and green; real-registry smoke test (`dotf secrets verify` → 33 ok) run. |
| Scope | A | Diff touches only the files the spec describes: `registry.go` (split + partial), `secrets.go` (verify wiring + scopeVerify), test files, spec files. The unrelated `probe` command was added in a later commit. No creep. |
| Reliability | A | All error paths handled: structural failures (YAML, version) stay fatal; per-secret defects recorded and excluded; nil-map reads safe; empty registries yield no entries; exit codes correct for all conditions. |
| Maintainability | B | `validateSecret` at 72 lines exceeds the 40-line guideline; `scopeVerify` at 41 lines barely does. Both are clear, well-documented, and follow a sequential-check pattern, but the length gap is measurable. |
| Handoff-readiness | A | Spec artifacts (proposal, tasks, verification) complete and consistent with the diff. Lessons evaluated as "no new rule" with a reasoned citation to existing lessons. No stale sections. |

### Verdict

**PASS**

No blockers or majors. The rubric grades are all B or above (aggregation rule: PASS). The two Minor findings — function length (`validateSecret`) and an undocumented edge case in `scopeVerify` — are surfaced but do not move the verdict. See recommended next steps before archive.

### Recommended next steps (before archive)

1. **Minor (maintainability):** Either add an inline comment in `validateSecret` explaining why the sequential-check pattern justifies exceeding 40 lines, or extract the bw.folder block and/or the file-mode block into named helpers. Either approach resolves the finding without a behaviour change.
2. **Minor (clarity):** If the collision case (a token matching both a defect id and a valid var name) is considered a supported scenario, document it in `proposal.md` under AC8 or a new subsection. If it is not intentional and should be a guard, add a test and a short-circuit. This does not block archive.
3. **Archive readiness:** No `[AGENT-DRAFT]` tags, spec files are stable, and `review.md` is present with the correct frontmatter. `dotf spec archive` is **advisable** in the current state. The two minors above are optional follow-ups, not preconditions.