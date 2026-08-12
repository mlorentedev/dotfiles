---
tags: [spec, tasks, templates]
created: "2026-08-12"
---

# Tasks - HARNESS-069

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/harness-record-provenance` (worktree `dotfiles-wt-harness069`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — both resolved before writing code (`do_check` read directly; duplication risk reproduced and fixed)

## Implementation

- [x] [AC1] Verify `do_check` does not compare committed records byte-for-byte against the vault (read `do_check`'s skill/agent branches directly, before writing any code) — confirmed schema+render-only, no diff
- [x] [AC1] Add `inject_record_provenance()` — mirrors `render_skill`/`render_agent`'s frontmatter-injection awk, but rewrites a file in place instead of rendering to stdout
- [x] [AC1] Call it from `do_refresh` step 3 (skills), after the verbatim `cp -rf`, with `generated_from` = vault-relative source path and `generated_sha` = `sha_of` the vault source
- [x] [AC1] Call it from `do_refresh` step 4 (agents), same shape
- [x] [AC2] Reproduce the stacking risk: run real `--refresh` then `--deploy` to a scratch `$HOME`, grep the deployed skill output for a duplicated `generated:`/`generated_from:`/`generated_sha:` — confirmed duplicated before the fix
- [x] [AC2] Fix `render_skill`'s awk to strip any pre-existing `generated`/`generated_from`/`generated_sha` line from its source before injecting its own set
- [x] [AC2] Confirm `render_agent` needs no fix (its frontmatter passthrough already allowlists only `name`/`description`) — verified by reading, then by a real deploy showing exactly one set
- [x] [AC3] Run real `--refresh` + `--check` against the actual vault — clean, no drift, confirming AC1's resolved risk in practice
- [x] [AC4] Update `tests/compile-harness.bats`'s two "no provenance" tests (skills + agents) to assert the new content instead of its absence
- [x] [AC4] Add explicit single-set assertions (`grep -c` == 1) to the deploy tests for skills and agents, pinning the no-duplication invariant as a regression test, not just a one-off manual check

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `bash -n` / `zsh -n` / `shellcheck` pass on `scripts/compile-harness.sh`
- [x] Full `tests/compile-harness.bats` suite green (44/44)
- [x] No unrelated changes in the diff — the two stale-comment touch-ups (`verbatim` copy description) are in the same functions this spec changes, not scope creep
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#927)

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-069/features.json`):

```json
[
  {
    "id": "HARNESS-069-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
