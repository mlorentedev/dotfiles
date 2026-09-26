---
spec: "HARNESS-106-skill-capability"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "617cbafe27ba655dc00e46de070ec63416137d59"
reviewer: "nan/mimo-v2.5"
date: "2026-09-25"
---

## Adversarial review

**Scope**: HARNESS-106-skill-capability (diff `773b5b8...HEAD`, 4 commits)
**Sources**: `specs/HARNESS-106-skill-capability/{proposal,tasks,verification,features}.md`, full diff including HARNESS-106 code + OPS-040 code mixed in by the launcher's base selection

### Spec and task alignment

The spec addresses a real defect: no persona could invoke a skill because no capability mapped to a skill-invocation primitive, and no durable record existed for gate decisions. The implementation adds `skill` to the vocabulary, maps it for claude, declares it unsupported for opencode, adds a persona-level guard, and builds a decision journal with rotation, concurrency safety, and schema pinning. All nine acceptance criteria have named tests. The verification.md is thorough, with live measurements and red-direction proofs.

The diff includes one commit from OPS-040 (`06c3b7a`) that is unrelated to HARNESS-106 — see Finding 1.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec) |
|----------|---------|------|---------|----------|---------------------------|-------------------------------------|
| Major | REAL | scope | The diff contains OPS-040-dead-migration-purge work (commit `06c3b7a`): ~92 lines deleted from setup scripts, new `guard-doctrine-target-not-deleted.bats`, changes to `hive-upgrade-timer.bats`, `secrets-show-callsites.bats`, `setup-linux.bats`, `setup-windows.bats`, plus OPS-040 spec files and lessons. `tasks.md` closing claims "No unrelated changes in the diff (no scope creep)" — contradicted by the actual diff. | `git diff 773b5b8...HEAD --stat` shows 42 files changed, ~2744 insertions; ~2000 of those are OPS-040 | N/A (process/discipline, not code) | spec (`tasks.md` closing checkbox) |
| Major | REAL | spec-artifacts | `features.json` f4 is `"state": "pending"` with empty `evidence`, but `verification.md` documents AC4 as verified by a real dispatch on 2026-09-25. `tasks.md` also says "f4 is `pending`" which is stale after the verification landed. The three spec artifacts disagree. | `features.json` f4 vs `verification.md` AC4 vs `tasks.md` Machine-readable features section | N/A (spec artifact inconsistency) | spec (`features.json`, `tasks.md`) |
| Minor | THEORETICAL | maintainability | `RunE` in `harness_gate.go` is ~134 lines with ~11 decision points, exceeding the repo's <40-line function length and CC<10 thresholds. The verification.md already discloses this and suggests extracting the record-building switch. | `wc -l` on RunE body; grep count of decision keywords | N/A (acknowledged in verification.md) | code |
| Minor | THEORETICAL | scope | The diff amends `specs/HARNESS-045-hook-emission/proposal.md` with a section documenting falsified AC3/AC4. Legitimate knowledge placement (Standing Order #3), but technically modifies a different spec's artifacts within this change. | `git diff 773b5b8...HEAD -- specs/HARNESS-045-hook-emission/proposal.md` | N/A | spec (HARNESS-045) |
| Question | — | security | `#1434`: a named dispatch makes `agent_type` carry the caller-supplied name, which makes the role unresolvable and turns enforcement off. Under `enforce: block` this is a bypass with a one-word opt-out. The spec correctly surfaces this as a hard precondition on block promotion and files it separately. No action needed in this spec, but the human should confirm #1434 is on the bitácora board. | `verification.md` "What the record caught on its first day" table | N/A (filed as #1434) | separate issue |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 9 ACs verified with named tests; negative paths covered; red-direction tests for new guards |
| Verification       | A | Every criterion backed by test names, live measurements, and reproducible commands; AC4 proven by real dispatch |
| Scope              | C | Diff includes unrelated OPS-040 work (~2000 lines); tasks.md claims no scope creep |
| Reliability        | A | Concurrency tested (24 writers × 40 records), rotation ordering pinned, torn-line survival, fail-open on all error paths |
| Maintainability    | B | RunE exceeds thresholds (disclosed), but overall structure is clean; `UnsupportedFor` separation is well-reasoned; schema pinning prevents silent drift |
| Handoff-readiness  | B | Spec artifacts filled, lessons written, #1434 filed; features.json f4 stale after verification landed |

### Verdict
PASS WITH GAPS

Two REAL Majors, both in spec artifacts (not code correctness): the scope mix in the diff and the features.json/tasks.md staleness on f4. Neither affects the correctness of the implemented code, which is solid — every AC has named tests, the decision journal is concurrency-safe and bounded, the `unsupported` declaration covers all four validation directions, and the security property (no tool inputs in the journal) is pinned by test.

### Recommended next steps

1. **(spec — tasks.md)** Correct the closing checkbox "No unrelated changes in the diff" to disclose the OPS-040混入, or rebase the HARNESS-106 work onto a base that excludes commit `06c3b7a`. The launcher chose this base, so the disclosure path is the one that does not invalidate the review.

2. **(spec — features.json)** Update f4's `state` to `"implemented"` and populate `evidence` with the dispatch attestation from `verification.md` AC4. Update `tasks.md`'s "Machine-readable features" section to match (currently says "f4 is `pending`").

3. **(code — follow-up, not blocking)** Extract the record-building switch out of `RunE` to satisfy the <40-line / CC<10 rules. Already noted in `verification.md`; no need to block archive on it.

4. **(process)** Confirm `#1434` is on the bitácora board as a hard precondition on promoting any skill to `enforce: block`.

`dotf spec archive` is **advisable after items 1–2 are addressed**. The contract set (`proposal.md`, `tasks.md`, `features.json`) needs the f4 correction; item 1 is a disclosure, not a contract edit. Item 3 is a follow-up ticket. Item 4 is a board check.
