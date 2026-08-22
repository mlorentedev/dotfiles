---
tags: [spec, tasks, templates]
created: "2026-08-22"
---

# Tasks - HARNESS-076-model-map-tier-render

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/model-map-tier-render`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — the `dotf` dependency is decided and
      recorded with its rejected alternative; the reviewer can reverse it at one seam

## Implementation

> The Go subcommand (behaviors 1–3) and the shell render (behaviors 4–7) are independent until the wire-up,
> so each chain's first test carries `[P]`.

- [x] [P] [AC1] Failing Go table test: `resolve-tier top --harness claude` prints `opus`, exit 0
- [x] [AC1] Add `cli/internal/harness/cmd_resolve_tier.go` wiring `LoadModelMap` + `ResolveTier` to a cobra
      subcommand under `dotf harness`
- [x] [AC2] Failing test: an undeclared tier/harness pair exits non-zero, names both, writes nothing to stdout
- [x] [AC2] Return the loader's error unchanged — `ResolveTier` already phrases both cases; do not re-wrap
- [x] [AC3] Failing test: an absent and a schema-invalid `harness/model-map.json` both exit non-zero (C15)
- [x] Refactor: keep the command body thin; resolution stays in `model_map.go`, never duplicated here
- [x] [P] [AC4] Failing bats: `render_agent` on a record with `model: top` emits `model: opus`
- [x] [AC4] Resolve the tier in `deploy_agents` (extracted to `agent_model_line`) and pass the
      finished line into `render_agent`, which stays a pure renderer
- [x] [AC5] Failing bats: a record whose tier the map cannot answer fails the render non-zero, naming both
- [x] [AC5] Propagate the resolver's exit status out of `agent_model_line` into the deploy loop
- [x] [AC6] Failing bats: skill deploy still succeeds when the agent render fails and when `dotf` is off PATH
- [x] [AC6] Scope the failure to `deploy_agents`, leaving `deploy_skills` untouched
- [x] [AC7] Assert `kind`, `capabilities`, `skills`, `targets` are still dropped — only `model` changed
- [x] Update `harness/agent-frontmatter.schema.json`'s `model` description: it documents a mapping that now
      happens, and its "Dropped from the slice render" clause is no longer true for agent records

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `cd cli && go build ./... && go vet ./... && go test ./...` green
- [x] `GOOS=windows go vet ./...` green (the Windows leg compiles the same tree)
- [x] `golangci-lint` at the `versions.conf` pin reports clean
- [x] `shellcheck scripts/compile-harness.sh` and `bats tests/*.bats` green under bash **and** zsh
- [x] No unrelated changes in the diff, with one declared exception: `ai/claude/settings.json`
      gains `outputStyle` at the repo owner's explicit request (see proposal "Out of scope")
- [x] `verification.md` filled in with this session's command output
- [x] PR opened referencing this spec folder
- [x] `dotf spec review HARNESS-076-model-map-tier-render` run on `nan/deepseek-v4-flash`;
      verdict PASS-WITH-GAPS, all six findings dispositioned (see the PR's `## Review triage`)

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.
