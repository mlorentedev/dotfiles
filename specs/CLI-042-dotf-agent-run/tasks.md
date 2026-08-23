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

- [ ] Branch created from main: `feat/dotf-agent-run`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions" — **two are still open**: the
      backend tie-break for a `nan` chain entry, and the secret-delivery shape for the hive daemon
- [ ] Issue opened on `mlorentedev/hive` for PR A, linked as blocking #1190

## Implementation

### PR B — the command and its contract

- [ ] [P] [AC1] Failing test: `dotf agent run` writes one JSON object to stdout with `status`,
      `pool`, `model`, `exit`, `duration_ms`, `output`, and nothing but JSON on stdout
- [ ] [AC1] Implement the command, its flags (`--role`, `--task`, `--tier`, `--cwd`, `--timeout`,
      `--backend`) and the JSON encoder; every log line goes to stderr
- [ ] [P] [AC2] Failing test: a fake backend scripted to return *pool unavailable* advances to the
      next `chains` entry; one scripted to return *task failed* does not
- [ ] [AC2] Implement the chain walk over `harness.ResolveChain`, and the error classification that
      keeps the two cases distinct
- [ ] [AC2] Failing test + implement: chain exhausted — every entry unavailable — is its own outcome,
      distinguishable from a task failure
- [ ] [P] [AC5] Failing test: `--tier top` with `claude:opus` unavailable escalates and exits
      non-zero, and never resolves a mid-tier model
- [ ] [AC5] Implement the top-tier no-degrade rule
- [ ] Refactor: the backend interface is the seam — one method, the five semantics, no backend-specific
      types leaking into the dispatcher

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
