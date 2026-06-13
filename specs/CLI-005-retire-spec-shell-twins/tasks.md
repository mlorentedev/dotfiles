---
tags: [spec, tasks, templates]
created: "2026-06-13"
---

# Tasks - CLI-005-retire-spec-shell-twins

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `chore/retire-spec-shell-twins` (worktree)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"
- [ ] **GATE: CLI-009 #365 merged** (dot installs on PATH) — do not start deletions until green

## Implementation

> Mostly deletions + reference repointing. Update the test (`agents-md.bats`) in lockstep with the AGENTS.md edit. After #365 merges, rebase this branch onto fresh main first.

- [ ] Rebase `chore/retire-spec-shell-twins` onto post-#365 main
- [ ] Update `tests/agents-md.bats:34` to assert `dot spec init` (red until AGENTS.md edited)
- [ ] Edit `AGENTS.md` §389 + §406 to point to `dot spec init` / `dot spec archive` (greens agents-md.bats)
- [ ] `git rm` the 4 shells: `scripts/{init-spec,archive-spec}.{sh,ps1}`
- [ ] `git rm tests/init-spec.bats`
- [ ] Update `harness/skills/spec/SKILL.md` invocations → `dot spec`
- [ ] Update `scripts/check-spec-gate.sh:193` hint string → `dot spec init`
- [ ] Update `docs/adr/dotfiles-architecture-map.md` rows naming the scripts
- [ ] Guard: `grep -rE 'init-spec|archive-spec'` returns only CHANGELOG / audits / spec-provenance docs
- [ ] Run full `bats tests/*.bats` (minus init-spec.bats) + `shellcheck`; smoke `dot spec init/archive`

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-005-retire-spec-shell-twins/features.json`):

```json
[
  {
    "id": "CLI-005-retire-spec-shell-twins-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
