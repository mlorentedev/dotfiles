---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - SDD-011-agent-side-spec-trigger

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/sdd-011-agent-side-spec-trigger` (worktree-isolated)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (all three resolved/mitigated in-scope)

## Implementation

- [x] Add SDD-011 vault-gate entry to `vault/10_projects/dotfiles/11-tasks.md` (the vault gate `init-spec.sh` enforces)
- [x] Author the `## Agent-Side Activation Rule` section in vault SSOT `00_meta/skills/spec/SKILL.md` — the checks the agent runs, the proposal format, and the "when NOT to propose" guardrails + once-per-change debounce (AC1)
- [x] Add the always-on proactive trigger + pointer to `AGENTS.md` Discipline Gate, with `do not duplicate it here` to keep the detail single-sourced in the skill (AC2)
- [x] Regenerate the committed record via `compile-harness.sh --refresh`; confirm minimal diff (only `harness/skills/spec/SKILL.md`) and `--check` green (AC3)
- [x] Add a bats regression assert that the deployed `/spec` (claude + opencode) carries the rule heading — guards against a future record regen from a SKILL.md missing the section (AC4)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test/check
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Lint passes (no shell scripts modified; `--check` + bats green)
- [x] No unrelated changes in the diff (no scope creep) — verified `git diff --stat` = AGENTS.md + record only
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.
