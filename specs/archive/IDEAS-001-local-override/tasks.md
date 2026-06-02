---
tags: [spec, tasks, ideas-001]
created: "2026-05-25"
---

# Tasks - IDEAS-001-local-override

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [ ] Branch created from main: `feat/IDEAS-001-local-override`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions" (R3 explicitly deferred — non-blocker)
- [ ] Confirm current trailing lines of `.zshrc` and `.bashrc` to know exact insertion point

## Implementation

> TDD order. Tests first, then code, then refactor.

- [ ] Write failing bats test `tests/local-override.bats` #1: zsh sources `.zshrc.local` if present (sets sentinel env var, subshell asserts).
- [ ] Write failing bats test #2: zsh sources `.zshrc` cleanly with no `.local` file (exit 0, no stderr).
- [ ] Append guarded source line to `.zshrc` (last non-blank line). Test #1 + #2 pass.
- [ ] Write failing bats test #3 + #4: same for bash + `.bashrc.local`.
- [ ] Append guarded source line to `.bashrc`. Tests #3 + #4 pass.
- [ ] Add `.zshrc.local` and `.bashrc.local` to `.gitignore`. Verify with `git check-ignore`.
- [ ] Write `.zshrc.local.example` and `.bashrc.local.example` (committed) with 2-3 commented use-cases each (host-specific PATH, machine alias, work-only env var).
- [ ] Update `.claude/CLAUDE.md` "Common Workflows" with a "Using `.local` overrides" section that contrasts with the age secrets system.
- [ ] Run drift detector — confirm exit 0 after deploying via `setup-linux.sh`.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks N/A (shell)
- [ ] Lint passes (`shellcheck` on modified .sh, none on rc files since they aren't .sh per se — confirm)
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/IDEAS-001-local-override/features.json`):

```json
[
  {
    "id": "IDEAS-001-local-override-f1",
    "behavior": ".zshrc sources $HOME/.zshrc.local when present",
    "verification": "bats tests/local-override.bats --filter 'zsh sources local'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-001-local-override-f2",
    "behavior": ".zshrc no-ops gracefully when local file absent",
    "verification": "bats tests/local-override.bats --filter 'zsh no local file'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-001-local-override-f3",
    "behavior": ".bashrc symmetric coverage",
    "verification": "bats tests/local-override.bats --filter 'bash'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-001-local-override-f4",
    "behavior": "Local files are gitignored",
    "verification": "git check-ignore .zshrc.local .bashrc.local",
    "state": "pending",
    "evidence": ""
  }
]
```
