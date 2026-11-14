---
id: "HARNESS-101-model-eval-runner"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-11-13"
issue: "mlorentedev/dotfiles#1399"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, evals, reviewer-pool]
template_version: "1.0"
owner: manu
---

# HARNESS-101-model-eval-runner

## Why

<!-- from issue #1399: HARNESS-101: Deterministic Multi-Provider Eval Runner & Grader Engine -->

The models in the reviewer pool and the model map have been measured by hand three times in four weeks (#2385, #3273, #3476). Each time the method was rebuilt from an issue comment, and #2385's prompt was lost. #3476's replay of an archived review gave the strongest signal: it found #3481. But it was built by hand from the merged state, so the reviewers flagged the construction itself, and the replay could not see the deleted tests that its round was about. ADR-063 decides a command that does this reproducibly and grades it without a judge model. This spec builds the first slice: the spec-reviewer role.

## What

After this spec:

- dotfiles' `.dotf/evals/cases/` holds a schema-validated corpus, each case carrying its `repo`, and `harness/evals/` holds the schemas, the candidates and the planted cases:
  - at least 5 replay cases, each an archived review round, pinned to its `reviewed_sha` and `base_sha`;
  - at least 2 planted-defect cases, including #3476's prompt and its known false bash 3.2 claim as a decoy;
  - an anchor-encoded answer key per case.
- `dotf eval check` proves each key in the current repository against its own source review: full recall and no decoy hit, with no model involved.
- `dotf eval run` replays the cases for the named candidates over N epochs:
  - each replay runs in a detached worktree at the pinned commit, outside every repository tree;
  - it launches through a new eval slot of the `internal/spec` review launcher, which takes an explicit candidate and base and writes outside `specs/`;
  - it grades every run deterministically.
- `dotf eval run --plan` prints the run matrix and a cost estimate per runner, and an invocation above its run cap is refused.
- Each run writes `results.json` (schema-validated), `report.md` (per-role table) and `proposal.md` (the pool or map change the results support). Nothing is written to the pool, the map or the pins.
- `scripts/nan-quality-bench.sh` and `tests/evals/harness-evals.json` are retired, with their references.

## Out of scope

- Cases from other repositories (kubelab, ts-bridge). The schema supports them; each one lands in its own PR.
- The PR-Agent, implementer and daily-use roles. ADR-063 §5 names their case kinds, and each one gets its own spec.
- Scheduled runs in ci-workflows (ADR-063 §7, deferred with triggers).
- Editing `reviewer-pool.json`, `model-map.json` or `model-pins.json`. Admission stays an owner decision (C36).
- A judge model of any kind (C35).
- Changing the surface of `dotf spec review`. The eval slot is an internal API.

## Risks / open questions

- **Gate integrity (C37).** The eval slot must not become a way around the pool's allow-list. Mitigation: output is written outside `specs/`, the reviewer is recorded as `eval:<id>`, and a test asserts that `checkReviewGate` refuses an eval record copied into a spec folder.
- **Anchor brittleness.** A reviewer that describes a defect without naming its symbol leaves the matching `expected` entry unmatched, which is a miss in that grading, and its row is reported as unadjudicated. Mitigation: raw outputs are kept, adjudication extends the key, and `dotf eval grade <run-id>` re-grades stored runs at no model cost.
- **SHA durability.** A squash-merged `reviewed_sha` survives only through GitHub's `refs/pull/<n>/head`. Mitigation: each case records `fetch_ref`, and an unresolvable case fails loudly as infrastructure.
- **Key circularity.** `nan/deepseek-v4-flash` authored 62 archived reviews. Mitigation: `key_author` on each case, and a candidate's results on its own keys are flagged and left out of its reduced score.
- **Cost estimate for a case that has never run.** `--plan` falls back to a per-kind default envelope declared in `candidates.json`, labelled `default`, never zero.
- **Token accounting per runner.** pi, agy and the Anthropic path report tokens differently in their transcripts. Open until the first runner adapter is written; a runner that cannot report tokens says so in the result rather than reporting 0.
- **Owner decisions** pending in ADR-063: recall policy, budget defaults, retirements, scheduling. This spec stays in `draft` until they are answered.

## Acceptance criteria

- [ ] AC1. `harness/evals/case.schema.json` exists and every case under `.dotf/evals/cases/` validates against it, including a `repo` field. At least 5 `replay` and 2 `planted` cases exist for `role: spec-reviewer`.
- [ ] AC2. `dotf eval check` (exercised by a Go test and run in CI) grades every case against its own `source_review` (replay) or recorded reference answer (planted) and asserts full recall over `expected` with zero decoy hits. A key whose anchors no longer match fails the test.
- [ ] AC3. A replay of a case resolves its `reviewed_sha`, fetching `fetch_ref` when absent, and checks it out detached outside the repository. A case whose SHA cannot be resolved exits non-zero naming the case, and is not scored.
- [ ] AC4. The eval slot launches a candidate from `harness/evals/candidates.json` with an explicit base. It never reads the pool, and it writes only to the run directory. A test asserts that `checkReviewGate` refuses an eval record placed in a spec folder.
- [ ] AC5. The grader reports, per run: completion class, format validity, verdict match, recall, decoy hits, unadjudicated rows, duration and tokens. An unparseable answer is a format failure, not zero findings.
- [ ] AC6. `dotf eval run --plan` prints the case × candidate × epoch matrix with a per-runner estimate: NaN credits, Anthropic money, agy run count. A run above `--max-runs` (default 30) is refused.
- [ ] AC7. `results.json` validates against `harness/evals/result.schema.json` and carries the provenance: dotf version, corpus commit, skill digest, runner version, model id. `report.md` and `proposal.md` are written beside it. A test asserts that no `dotf eval` code path opens the pool, map or pins for writing.
- [ ] AC8. `scripts/nan-quality-bench.sh`, `tests/evals/harness-evals.json` and their references are removed, and the suites that named them still pass.

## References

- ADR: `docs/adr/adr-063-model-evaluation-runner.md`
- Issues: #1399, #3476, #3482, #3481, #2385, #3273
- Pattern: `00_meta/patterns/pattern-llm-evals.md` (amended for answer-key grading)
