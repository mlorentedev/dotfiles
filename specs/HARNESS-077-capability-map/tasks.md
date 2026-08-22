---
tags: [spec, tasks, templates]
created: "2026-08-22"
---

# Tasks - HARNESS-077-capability-map

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/capability-map`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left — the two native shapes were verified against the shipping tools, and
      the scope cut (doctor checks, rule-plumbing refactor) is recorded in Out of scope with reasons

## Implementation

- [x] [AC5] Failing Go test: an absent / unschema'd / non-JSON / partial-coverage map all refuse
- [x] [AC5] `capability_map.go`: loader + `ValidateCapabilityMap` + `checkVocabularyCoverage`
- [x] [AC1][AC2] Failing Go table test: `csv` renders an allow-list, `decision-map` a flow mapping
- [x] [AC1][AC2] `ResolveCapabilities` — resolves a SET, returns the whole frontmatter line
- [x] [AC3][AC4] Failing Go test: unmapped capability and undeclared harness each name themselves
- [x] [AC3][AC4] Return the loader's phrasing unchanged; do not re-wrap
- [x] `dotf harness resolve-capabilities` wired under `dotf harness`
- [x] Generalise the shell capability probe over the subcommand name; retarget the Go tripwire test
      at BOTH subcommands, since each field probes independently
- [x] [AC6] Failing bats: `capabilities: [read, search, edit]` renders native `tools:`
- [x] [AC6] `agent_capability_line` + `render_agent` emits it
- [x] [AC7] Failing bats: a record declaring no capabilities renders without the field
- [x] [AC8] Failing bats: a dotf knowing resolve-tier but not resolve-capabilities warns for that
      field only, and the model line still resolves
- [x] #1168: a whitespace-bearing model id blames the map, not a stale dotf (own exit status)
- [x] #1169: collect failing records and report them all, instead of aborting on the first
- [x] [AC9] Real-binary cases in `tests/compile-harness-real.bats`, including that the two native
      forms genuinely differ

## Closing

- [x] Every acceptance criterion covered by at least one test
- [x] `features.json` entries propagate the runner's exit status and pin tests by unique name
- [x] `go build` / `go vet` / `GOOS=windows go vet` / `go test ./...` green
- [x] `golangci-lint run` -> 0 issues at the `versions.conf` pin; `gofmt` clean
- [x] `shellcheck --severity=error` (CI's level) clean; `bash -n` and `zsh -n` clean
- [x] `bats tests/*.bats` -> 1429 passing, 0 failing
- [x] Scope kept to one idea; the doctor checks and the rule-plumbing refactor are filed, not folded
- [x] `verification.md` filled in with this session's output
- [x] PR opened referencing this spec folder
