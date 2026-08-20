---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - MEMORY-001-mirror

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/memory-001-cross-agent-mirror`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] Document cross-agent memory guide in `docs/runbooks/guide-cross-agent-memory.md`
- [x] Verify Claude Code `SessionEnd` hook in `ai/claude/settings.json` invoking `dotf mem session-end`
- [x] Verify OpenCode `/handoff` slash command deployed from harness skills (`~/.config/opencode/commands/handoff.md`)
- [x] Verify AGY `/handoff` skill execution in harness
- [x] Document Copilot CLI manual handoff design decision (daemon hooks excluded)
- [x] Validate cross-OS parity via Go CLI `dotf mem` test suite

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/MEMORY-001-mirror/features.json`):

```json
[
  {
    "id": "MEMORY-001-mirror-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
