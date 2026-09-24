---
spec: "HARNESS-077-capability-map"
verdict: "PASS"
reviewed_sha: "2dd16a2c51d4bb5c21e35490da6f44b3269c71be"
reviewer: "nan/mimo-v2.5"
date: "2026-09-23"
---

## Adversarial review

**Scope**: HARNESS-077-capability-map (full diff `358e8caa...HEAD`, 6 commits)
**Sources**: `specs/HARNESS-077-capability-map/{proposal,tasks,verification,features}.md`; `harness/capability-map.json` + `capability-map.schema.json`; `cli/internal/harness/capability_map.go` (+ test); `cli/internal/cmd/harness_resolve_capabilities.go`; `scripts/compile-harness.sh`; `tests/compile-harness.bats` / `tests/compile-harness-real.bats`.

### Prior review disposition

This is the second adversarial round. The prior review (round 1, `nan/deepseek-v4-flash`) issued FAIL on two REAL Majors and three Theoretical Minors. All three Majors have been addressed in the follow-up commit `2dd16a2` ("land the capability-map review fixes that #1172 merged without"):

| Previous finding | Status in current HEAD |
|---|---|
| **Major/REAL** — zsh runtime abort from unescaped bracket pattern `${caps#[}` | **Fixed.** `compile-harness.sh:454` now uses `${caps#\[}` and `${caps%\]}`. A runtime zsh probe exists: `@test "agents: the capability list parses identically under bash and zsh"` in `tests/compile-harness.bats`. |
| **Major/REAL** — `verification.md` "end-to-end payoff" quoted stub values as real | **Fixed.** Now correctly reads `tools: Read, Glob, Grep, Edit, Write, Bash` and notes the earlier draft mistake parenthetically. |
| **Minor/THEORETICAL** — block-style `capabilities:` silently treated as none | **Fixed.** `agent_capability_line` now detects block-style via awk and fails with `[ERROR] ... block style ... would GRANT`. Tested by `@test "agents: a block-style capabilities list fails loudly..."`. |
| **Minor/THEORETICAL** — `grant` not required when `form=decision-map` | **Fixed.** Schema now has `allOf` / `if form=decision-map then required: [field, form, capabilities, grant]`. Tested by `@test "a decision-map harness with no grant cannot grant anything"`. |
| **Minor/THEORETICAL** — fail-open degrade semantics undocumented | **Tracked.** The runtime direction (missing allow-list = grants default set) is now spelled out in the deploy warning text. Spec-level documentation is a follow-up, not a gate. |

### Spec and task alignment

- All 9 acceptance criteria map to ticked tasks; every `[x]` has diff evidence across the 6-commit range. Contract files (`proposal.md`, `tasks.md`, `features.json`) read but not modified; no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain.
- All 13 `features.json` verifier commands executed this session: **13/13 exit 0** (their `state: pending` is correct — only the harness may set `passing`).
- Full toolchain verified: `go build ./... && go vet ./... && GOOS=windows go vet ./...` clean; `go test ./...` 18 packages pass; `golangci-lint run` → `0 issues`; `gofmt` clean; `shellcheck --severity=error` clean; `bash -n` + `zsh -n` clean; `bats tests/compile-harness.bats` 72/72; `bats tests/compile-harness-real.bats` 9/9.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | SPECULATIVE | scope | The diff includes ~1600 lines of unrelated work: GUARD-005 (review verdict provenance), GUARD-006 (agent-tier agreement), AI-023 (Orca ADE overlay), plus harness manifest/skills changes for those specs. These are structurally independent — no code dependency on the capability map — but they inflate the diff to ~3500 lines across 53 files, well beyond ADR-017's ~300 LOC cap. The proposal's "Out of scope" section explicitly deferred the doctor checks to a follow-up, yet `cli/internal/doctor/checks_agent_tiers*` (157+186 lines) is in this diff. | `git diff --stat` shows 53 files; ~1600 lines from unrelated specs | N/A (process concern, not a code defect) | — (already shipped; flag for future discipline) |
| Minor | THEORETICAL | deploy summary | The `deploy_agents` error summary (`%s agent record(s) failed to render: %s`) uses `${failed[*]}` which joins with `$IFS` — if agent names ever contain spaces or glob characters, the summary is ambiguous. Current agent names are bare identifiers, so this is safe today. | code read of `deploy_agents` | `@test "DESIGN-1169: every failing record is named in one run"` — names contain no spaces | code (quote each element: `"${failed[@]}"` requires IFS handling; or document the identifier-only assumption) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 9 ACs verified; zsh escape fixed and runtime-tested; negative paths (unmapped, undeclared, empty, block-style, missing grant) all fail loudly |
| Verification       | A | 13/13 features verified; real-binary tests prove the core claim (two native forms differ); Go table tests cover all resolver paths; bats covers shell render + degradation |
| Scope              | B | Core capability map maps exactly to the proposal; ~1600 lines of unrelated work in the same diff is noted but structurally independent |
| Reliability        | A | Fail-loud on every boundary (absent map, schema-invalid, unmapped cap, undeclared harness, empty request, block-style, missing grant); collect-not-abort (#1169) works correctly; degrade warns about the fail-open direction |
| Maintainability    | A | All functions under 40 lines; clear naming and why-comments; `$comment` block in the map is a measured rationale, not decoration; schema is self-documenting with descriptions |
| Handoff-readiness  | A | All spec artifacts written and accurate; lessons/ADR/promotion correctly assessed as not applicable; verification.md corrected after round 1 |

### Verdict
**PASS** — no blockers or majors; all three previous Majors resolved with named tests; rubric all A except Scope at B (unrelated work in the diff). The two remaining Minors are tracked follow-ups that do not gate archive.

### Recommended next steps

1. **Archive is advisable.** `dotf spec archive HARNESS-077-capability-map` should succeed — all prior review findings are dispositioned, all 13 features verified, no contract-file changes since this review.
2. **Follow-up (not gating):** Open a ticket for the scope discipline finding — the diff should have been split per ADR-017's ~300 LOC cap. The unrelated work (GUARD-005, GUARD-006, AI-023) landed correctly but belongs in its own PRs.
3. **Follow-up (not gating):** The `${failed[*]}` in `deploy_agents` could be made IFS-safe if agent names ever include whitespace, or the identifier-only assumption documented in a comment.
