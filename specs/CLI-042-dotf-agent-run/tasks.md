---
tags: [spec, tasks, templates]
created: "2026-08-23"
---

# Tasks - CLI-042-dotf-agent-run

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## PR sequence

This spec does not fit one PR, and "first step of a multi-PR sequence" is itself a Discipline Gate
trigger. The split is declared here so no PR silently absorbs the next:

| PR | Repo | Lands | Criteria |
|---|---|---|---|
| **A** | `mlorentedev/hive` | worker becomes NaN-only (Ollama + OpenRouter removed); `hive delegate` dispatch verb, single-shot, `--model` and `--timeout` required | unblocks AC6 |
| **B** | `dotfiles` | `dotf agent run` — flags, JSON contract, chain walk through the existing loader, fake backend | AC1, AC2, AC5 |
| **C** | `dotfiles` | semaphore, timeout + cancellation, fail-closed refusals | AC3, AC4 |
| **D** | `dotfiles` | real backends: subprocess and hive probes, the tie-break, end-to-end smoke | AC6 |
| **E** | `dotfiles` | IaC: hive service unit via `dotf secrets run`, `environment.d` and `mcp_servers.json` cleanup, `dotf doctor` reachability check | AC7, AC8, AC9 |

**A is a hard dependency of D.** It is tracked as `mlorentedev/hive#384` with its own spec,
`specs/HIVE-384-nan-worker-and-delegate-verb/` in that repo, and it ships as `feat!` → **4.0.0**.
What D waits on is not the merge but the **release**: merge → release-please PR → merge → GitHub
Release → PyPI → `uv tool upgrade hive-vault`. A PR merged upstream is not a version this machine
runs.

**The seam's error classification is the cross-repo contract, so it is pinned on both sides.** The
hive verb exits `3` for *pool unavailable* and `1` for *task failed*; this dispatcher advances the
chain on the first and must not on the second. If either side changes those codes unilaterally, a
bad answer becomes a silent retry against a different model — the exact failure ADR-032 §2 names.
The dispatcher must therefore treat an unrecognised exit code as *task failed*, never as
*unavailable*: the fail-closed direction is the one that does not retry.

## Setup

- [x] Branch created from main: `feat/dotf-agent-run`
- [x] `proposal.md` is complete and acceptance criteria are testable — the nine were blessed in PR B
      (2026-08-23), with AC6 and AC8 reworded against what PR A actually shipped
- [x] No open questions left in `proposal.md` "Risks / open questions" **that gate the PR being
      worked**. The two that remain — the backend tie-break for a `nan` chain entry, and the
      secret-delivery shape for the hive daemon — are scoped to PRs D and E respectively, and neither
      is reachable from B or C: there is no second backend to tie-break between until D, and no daemon
      to deliver a secret to until E. They must be closed before their own PR opens, not before this
      one does.
- [x] Issue opened on `mlorentedev/hive` for PR A, linked as blocking #1190 — `mlorentedev/hive#384`,
      shipped and released as `hive-vault` 4.1.0; `hive delegate` is installed and dispatches

## Implementation

### PR B — the command and its contract

- [x] [P] [AC1] Failing test: `dotf agent run` writes one JSON object to stdout with `status`,
      `pool`, `model`, `exit`, `duration_ms`, `output`, and nothing but JSON on stdout
- [x] [AC1] Implement the command, its flags (`--role`, `--task`, `--tier`, `--cwd`, `--timeout`,
      `--backend`) and the JSON encoder; every log line goes to stderr
- [x] [P] [AC2] Failing test: a fake backend scripted to return *pool unavailable* advances to the
      next `chains` entry; one scripted to return *task failed* does not
- [x] [AC2] Implement the chain walk over `harness.ResolveChain`, and the error classification that
      keeps the two cases distinct
- [x] [AC2] Failing test + implement: chain exhausted — every entry unavailable — is its own outcome,
      distinguishable from a task failure
- [x] [P] [AC5] Failing test: `--tier top` with `claude:opus` unavailable escalates and exits
      non-zero, and never resolves a mid-tier model
- [x] [AC5] Implement the top-tier no-degrade rule
- [x] Refactor: the backend interface is the seam — one method, the five semantics, no backend-specific
      types leaking into the dispatcher

