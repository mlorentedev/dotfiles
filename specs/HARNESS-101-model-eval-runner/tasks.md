---
tags: [spec, tasks, templates]
created: "2026-11-13"
id: "HARNESS-101-model-eval-runner"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-101-model-eval-runner

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Owner answers ADR-063's decisions (delegated by the owner, 2026-11-13); `proposal.md` moves to `implementing`
- [ ] Branch per slice from main (`feat/eval-corpus`, `feat/eval-slot`, `feat/eval-run`, `chore/eval-retire`)

## Implementation

> Four slices, one PR each, in this order. Each slice is TDD: the failing test first.

### Slice 1: corpus and key (no model involved)

- [x] [AC1] Failing test: a case missing `reviewed_sha`, or carrying an unknown `kind`, is rejected by `case.schema.json`
- [x] [AC1] Write `harness/evals/case.schema.json` and the loader in `cli/internal/evals/`
- [x] [AC2] Failing test: the anchor grader, table-driven, covering a one-to-one assignment when two findings share a symbol (the AI-055 round-2 shape)
- [x] [AC2] Implement the findings-table parser and the anchor grader
- [x] [P] [AC1] [AC2] Encode 5 replay cases in `.dotf/evals/cases/` from multi-round archived specs (AI-055 round 2 first), only confirmed findings in `expected`, and nothing the replayed tree already dispositions; record `key_author` and `fetch_ref`; choose cases so each candidate has at least three it did not author (take them from the 27 `agy/gemini-3.1-pro-high` and 14 `nan/mimo-v2.5` reviews, both retired)
- [x] [P] [AC1] [AC2] Encode 2 planted cases: #3476's `((count++))` prompt with the bash 3.2 decoy, plus `go-first-error-race` with the Go 1.22 loop-variable decoy
- [x] [AC2] `dotf eval check`: every case in `.dotf/evals/cases/` scores full recall and zero decoys against its own source; wire it into this repository's CI

### Slice 2: eval slot in the review launcher

- [x] [AC4] Failing test: `checkReviewGate` refuses a review recorded as `eval:<id>` inside a spec folder
- [x] [AC4] Failing test: an eval launch with an explicit base and candidate reads no pool file and writes nothing under `specs/`
- [x] [AC4] Implement the eval slot (`internal/spec`): candidate from `harness/evals/candidates.json`, base passed explicitly, output directory supplied by the caller
- [x] [AC3] Failing test: a case whose `reviewed_sha` is absent locally is fetched from `fetch_ref`; an unresolvable one exits non-zero naming the case
- [x] [AC3] Implement replay checkout: a detached worktree at `<run dir>/<case>-<epoch>/`, removed afterwards; the run dir is the caller's (the launcher passes `<run>/trees/<candidate>/`)

### Slice 3: `dotf eval run`

- [ ] [AC3] `dotf eval run` places each run under dotf's state directory (`env.StateDir()`, `$XDG_STATE_HOME/dotfiles/evals/<run-id>/`) and exits non-zero naming a case whose commit cannot be resolved
- [x] [AC6] Failing test: `--plan` prints the matrix and per-runner estimate; a matrix above `--max-runs` is refused
- [x] [AC6] Implement the plan and the cap, for NaN (credits) and agy (run count); rates in `candidates.json`, held to `ai/nan/README.md` by a test
- [ ] [AC6] Anthropic estimate in money from `dotf pr route`'s price rows, once a candidate runs on Anthropic (none does today)
- [ ] [AC6] `measured` envelopes: the median of a case's earlier runs, read from stored `results.json` (needs AC7)
- [x] [AC5] Launch path for a planted case: it has no commit or tree, only `prompt.md`; decide how the eval slot sends it (ADR-063 §1). `spec.PlanPlantedLaunch`, answer at `.dotf-eval/review.md` in an empty tree
- [ ] [AC5] Concurrency bounded per pool by `model-map.json`, the bound `dotf agent run` holds
- [x] [AC5] Failing test: per-run grading record, with completion class mapped from the launcher's exit, and format failure distinct from zero findings. `evals.Run` with an injected launcher: one attempt at a time, `record.json` before the next, `infra-error` never charged to the model, own-key flag, interrupt passed to the runner in flight
- [x] [AC5] The real launcher (`evals.ExecLauncher`): the tree is a replay checkout or an empty dir with `.dotf-eval/` made up front, removed however the attempt ends; the runner starts in the tree (`spec.Limits.Dir`) under `spec.RunWithLimits`, its stdout filtered into the transcript by `spec.SinkTranscript`; a deadline, a stall and a signal (`spec.RunResult.Signaled`) map to their classes; the answer is copied out without following a symlink. Tokens stay "not reported": no transcript usage reader exists yet
- [x] [AC5] Preflight (`evals.Preflight`): every commit resolved, every runner on PATH, every secret checked by exit status, and a replay refused when its reviewed commit holds its `source_review` or its case file; all problems named at once
- [ ] [AC5] Tokens (input, output, thinking) read from each runner's transcript
- [ ] [AC5] Flag a run whose transcript shows it reading a commit after `reviewed_sha` (ADR-063, known limit)
- [ ] [AC3] Wire `dotf eval run` to `Preflight`, `ExecLauncher` and `Run` (real `LookPath`, secrets checked through `dotf secrets run`), and keep it refusing to start without an approved `--plan`
- [ ] [AC5] Tell `rate-limit`, `quota` and `content-filter` apart from `provider-error` by the transcript
- [ ] [AC5] A request record for a planted run (`WriteEvalRequest` takes only an `EvalLaunch`), with the `prompt.md` digest
- [ ] [AC5] Remove trees an earlier interrupted run left under `<run>/trees/`
- [ ] [AC5] Record an attempt's stray writes in its replay tree (`spec.SnapshotTree` before and after, plus `HEAD`, which a status snapshot cannot see move), as `spec review` reports them
- [ ] [AC3] The run's interrupt context covers the gaps between attempts, and the stall timeout is set per runner so a silent `agy --print` is not cut as stalled (`ExecLauncher.Stall` zero means the 10-minute default; no value turns the detector off today)
- [ ] [AC3] A replay checkout under lanes: check what `dotf lane status` and `dotf lane doctor` report for a worktree the run adds
- [ ] [AC5] Implement run, grade and epoch reduction; flag results on a candidate's own keys
- [ ] [AC7] Failing test: `results.json` validates against `result.schema.json` with the provenance fields; no code path opens the pool, map or pins for writing
- [ ] [AC7] Implement `results.json`, `report.md`, `proposal.md`, and `dotf eval report --comment <issue>`
- [ ] First real run: hydrogen-high, deepseek-v4-flash, glm5.3-flash on the corpus; post the report on #1399

### Slice 4: retirements

- [ ] [AC8] Remove `scripts/nan-quality-bench.sh` and `tests/evals/harness-evals.json`, and their references in `tests/nan-scripts-secrets.bats`, `ai/opencode/README.md` (spec folders and ADRs that mention them are history and stay); run every suite that named them (`git grep -l` over `tests/`)

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-101-model-eval-runner/features.json`):

```json
[
  {
    "id": "HARNESS-101-model-eval-runner-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
