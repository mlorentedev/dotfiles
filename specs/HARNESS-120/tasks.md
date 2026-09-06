---
tags: [spec, tasks, templates]
created: "2026-09-05"
---

# Tasks - HARNESS-120

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/harness-120-agent-auto`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> The resolution layer (`harness`) is a pure function and lands first; the
> command (`cmd`) is wiring over it and lands second. That split is not
> stylistic — every piece this composes already pushes its I/O to the caller
> (`ResolveRoles` takes loaded personas, `Dispatch` takes a resolved chain), so
> the new logic can be tested without a filesystem, a pool or a git repo.

### 1. The missing link: a persona's declared tier

- [ ] [P] [AC1] Failing test: `ResolveTierForPersona` returns `mid` for a
      persona whose record says `model: mid`
- [ ] [AC1] Implement `harness.ResolveTierForPersona` — the first reader of
      `Persona.Model`, which `persona.go:107` has been writing to nobody
- [ ] [P] [AC4] Failing test: a persona with an empty `model:`, and one with
      `model: enormous`, are each REFUSED with a message naming the persona,
      the value, and the three legal tiers — never defaulted
- [ ] [AC4] Implement the refusal

### 2. Resolving one persona from a task, or refusing

- [ ] [P] [AC1] Failing test: `ResolveOne` over a suggestion matching one
      persona returns that persona plus the pattern that matched it
- [ ] [AC1] Implement `harness.ResolveOne` as a pure function over
      `(Suggestion, []*Persona)`, reusing `ResolveRoles` — never re-deriving
      the join, per the warning at `roles.go:27-31`
- [ ] [P] [AC2] Failing test: a suggestion resolving to `[builder, reviewer]`
      returns an ambiguity error naming BOTH, and no persona
- [ ] [AC2] Implement — the refusal is the deterministic behaviour; do not rank
      and do not tie-break (HARNESS-110)
- [ ] [P] [AC3] Failing test: a suggestion matching no rule returns an error
      distinguishable from the ambiguous one (assert the two are not equal,
      not that either has a fixed wording)
- [ ] [AC3] Implement
- [ ] Refactor: one error type carrying the candidates, so the command can
      render both cases without re-testing the string

### 3. The command

- [ ] [AC1] Failing test: `dotf agent auto --task "…" --backend dry-run` with
      neither `--role` nor `--tier` emits one JSON object whose `role` and
      `tier` come from the join and the record
- [ ] [AC1] Implement `newAgentAutoCmd`, wiring
      Suggest → ResolveOne → ResolveTierForPersona → `agent.Dispatch`, reusing
      the machine-policy, semaphore and capacity setup `agent run` already does
- [ ] Refactor: extract the shared dispatch preamble out of `newAgentRunCmd` so
      denial, semaphore and capacity are configured in ONE place — two callers
      of a fail-closed policy is two places it can be forgotten
- [ ] [AC5] Failing test: `--role` skips the join entirely and `--tier`
      overrides the record, with both marked in the output as dictated
- [ ] [AC5] Implement the overrides and the `resolved_by` field
- [ ] [AC2] [AC3] Failing test: both refusals exit non-zero and dispatch
      NOTHING (assert against a backend that records its calls, not against
      stdout)
- [ ] [AC2] [AC3] Implement

### 4. Optional isolation

- [ ] [AC6] Failing test: `--worktree <slug>` creates the worktree, passes a
      cwd inside it, reports path and branch in the JSON, and the worktree
      still exists after a FAILED dispatch
- [ ] [AC6] Implement over `worktree.Add`, in-process — never shelling out to
      `dotf worktree add`
- [ ] [AC6] Guard: a test asserting `auto` calls nothing that removes a
      worktree. The teardown is a human-authorised `dotf worktree done`, and
      the reason is data loss, so it needs an assertion and not a convention

### 5. Evidence

- [ ] [AC7] One real dispatch, quoted verbatim in `verification.md` with the
      pool that answered and the tier it came from

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] `cd cli && go build ./... && go vet ./... && go test ./...`
- [ ] `GOOS=windows go vet ./...` — the Windows leg fails the whole package
- [ ] `golangci-lint run` at the pin from `versions.conf`
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] Independent adversarial review before `spec archive`

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.