Added while implementing B, each because the work surfaced it rather than because it was planned:

- [x] [AC1] Exit-code plumbing: `main()` could only ever exit 1, so `chain_exhausted` was
      indistinguishable from a task failure at the process boundary — the exact collapse AC2 forbids
      one layer down. A tagged error, unwrapped in `main`, rather than the `os.Exit`-inside-`RunE`
      precedent in `secrets.go`, which costs a re-exec harness in its test file.
- [x] [AC1] A `dry-run` backend, so the criterion's bats half runs against something. See the decision
      table in `proposal.md`.
- [x] [AC1] The record's output cap (`agent.OutputCap`) with an in-band `truncated` marker — ADR-032
      §2's "output bounded by a cap" is part of the record contract, so it belongs to B, and a capped
      output that does not say so reads as a complete short answer.
- [x] [AC5] `chains.top` capped at one entry in `model-map.schema.json`. The behaviour rule alone
      would leave a map free to *declare* a fallback the dispatcher silently ignores. **The first
      attempt at this was wrong in an instructive way:** naming `top` under `properties` exempts it
      from `additionalProperties`, so a bare `maxItems` removed the array type, the minimum and the
      `pool:model` pattern — an edit that reads as a tightening and is a loosening. The sibling
      `$ref` is what holds the shape, and `TestChainsTopIsCappedWithoutLosingTheChainShape` fails
      without it.
- [x] [AC1] `agent run` registered in `TestStdoutContracts`, the existing registry of subcommands
      whose output is machine-read.

### PR C — the brakes

- [ ] [P] [AC3] Failing test: a fake backend that outlives its deadline is killed, and the semaphore
      slot is observably free **before** the worker is reaped
- [ ] [AC3] Implement per-dispatch timeout (required, no silent default) and cancellation
- [ ] [P] [AC4] Failing test: an unreadable concurrency counter exits non-zero rather than reading as
      zero in use
- [ ] [AC4] Failing test: a machine whose identity cannot be established denies every non-local pool
- [ ] [AC4] Implement the semaphore over `harness.DeclaredBudget`, and both fail-closed paths
- [ ] [AC4] Assert the wording: help text and error messages state *`dotf` alone will never be the
      cause of exhaustion*, never that exhaustion cannot happen

### PR D — the real backends

- [ ] [AC6] Failing test: backend probe selects `subprocess` when a harness binary is present and
      `hive` when it is not; `--backend` overrides both
- [ ] [AC6] Implement the subprocess backend (`claude -p`, `pi -p`) on the shared process machinery
- [ ] [AC6] Implement the hive backend as `hive delegate` over the **same** process machinery — argv
      is its transport, not a second mechanism
- [ ] [AC6] Resolve and encode the tie-break for a `nan` chain entry servable by both backends
- [ ] [AC6] End-to-end smoke: `dotf agent run --backend hive --tier mid` answers, and reports
      `pool=nan` with the model the chain resolved

### PR E — the deployment, as IaC

- [ ] [AC7] Failing test: the rendered hive service unit invokes `dotf secrets run -- hive serve`
- [ ] [AC7] Implement the unit and its deployment in `setup-linux.sh`, verified idempotent
      (`changed=0` on re-run)
- [ ] [AC7] Assert no credential in the deployed `environment.d` fragment; verify the daemon by
      consequence (it answers), never by printing the value
- [ ] [P] [AC8] Remove `HIVE_OLLAMA_ENDPOINT` from `ai/agy/mcp_servers.json` and any other config this
      repo deploys naming Ollama or OpenRouter for hive's worker
- [ ] [AC8] Assertion that the removal stays removed — the incident-to-guard rule
- [ ] [P] [AC9] Failing test: `dotf doctor` fails when a backend probes present but can serve nothing
- [ ] [AC9] Implement the reachability check, so "zero models" is caught at diagnostic time rather
      than under dispatch load

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] `GOOS=windows go vet ./...` clean — the Windows leg compiles the same tree
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-042-dotf-agent-run/features.json`):

```json
[
  {
    "id": "CLI-042-dotf-agent-run-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
