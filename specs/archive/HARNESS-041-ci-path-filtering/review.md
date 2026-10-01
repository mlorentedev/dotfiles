---
spec: "HARNESS-041-ci-path-filtering"
verdict: "PASS"
reviewed_sha: "d990b3dd1affd40cd67684dc6114c4dbc29f64f8"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: HARNESS-041-ci-path-filtering, round 2 (base `76d875159d3ac4644357b04fb3d328b7d2c58464` → `d990b3dd1affd40cd67684dc6114c4dbc29f64f8`, as resolved by the launcher; the whole change, judged whole — round 1 failed it at `b776e9d`, the fix commit is `d990b3dd`)

**Sources**: `specs/HARNESS-041-ci-path-filtering/{proposal,tasks,verification}.md`, `git diff 76d8751...HEAD`
(~376 first-parent commits; the spec's own change is `3dadead` + the round-2 fix `d990b3dd`; the rest of the
range is later work reviewed for how it interacts with this spec's mechanism), `.github/workflows/ci.yml`,
`tests/ci-path-filtering.bats`, `tests/workflow-job-names.bats`, `tests/lib/check-workflow-contexts.py`,
`forge/branch-protection.json`, `.pre-commit-config.yaml`, `harness`-side digest code in `cli/internal/spec/`.

**Verification commands run this session** (all at `d990b3d`):

- `bats tests/ci-path-filtering.bats tests/workflow-timeouts.bats tests/workflow-job-names.bats` → 9/9 ok, exit 0, no skips (PyYAML present)
- `bats tests/check-doc-paths.bats tests/check-lessons.bats tests/docs-drift.bats tests/guard-lesson-numbers-unique.bats tests/guard-spec-ids-unique.bats` → 37/37 ok, exit 0 (the suites behind the docs-only decision and over the new lesson 322)
- Contract freshness: recomputed `spec.ContractDigests` for the spec dir via a throwaway Go test (deleted afterwards) → `proposal.md ccdef205…`, `tasks.md 6e4d95ae…`, `features.json ""` — **exactly** the values in `review-request.json`; the contract set is fresh against this review
- Mutation 1 (`needs: []` on `test-windows`) → `not ok 2 every filtered job needs the changes job`
- Mutation 2 (delete the `if:` of the heavy "Run bats test suite" step) → `not ok 3 every step after checkout … is guarded`
- Mutation 3 (drop `'secrets/**'` from the `code` filter) → `not ok 4 every top-level entry …`
- Mutation 4 (add a skipping job-level `if:` to required job `test`) → `not ok` from `check-workflow-contexts.py` run on the real repo (required-rule catches it)
- Mutation 5 (unpin `dorny/paths-filter@v3`) → `not ok 1 … runs a SHA-pinned dorny/paths-filter`
- Mutation 6 (repoint a heavy step's guard at `outputs.powershell`) → **stays green** (finding 1)
- Mutation 7 (replace a guard with `${{ true }}`) → `not ok 3`
- All mutations reverted; `git status --porcelain` clean apart from the launcher's own `review-request.json`
- `gh issue view 1869` → OPEN ("Remove the stray diff.patch committed at the repository root") — round-1 disposition 9's ticket is real
- `.pre-commit-config.yaml` → `check-doc-paths` and `check-lessons` hooks present — the docs-only decision's pre-commit backstop is real for those two
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags in proposal/tasks/verification

### Spec and task alignment

| AC | Claimed | Actual at `d990b3d` | Named test | Verdict |
|----|---------|----------------------|------------|---------|
| 1 | `changes` job using a SHA-pinned `dorny/paths-filter` | job exists; `dorny/paths-filter@ceb8a2b8… # v4`, regex-asserted as 40-hex | `@test "HARNESS-041: the changes job runs a SHA-pinned dorny/paths-filter"` — red under mutation 5 | met |
| 2 | five jobs depend on `changes` | verified in YAML: `lint`, `lint-powershell`, `test`, `test-windows` → `needs: [changes]`; `integration` → `needs: [changes, lint]` | `@test "HARNESS-041: every filtered job needs the changes job"` — red under mutation 1 | met, mutation-proven |
| 3 | every step after checkout carries a guard on a `changes` output | verified by YAML parse: every non-checkout step in the five jobs has an `if:` containing `needs.changes.outputs.` (the round-1 event-guarded-only retired-twin step is now `pull_request && code`) | `@test "HARNESS-041: every step after checkout in a filtered job is guarded by a changes output"` — red under mutations 2 and 7 | met, mutation-proven (see finding 1 for the semantic edge) |
| 4 | regression suite passes; every tracked top-level entry in `code` or declared docs-only | 4/4; coverage test green on the final tree | `@test "HARNESS-041: every top-level entry is in the code filter or declared docs-only"` — red under mutation 3 | met, mutation-proven |

Round-1's two REAL Majors are both closed with shown evidence: the tests now parse the workflow per job and
per step (mutations 1–2 red, the exact mutations round 1 showed green), and every path round 1 named as an
omission (`secrets/**`, `install.sh`, `ssh/**`, `systemd/**`, `forge/**`, `env-contract.json`,
`machine.json.example`, `session-start-config.json`, `.gitattributes`, `.gitignore`, `.claude/**`) is in the
`code` filter, with more added (`AGENTS.md`, `.pr_agent.toml`, `.geminiignore`, both `*.local.example`).
All round-1 Minors are dispositioned in `verification.md` with apply/decline/defer, and the two I could
check independently hold: the `@v3` drift is gone from AC1, "3/3" is now 4/4, `features.json` promise
replaced, `status: verifying` with ACs ticked, retired-twin step guarded, #1869 open.

Non-goals respected: no branch-protection files altered; no AI-reviewer integration touched.

**Strengths that mitigate documented risks:** the YAML-parsing tests close round 1's exact mutation gaps
(shown, not claimed); the coverage test turns "a new directory silently becomes docs-only" from a
possibility into a red build (mutation 3); the required-check property (a skipping job-level `if:` on
`test`) is independently caught by `check-workflow-contexts.py` on the real repo (mutation 4), so the
proposal's named "required check blockage" risk has a live guard at both step and job level.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | THEORETICAL | test-traceability | The step-guard test accepts **any** `needs.changes.outputs.*` reference, so a guard pointed at the wrong output passes: repointing the heavy "Run bats test suite" step at `outputs.powershell` left all four tests green. If that edit ever landed, a pure `cli/**` PR would skip the entire bats suite while every test stays green. AC3's letter ("a guard on a `changes` output") is met by the test as written; its intent (the *right* filter per job) is not asserted. | mutation 6 this session (green baseline, green after wrong-output edit) | `@test "HARNESS-041: every step after checkout in a filtered job is guarded by a changes output"` survives it → **UNTESTED** for wrong-output | tests (assert the expected output per job: `code` for lint/test/test-windows/integration, `powershell` for lint-powershell) |
| Minor | THEORETICAL | filter coverage (residual) | `docs/` and `specs/` stay declared docs-only, yet the suite reads both (`tests/docs-drift.bats`, `check-doc-paths`, `guard-lesson-numbers-unique`, `guard-spec-ids-unique`), so a docs/specs-only PR skips those checks pre-merge; they catch it on the post-merge push, making this a red-main-not-red-PR class. AC4 explicitly allows declared docs-only entries and `verification.md` dispositions this as a decision (pre-commit + push backstop); I confirmed the push guard and the `check-doc-paths`/`check-lessons` pre-commit hooks, but `docs-drift` and the spec-id guards are in neither pre-commit list. | read of `ci.yml` `docs_only` block + `.pre-commit-config.yaml`; no observed breakage in the range | **UNTESTED** — no test asserts a docs-only classification stays inside the decided set beyond the test's own list | tests / accepted decision (carry in `verification.md`; no contract edit) |
| Question | THEORETICAL | resilience | The spec's mitigation covers step-level skipping, but if the `changes` job itself fails (paths-filter step error, action outage), the five dependents skip at **job level** via default `needs` semantics — the exact job-level skip the proposal's risks section warns about, arriving one level up. What branch protection does with a skipped required check in this configuration is the open question; nothing here reproduces it. | code read of `ci.yml` `needs:` semantics; no repro, no observation | **UNTESTED** | code (e.g. evaluate outputs under `always()` with fail-open/ fail-closed chosen deliberately) — needs author confirmation first |
| Minor | REAL | disposition integrity | Round-1 finding 9 (stray `diff.patch`) is deferred to #1869 and listed docs-only in the test with that reference; `gh issue view 1869` confirms the ticket is OPEN, so the defer is backed by a live ticket, not a verbal promise. | `gh issue view 1869` → open | n/a | code (follow-up already ticketed; no action in this spec) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All four ACs verified against the final tree with red-green mutations; one semantic edge (wrong-output guard accepted) keeps it off A |
| Verification       | A | Every mutation claim in `verification.md` reproduced exactly this session, plus five more of my own; contract digests recomputed and matched |
| Scope              | B | The spec's own commits touch only ci.yml, its tests, its spec files and lesson 322 — no creep; the launcher-resolved range carries ~370 unrelated commits by construction, reviewed here only for interaction |
| Reliability        | B | Push/pull-request guards and all error paths of the filter job itself behave; the unexamined path is a failing `changes` job skipping the five dependents (question above) |
| Maintainability    | B | Tests parse YAML instead of grepping, comments explain why, docs-only classification carries a reason per entry; embedded Python heredocs in bats are unlintable but short |
| Handoff-readiness  | A | proposal/tasks/verification all brought in line, nine round-1 findings dispositioned, lesson 322 + index entry written in the same commit |

### Verdict

**PASS**

No blockers, no Majors — both round-1 REAL Majors are closed with mutation evidence I reproduced, and the
rubric is all B or above. The four open findings are Minors and one Question, all THEORETICAL, each with a
disposition path; per the severity × reality rule none gates the archive.

### Recommended next steps

- Finding 1 (wrong-output guard): harden the named test to assert the expected output per job — `tests/` change, outside the contract set; can follow up or land before archive without invalidating this verdict.
- Finding 2 (docs/specs residual): already dispositioned in `verification.md`; if wanted, add `docs-drift` / spec-id guards to the pre-commit set or note the accepted red-main window — record the disposition there, no contract edit.
- Finding 3 (failing `changes` job): confirm GitHub's behaviour for a skipped required check under this workflow shape, then decide fail-open vs fail-closed — `code` if it warrants a guard, otherwise a line in `verification.md`.
- Finding 4: no action; #1869 tracks it.
- `dotf spec archive HARNESS-041-ci-path-filtering` is **advisable** in this state: the contract digests match this review's launch record, no draft tags remain, and the verdict is PASS.
