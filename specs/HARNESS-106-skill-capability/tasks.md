---
tags: [spec, tasks]
created: "2026-09-01"
---

# Tasks - HARNESS-106-skill-capability

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/persona-skill-enforcement`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> Two independent behaviours: the **capability** (map + resolution) and the
> **record-level guard** (persona frontmatter). Their first test tasks carry `[P]`.
> The decision record (AC5–AC7) is a third, sequenced last because AC4 cannot be
> proven without it — a dispatch leaves no evidence until the gate writes one.

### The capability and its resolution

- [x] [P] [AC1] Write failing test that the shipped map grants a skill verb for claude (`TestShippedMapGrantsTheSkillCapabilityWhereItExists`)
- [x] [AC1] Add `skill` to `vocabulary` and to `claude.capabilities` in `harness/capability-map.json`; bump the map to version 2
- [x] [AC2] Write failing loader test for a verb both mapped and declared `unsupported`, and for an `unsupported` verb outside the vocabulary
- [x] [AC2] Add the `unsupported` property to `capability-map.schema.json` and enforce both rules in `checkHarnessCoversVocabulary`
- [x] [AC2] Add `UnsupportedFor` as a separate exported function so `ResolveCapabilities` stays pure, and report the omission on **stderr** — stdout is substituted into a frontmatter line
- [x] [AC2] Declare `opencode.unsupported = ["skill"]`: measured against `@opencode-ai/plugin`, its permission vocabulary has no skill concept

### The record-level guard

- [x] [P] [AC3] Write failing guard that every persona declaring `skills:` also declares the `skill` capability (`TestEveryPersonaDeclaringSkillsCanInvokeThem`) — it named all seven, red on the real defect
- [x] [AC3] Read `capabilities:` in `LoadPersona` so the tie between the two frontmatter keys is assertable at all
- [x] [AC3] Fix in the vault SSOT — `00_meta/agents/definitions/<role>/AGENT.md` ×7 — never in the generated files

### Propagation and compatibility

- [x] [AC8] Verify a scratch-`$HOME` deploy renders `Skill` for all seven personas, and that a second run is byte-identical (`changed=0`)
- [x] [AC8] Make `compile-harness.sh` name its own fix when an older `dotf` rejects the v2 map — it fails closed with `NO agent deployed`, and the message blamed the (correct) map

### The decision record — deferred, tracked here

- [ ] [AC5] `dotf harness gate` writes a durable record for every decision, including `allow` and `role did not resolve`
- [ ] [AC6] A `warn` decision is readable from that record after the session ends
- [ ] [AC7] `agent_type` is carried in the record, converting the standing inference into a measurement
- [ ] [AC4] Re-run a real dispatch and confirm a consumption record — the criterion AC5–AC7 exist to make provable

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test **or** explicitly carried as pending
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass (`go build`, `go vet`, `GOOS=windows go vet`)
- [x] Lint passes (`golangci-lint`, `shellcheck`)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#1428)

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

The nine entries map 1:1 onto the nine acceptance criteria. Five are
`implemented` (code landed, harness has not run them); four are `pending` and
their verification commands are red, which is the honest state for a criterion
whose implementation is deferred — not a vacuous `exit 1`.
