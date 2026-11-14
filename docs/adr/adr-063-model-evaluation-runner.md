---
id: "ADR-063-model-evaluation-runner"
type: adr
status: proposed
owner: manu
date: "2026-11-13"
issue: "mlorentedev/dotfiles#1399"
extends: [adr-020-tooling-cli-go-convergence, adr-035-model-map-routing-registry, adr-046-anthropic-reviewers-never-sign-alone]
tags: [architecture, decision, evals, models, reviewer-pool, model-map]
created: "2026-11-13"
---

# ADR-063: `dotf eval` measures models by replaying archived spec reviews against an answer key

## Status

Proposed 2026-11-13. The spec is `specs/HARNESS-101-model-eval-runner/`. The owner decides the
points listed under *Decisions for the owner* before the spec leaves `draft`.

## Context

The models this harness uses have been measured by hand three times, and each time the method was
rebuilt from an issue comment:

- **#2385** (2026-10-14) admitted `gemini-3.8-flash-high` on a planted defect. Nobody saved the prompt.
- **#3273** (2026-11-10) sampled the larger NaN models once each. Two of them spent their whole output
  budget reasoning and returned nothing.
- **#3476** (2026-11-13) measured Gemini 4 Hydrogen. It ran a planted defect, then replayed the AI-055
  review in throwaway worktrees. The replay was the strongest signal of the three: it separated
  `medium` from `high`, and it found a real defect no reviewer of AI-055 had reported (#3481). It led
  to the pool change in #3482.

#3476 also showed where the hand method fails:

- **The replay was rebuilt from the archive, not from history.** The spec folder was taken from the
  merged commit and its review rounds were cut out by hand. The reviewers then flagged the cuts
  (`status: archived`, a `tasks.md` line pointing at dispositions that had been removed), and those
  findings had to be labelled by hand so they were not counted.
- **The replay missed what the round it replayed was about.** Round 2's two REAL Majors concerned
  tests that an intermediate commit deleted and a later one restored. The merged state no longer
  showed them.
- **One run, read by a person.** Only a manual check exposed `medium`'s false bash 3.2 claim. A single
  run with no false-positive check would have ranked `medium` equal to `high`.

### What was measured for this ADR

Every number below was measured in this session on `main` at `99cd03ae`.

| Fact | Measurement |
|---|---|
| Review files in `specs/archive/` that carry a `reviewed_sha` in their frontmatter | 209 of 209 |
| Archived specs with a `review-request.json` (which carries `base_sha`) | 141 of 321 |
| Archived specs with two or more review rounds | 35 |
| Findings rows marked `Blocker` or `Major` and `REAL` | 68 |
| `reviewed_sha` values still in the local clone | 80 of 141; 42 are ancestors of `origin/main` |
| A squash-merged `reviewed_sha` (`fb88359c`, AI-030) | still served by GitHub through PR #1922's ref |
| AI-055 round 2 at its own `reviewed_sha` (`06f3100b`) | `TestCheckReviewBudget_AFileWithoutAClipFailsClosed` is absent. The deleted test is visible, so the two REAL Majors #3476 could not see can be found |
| Earlier rounds' `review*.md` inside the replayed tree, over all 122 rounds whose `reviewed_sha` is in the clone | 0. A review file never leaks into the replayed tree. `verification.md` can: at AI-055 round 4 it already dispositions round 2's Minors |
| Fetching a squash-merged `reviewed_sha` | `git fetch origin refs/pull/1922/head` then `git cat-file -e fb88359c`: present |
| Anchor matching, tested by hand on AI-055 rounds 2 and 4 | `declaresStreaming`, `answerReserve` and `stepCeiling` each match exactly one findings row in both rounds. `AFileWithoutAClipFailsClosed` matches two rows in round 2, because two different findings name it |

The launcher was read for this ADR too:

- **Base.** `ResolveReviewBaseDetail` (`cli/internal/spec/review_request.go:229`) traces the commit
  that first added the spec folder, then widens to the fork point from `origin/main`. Which base it
  resolves depends on when it runs and on the clone. A replay must pin the base, not re-derive it.
- **Pool.** `loadReviewerPoolEntries` reads `harness/reviewer-pool.json` from the checkout. At a
  historical commit that is the old pool, so a candidate cannot come from there. In #3476 the pool
  file was edited and left uncommitted.
- **Skill.** The review prompt points at the deployed `adversarial-review/SKILL.md` under the
  runner's home (`review_launch.go:492`), not at the checkout. A replay therefore runs today's
  reviewer instructions against the historical code. That is the comparison we want, provided the
  skill's digest is recorded with each result.

### References audited (Phase B)

| Reference | Shape | What we take | What does not fit |
|---|---|---|---|
| `00_meta/patterns/pattern-llm-evals` (vault, local) | A golden dataset in `tests/evals/`, deterministic assertions first, an LLM judge for semantic dimensions, ablation runs | A versioned dataset next to the code. Deterministic checks wherever a key exists. Provenance pinning | The LLM judge. #1399 rules it out, and a key built from the spec's own review rounds makes it unnecessary for this role. The pattern is amended to say so |
| inspect-ai (UK AISI) | `Task` = dataset of `Sample`s + solver + scorer; `Epochs(n, reducer)`; a fresh sandbox per (sample, epoch); structured JSON logs | The vocabulary: case, epoch, reducer. One isolated environment per run. A log every result can be traced from | It is a Python framework. ADR-020 §2: "Python is not a layer here". The solver would be a subprocess wrapping `dotf spec review`, and the scorer would be ours anyway |
| promptfoo | YAML config, `exec:` script providers, `javascript` assertions, `--repeat N`, `-o results.json` | Repeats with stability reporting. The config declares the matrix | It is a Node runtime. The provider would again be `exec: dotf spec review`, and grading would be a custom assertion. It adds a runtime and keeps none of the logic |
| SWE-bench harness | An instance = repo + `base_commit` + gold patch + `FAIL_TO_PASS`/`PASS_TO_PASS`; one environment per instance; graded by executing tests. *Verified* is a subset that humans curated | Replay from pinned commits. The key is curated by a person once, then graded mechanically every run. Its executable grading is the model for the later implementer role | Its grading needs a test that fails. A review produces findings, not a patch, so this role grades against findings, not tests |
| OpenAI evals | A registry of YAML + JSONL samples; `Match`/`Includes`/`FuzzyMatch` deterministic templates; model-graded templates | Deterministic includes-style matching as the base primitive | Model-graded templates (excluded) |

**Divergence log.**

- *Shared by every reference, so we adopt it:* data separate from the runner; a pinned target state
  per case; repeated runs reduced to one number; one isolated environment per run; a structured log
  with provenance.
- *Specific to one reference, so we do not:* sandbox images (SWE-bench, inspect); LLM judges
  (pattern-llm-evals, OpenAI evals, inspect); web viewers (promptfoo, inspect).
- *Build or adopt.* With either framework, the solver would be a subprocess wrapping
  `dotf spec review`, and the scorer would be code we write. All a framework would add is a
  runtime that ADR-020 rules out. So we build, and we take the frameworks' data model.

## Constraints

C1-C34 are the dotfiles session protocol (`10_projects/dotfiles/prestudy/session-protocol.md`).
Applied here: C4 (enforcement in the harness), C9 (degrade on Windows and headless), C10 (a guard
in the same PR), C15 (fail loudly). C8 is amended: NaN has billed from one credit pool since
2026-11 (`ai/nan/README.md`), so it is no longer a sunk cost. Added by this session:

| # | Constraint | Origin |
|---|---|---|
| C35 | Grading is deterministic; no model grades another model | #1399 body and refresh |
| C36 | The runner never edits `reviewer-pool.json`, `model-map.json` or `model-pins.json`. Its output is a proposal; admission is the owner's | #1399 refresh |
| C37 | Nothing an eval run writes can satisfy the review gate (`checkReviewGate`, ADR-046) | HARNESS-071 allow-list integrity |
| C38 | Cost is counted per runner: agy spends the owner's subscription quota, NaN spends credits, Anthropic spends API money under one Console limit (ADR-046 §6) | #1399 refresh, `ai/nan/README.md` |
| C39 | Scores are per role, and per (model, case) over N runs. One run is an anecdote | #1399 refresh |
| C40 | A replay is rebuilt from git alone: pinned SHAs, fetchable refs, no hand-cut files | #3476 |

## Options considered

| Option | Door | C35 | C36 | C37 | C38 | C39 | C40 | ADR-020 |
|---|---|---|---|---|---|---|---|---|
| A. Adopt inspect-ai; `dotf spec review` as a subprocess solver | 1 | ok | ok | ok | gap (its own counters) | ok | ok | **violates** (Python) |
| B. Adopt promptfoo; `exec:` provider wrapping `dotf` | 2 | ok (custom JS) | ok | ok | gap | ok | ok | **violates** (Node) |
| **C. Build `dotf eval` in Go, reusing `internal/spec` launch; corpus in `harness/evals/`** | 2 | ok | ok | ok (eval slot) | ok | ok | ok | ok |
| D. Keep measuring by hand, with a runbook | 2 | partial | ok | gap (pool edited by hand) | gap | gap | gap | n/a |
| E. Grow `scripts/nan-quality-bench.sh` | 2 | gap (no key) | ok | ok | gap | gap | gap | violates the CLI drive |

**Socratic filter.**

- **Invariant:** the gate's allow-list cannot be bypassed (C37), and no model grades another (C35).
  This rules out any design that writes a `review.md` into a spec folder, and any judge model.
- **The naive trap:** "save the prompts and keep doing it by hand" (D). Three hand measurements in
  four weeks show its second-order failure: the method drifts each time, the prompt gets lost, and
  replay artifacts are graded by eye.
- **Rejection triggers:**
  - A and B reopen if a role needs a judge model the owner accepts, or if the runner grows past three
    task kinds that do not go through a `dotf` launch path.
  - D reopens never: three repetitions is the measured cost.
  - E is closed: the shell drive retires scripts and does not grow them.
- **Door:** C is Type 2. The corpus format is data and a schema, and the launcher change is one slot
  behind an internal API. Both are reversible inside a release.

## Decision

**C: `dotf eval`, a Go command that replays archived spec reviews and planted defects against a
curated answer key, grades the output deterministically, and writes a result plus a proposal.**

### 1. Where the corpus lives and how the key is encoded

**A replay case lives in the repository whose archive it replays**, at `.dotf/evals/cases/<id>.json`. `.dotf/` is
where a repository already commits its dotf markers. Each case carries a `repo` field (`owner/name`), so the
corpus is cross-project by construction (C12). A case has to sit beside its archive because the key is
checked against `source_review`, and only that repository's CI can read it. A repository adds cases
without touching dotfiles.

**What is repository-independent lives in dotfiles `harness/evals/`:**
- `case.schema.json` and `result.schema.json`;
- `candidates.json`;
- the planted cases.

`dotf eval run` runs from dotfiles and reads them there. `--repo <path>`, repeatable, adds another
repository's cases. The vault never holds cases, because repositories do not depend on the store. A
vault campaign whose synthesis adjudicates claims, such as the Fable correction ledger in
`10_projects/dotfiles/research/benchmarks/2026-09-23-gentle-ai/`, is a place to mine planted cases
and decoys, nothing more.

Each case is one JSON file, validated by `harness/evals/case.schema.json`:

```jsonc
{
  "id": "ai-055-r2",
  "repo": "mlorentedev/dotfiles",
  "kind": "replay",                       // replay | planted
  "role": "spec-reviewer",
  "spec": "AI-055-budget-guard-readable-reserve",
  "source_review": "specs/archive/AI-055-budget-guard-readable-reserve/review-round2.md",
  "key_author": "nan/glm5.3-flash",       // the model whose review the key came from
  "reviewed_sha": "06f3100b7efafdfdf77d250553b23994eefc7bd2",
  "base_sha": "2aa3ad1693bd95b19de67fd37aea1df8e798b6f1",
  "fetch_ref": "refs/pull/3349/head",     // used when reviewed_sha is not in the clone
  "expected_verdict": "FAIL",
  "expected": [{
    "id": "deleted-clip-test",
    "severity_floor": "Major",
    "anchors_all": ["AFileWithoutAClipFailsClosed"],
    "anchors_any": ["delet", "remov", "does not exist"],
    "confirmed_by": "restored in the round-4 commit; verification.md round-2 disposition"
  }],
  "optional": [],     // REAL by the reviewer's label, never confirmed: matching earns credit, missing costs nothing
  "decoys": [],       // claims known to be false; a match is a false positive
  "noise": []         // traces of the replay environment; matched rows are discarded
}
```

- **Anchors, not prose.** A finding matches when its findings-table row contains every
  `anchors_all` token and at least one `anchors_any` token, case-insensitively. Each row matches at
  most one expected finding and each expected finding at most one row (a one-to-one assignment).
  The AI-055 hand test is why: one symbol named two different findings.
- **`expected` holds only confirmed findings.** A finding counts as confirmed when a later commit
  fixed it, `verification.md` dispositioned it as real, or the owner adjudicated it. A finding
  that only carries the reviewer's own `REAL` label goes in `optional`.
- **The key is self-consistent by construction.** Graded against its own `source_review`, a case
  must score full recall with no decoy hit. `dotf eval check` asserts this for every case in the
  current repository, with no model involved, and each repository's CI runs it. It is the guard that keeps
  anchors from rotting.
- **Planted cases** carry their input under `harness/evals/planted/<id>/` and use the same
  `expected`/`decoys` fields. #3476's prompt is case one, and its false bash 3.2 claim is its first
  decoy.
- **What the tree already records is not expected.** A finding that the replayed tree already dispositions (in `verification.md` or `tasks.md`) goes in `optional`, never in `expected`. A reviewer gets credit for repeating it, and no penalty for skipping it.
- **Archived specs are never edited.** Reviewer artifacts are untouchable, so the key lives beside
  the archive, never inside it.

### 2. How a replay is built

1. Resolve `reviewed_sha`. If the clone lacks it, fetch the SHA itself, then `fetch_ref`, then verify. A branch rebased after its review would not carry the SHA on its PR ref.
   If it cannot be resolved, the case fails loudly as infrastructure; it is never scored zero.
2. `git worktree add --detach` it under `$XDG_STATE_HOME/dotf/evals/<run-id>/<case>-<epoch>/`,
   outside every repository tree, and remove it when the run ends.
3. Launch through `internal/spec`'s review launcher in a new **eval slot**:
   - The candidate comes from `harness/evals/candidates.json`, using a pool entry's shape: runner,
     provider, model, secret id. The pool is not read.
   - The base is the case's `base_sha`, passed explicitly instead of resolved.
   - The reviewer writes its verdict to the run directory, never to `specs/`.
4. The replay worktree is ephemeral state that `dotf eval` owns. It sits outside `WORKTREE_ROOT`, so lanes and the session cockpit never see it. A replay never commits. The launcher's stray-write check reports a commit as a defect of the run, and in a lane-managed repository the lane guard would also refuse it, because the worktree holds no lease. No exemption is needed.
5. The tree is the historical commit itself, so nothing is hand-cut. Any trace the environment
   still leaves (today's tools against old code) is listed once in the case's `noise`.

The review gate cannot be satisfied by an eval run (C37), for three reasons:

- the output never lands in a spec folder;
- the recorded reviewer is prefixed `eval:`, which no pool id matches;
- a test in the same PR asserts that `checkReviewGate` refuses an eval record copied into a spec
  folder.

### 3. What is graded deterministically, and what is not

Curation is human, once per case. Grading is mechanical, every run. Graded per run:

| Measure | How |
|---|---|
| Completion | The launcher's exit: answered, stalled, or one of the `--fallback-reason` classes (`rate-limit`, `quota`, `provider-error`, `timeout`, `content-filter`, `empty-answer`). #3273's two models ended `empty-answer` |
| Format | The findings table parses. An unparseable answer is a format failure, scored as such, never as zero findings |
| Verdict match | `PASS` / `PASS-WITH-GAPS` / `FAIL` against `expected_verdict` |
| Recall | Matched `expected` over `expected`, weighted by severity floor |
| False positives | Rows that match a `decoy` |
| Unadjudicated | Rows that match nothing. Reported as a count with the rows; **neither a miss nor a false positive** |
| Cost | Duration, and tokens (input, output, thinking) from the transcript |

Not graded deterministically: whether an unadjudicated finding is true, whether the reasoning
behind a matched finding is right, how well severities are calibrated beyond the floor, and prose
quality. An unadjudicated row goes to the owner. Each disposition becomes an `expected`, `optional`
or `decoy` entry, so the key grows from runs, the way SWE-bench Verified was curated. The
false-positive rate can only be measured against decoys and adjudicated rows, and the report says so.

**Key circularity.** When a candidate is the case's `key_author`, the result is flagged and left
out of that model's reduced score. `nan/deepseek-v4-flash` authored 62 of the archived reviews.

### 4. Runs and budget

- **Epochs.** The default is N=3 per (candidate, case). Results are reduced to: completion rate,
  mean and minimum recall, decoy-hit rate, and median duration and tokens.
- **Plan before spending.** `dotf eval run --plan` prints the matrix (cases × candidates × epochs) and
  an estimate per runner:
  - **NaN:** credits, from the token envelope a case recorded on earlier runs, times the rates in
    `ai/nan/README.md`.
  - **Anthropic:** money, from the price rows `dotf pr route` already holds (ADR-046 §5).
  - **agy:** a run count, because its quota cannot be read.
- **A run refuses above its cap.** The default cap is 30 runs per invocation, and `--max-runs` raises
  it explicitly.
- **Concurrency** is bounded per pool by `model-map.json`'s `concurrency` minus
  `reserve_interactive`, the same bound `dotf agent run` holds (ADR-035 level 2).

### 5. Roles

`role` is a field on each case, and a score is only ever reported per role. The first slice is
`spec-reviewer`. Each later role plugs in as a new case kind with its own launch path and grader;
the runner, result format and proposal stay the same:

| Role | Case kind | Launch path | Grader |
|---|---|---|---|
| spec-reviewer (first slice) | `replay`, `planted` | `internal/spec` eval slot | findings table vs anchors |
| PR-Agent | `pr-replay`: a merged PR's head, with its `## Review triage` dispositions as the key | `pr-agent` CLI on the PR diff, with the member's `pr_agent` block | the same anchor grader over PR-Agent's findings |
| implementer | `task`: SWE-bench shape, a spec at its base commit | `dotf agent run` | executable: the spec's `features.json` verification commands, fail before and pass after |
| daily use | `quick`: short tasks with exact or regex answers | `dotf agent run --tier low` | exact or regex match, plus latency |

### 6. Output

Each run writes to `$XDG_STATE_HOME/dotf/evals/<run-id>/`:

- `results.json`, valid against `harness/evals/result.schema.json`. Every record carries its
  provenance: dotf version, corpus commit, skill digest, runner version, model id and date.
- `report.md`, the per-role table.
- `proposal.md`, the pool or `model-map.json` change the results support, with the evidence lines a
  `why` would cite.

The runner exits non-zero when it could not answer: a case did not resolve, the plan was refused,
or a launch path is missing. It does not exit non-zero because a model scored low. It never writes
the pool, the map or the pins (C36). The owner's decision, and the PR that carries it, cite the
run's report the way #3482 cites #3476. `dotf eval report --comment <issue>` posts the table.

### 7. Where it runs

On demand, `dotf eval run` on the owner's workstation. agy authenticates with an interactive OAuth
login that CI does not hold, and its quota is the owner's. A scheduled run in ci-workflows is
**deferred**, with these triggers:

- the corpus holds ten or more cases;
- an NaN and Anthropic-only matrix costs less than a stated monthly ceiling;
- the pool canary (#2352) reports a provider catalogue change. Until then, that report opens an
  issue suggesting `dotf eval run`.

### 8. What happens to the existing pieces

- **`scripts/nan-quality-bench.sh` is retired** in the slice that ships `dotf eval run`, together
  with its references (`tests/nan-scripts-secrets.bats`, `ai/opencode/README.md`). It sends four
  ungraded prompts to three NaN models, and the `quick` kind replaces its intent.
- **`tests/evals/harness-evals.json` is retired** in the same slice. Nothing has read it since it
  was added (HARNESS-086, 2026-08-27), and its `expectations` are prose that only a judge could grade.

## Consequences

**Positive**

- A measurement becomes a command plus a commit. #3476's hand construction, and the artifacts it
  produced, are gone.
- A replay sees the round's own state, including the deleted tests #3476 could not see.
- Every admission in the pool's `$comment` can cite a reproducible run.

**Negative**

- Curating a key is human work, not yet measured: writing anchors once per case, then
  adjudicating unmatched rows after runs.
- Anchors couple to vocabulary. A reviewer that describes a defect without naming its symbol
  scores as unadjudicated, not as a match, until someone adjudicates it.
- A replay runs today's skill and tools against old code. Results compare models with each other,
  not with the reviewer that originally ran.

**Neutral**

- `internal/spec` gains an eval slot, an internal API that is not a user flag. The surface of
  `dotf spec review` does not change.

## Decisions for the owner

1. **Build or adopt.** (A) Build in Go, as decided above. (B) Adopt inspect-ai and amend ADR-020 §2.
   *Recommended: A.* With either framework, the solver and the grader are still ours.
2. **What counts toward recall.** (A) Only confirmed findings; label-only `REAL` goes in `optional`.
   (B) Every finding labelled `REAL`.
   *Recommended: A.* B rewards agreeing with the key author's mistakes.
3. **Budget defaults.** (A) N=3 epochs and a cap of 30 runs per invocation. (B) N=5 and 60.
   *Recommended: A* for the first slice: (5 replays + 2 planted) × 3 candidates × 3 epochs = 63 runs,
   so it is planned in three invocations or raised explicitly.
4. **Retirements.** Retire `nan-quality-bench.sh` and `tests/evals/harness-evals.json` in the
   `dotf eval run` slice: (A) yes, (B) keep them.
   *Recommended: A.*
5. **Scheduled runs.** (A) Defer, with the triggers in §7. (B) Schedule NaN and Anthropic runs in
   ci-workflows now.
   *Recommended: A.*
6. **Corpus scope.**
   - Measured: kubelab has 73 rounds, 33 of them in the clone, and 16 Blocker or Major REAL rows. 15 of its keys come from the retired `gemini-3.1-pro-high` and `mimo-v2.5`.
   - Measured: ts-bridge has 31 rounds, 24 in the clone, and 15 rows of the same kind. 27 of its reviews are by `deepseek-v4-flash`.
   - All three repositories use the same findings-table header.

   (A) Dotfiles only in the first slice, with the schema already carrying `repo`. kubelab and ts-bridge follow in their own PRs, with `dotf eval check` in their CI.
   (B) Include kubelab now.
   *Recommended: A.* The corpus is limited by curation time, not raw material. Proving the key format on 5 dotfiles cases first means a schema change is not paid for three times. kubelab is the next corpus, for its keys from retired models.

## References

- dotfiles#1399 (and its 2026-11-13 refresh), #3476, #3482, #3481, #2385, #3273, #3361, #2352
- ADR-020 §2, ADR-035 §3, ADR-043, ADR-046
- `harness/reviewer-pool.json`, `harness/model-map.json`, `harness/model-pins.json`, `ai/nan/README.md`
- `cli/internal/spec/review_request.go` (`ResolveReviewBaseDetail`), `reviewer_pool.go`, `review_launch.go`
- Vault: `00_meta/patterns/pattern-llm-evals.md`; audit matrix `10_projects/dotfiles/prestudy/harness101-eval-reference-audit-matrix.md`
- inspect-ai (`Task`, `Epochs`, per-sample sandbox), promptfoo (`exec:` providers, `--repeat`), SWE-bench harness (instance = repo + base commit + tests), OpenAI evals (deterministic match templates)
