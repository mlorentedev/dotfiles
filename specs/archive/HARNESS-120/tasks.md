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
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> The resolution layer (`harness`) is a pure function and lands first; the
> command (`cmd`) is wiring over it and lands second. That split is not
> stylistic — every piece this composes already pushes its I/O to the caller
> (`ResolveRoles` takes loaded personas, `Dispatch` takes a resolved chain), so
> the new logic can be tested without a filesystem, a pool or a git repo.

### 1. The missing link: a persona's declared tier

- [x] [P] [AC1] Failing test: `ResolveTierForPersona` returns `mid` for a
      persona whose record says `model: mid`
- [x] [AC1] Implement `harness.ResolveTierForPersona` — the first reader of
      `Persona.Model`, which `persona.go:107` has been writing to nobody
- [x] [P] [AC4] Failing test: a persona with an empty `model:`, and one with
      `model: enormous`, are each REFUSED with a message naming the persona,
      the value, and the three legal tiers — never defaulted
- [x] [AC4] Implement the refusal

### 2. Resolving one persona from a task, or refusing

- [x] [P] [AC1] Failing test: `ResolveOne` over a suggestion matching one
      persona returns that persona plus the pattern that matched it
- [x] [AC1] Implement `harness.ResolveOne` as a pure function over
      `(Suggestion, []*Persona)`, reusing `ResolveRoles` — never re-deriving
      the join, per the warning at `roles.go:27-31`
- [x] [P] [AC2] Failing test: a suggestion resolving to `[builder, reviewer]`
      returns an ambiguity error naming BOTH, and no persona
- [x] [AC2] Implement — the refusal is the deterministic behaviour; do not rank
      and do not tie-break (HARNESS-110)
- [x] [P] [AC3] Failing test: a suggestion matching no rule returns an error
      distinguishable from the ambiguous one (assert the two are not equal,
      not that either has a fixed wording)
- [x] [AC3] Implement
- [x] Refactor: one error type carrying the candidates, so the command can
      render both cases without re-testing the string

### 3. Making the persona travel

> The measurement this rests on: `Request.Role` is set at `dispatch.go:136` and
> read only by `dryrun.go:22`. Neither real backend sends it. So a dispatch is
> currently generic, and this is the section that changes that.

- [x] [P] [AC6] Failing test: `PersonaPreamble` renders a persona's record —
      body, not frontmatter — with a delimiter, and the task after it
- [x] [AC6] Implement, reading the body from `Persona.Path` via the existing
      `frontmatterBlock` splitter rather than a second parser
- [x] [AC6] Failing test at the dispatch seam: with `--role reviewer` the
      `Request.Task` the backend RECEIVES contains reviewer's mandate; with
      `--role builder` it does not. Assert on the backend's received request —
      a test reading stdout would pass on a preamble that was built and dropped
- [x] [AC6] Wire the preamble into the task at the command layer, not inside
      `Dispatch` — the walk retries across pools and must send identical bytes
      each time, so composition happens once, before the walk

### 4. The command

- [x] [AC1] Failing test: `dotf agent auto --task "open a ticket for the
      bitacora" --backend dry-run` with neither `--role` nor `--tier` reports
      `role: planner`, `tier: mid`
- [x] [AC1] Implement `newAgentAutoCmd`, wiring
      Suggest → ResolveOne → ResolveTierForPersona → preamble →
      `agent.Dispatch`, reusing the machine-policy, semaphore and capacity
      setup `agent run` already does
- [x] Refactor: extract that shared setup out of `newAgentRunCmd` so denial,
      semaphore and capacity are configured in ONE place — two callers of a
      fail-closed policy is two places it can be forgotten
- [x] [AC5] Failing test: `--role` skips the join entirely and `--tier`
      overrides the record, with both marked in the output as dictated
- [x] [AC5] Implement the overrides and the inferred/dictated reporting
- [x] [AC2] [AC3] Failing test: both refusals exit non-zero and dispatch
      NOTHING (assert against a backend that records its calls, not against
      stdout)
- [x] [AC2] [AC3] Implement
- [x] Personas and the model map load from `env.ResolveHarnessRoot()` with a
      `--repo-root` override matching `run`'s — never the cwd, which is the
      trap `loadGatePersona` already avoids

### 5. Evidence

- [x] [AC7] One real dispatch, quoted verbatim in `verification.md` with the
      pool that answered and the tier it came from. Done 2026-09-06: `claude:sonnet`
      off `chains.mid`, 12.7 s, `tier_from: inferred` — reviewer's own record chose
      the tier. Unblocked by the owner declaring `machine.id`; the gap that made it
      unrunnable is #1547
- [x] Note the `agent.Record` widening (`role`, `pattern`, inferred/dictated)
      in the PR body: additive for JSON readers, but it is a contract change

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `cd cli && go build ./... && go vet ./... && go test ./...`
- [x] `GOOS=windows go vet ./...` — the Windows leg fails the whole package
- [x] `golangci-lint run` at the pin from `versions.conf`
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#1546)
- [ ] Independent adversarial review before `spec archive`

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.
