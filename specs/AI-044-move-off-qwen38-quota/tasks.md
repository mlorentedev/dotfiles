---
tags: [spec, tasks, templates]
created: "2026-09-26"
---

# Tasks - AI-044-move-off-qwen38-quota

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/nan-quota-default-model`
- [x] `proposal.md` is complete and acceptance criteria are testable

## Implementation

- [x] [AC1] Change the default-model test to `glm5.3-flash` and `qwen3.6` titles; it fails on main
- [x] [AC1] Route opencode's `model`, `small_model` and `agent.plan`
- [x] [AC2] Add the opencode/pi context-window test; on main it fails on four models
- [x] [AC2] Align opencode's windows with pi's (`qwen3.8-flash`, `qwen3.6`, `gemma4`, `mimo-v2.5`)
- [x] [AC3] Re-head the low tier and chain; rename the rerank service
- [x] Refresh `ai/nan/README.md` and `.claude/CLAUDE.md`
