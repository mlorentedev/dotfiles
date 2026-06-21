---
tags: [spec, tasks, templates]
created: "2026-06-21"
---

# Tasks - CLI-026-dotf-harness-engine

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (line-endings, cutover, sequencing resolved 2026-06-21)
- [ ] Branch created from main for PR-A: `feat/CLI-026-dotf-harness-engine`

> **Status:** spec resolved, held for roadmap order (PR10) per the sequencing decision. Implementation begins when the slot arrives; the steps below are the planned breakdown, not a frozen list.

## Implementation

> Phased cutover (resolved): PR-A adds the Go engine alongside bash; PR-B removes bash and rewires callers. TDD order, one commit each.

### PR-A — add `dotf harness` alongside bash (no deletion, no caller rewiring)

- [ ] Golden-file fixture: capture current `compile-harness.sh` output over the live vault + records as the parity oracle
- [ ] Write failing test: `harness refresh` output diffs empty vs the golden fixture (marker sha, slugify, section extraction, line caps)
- [ ] Implement `cli/internal/harness/` refresh, modeled on `cli/internal/spec/` (embedded-asset + cobra wiring)
- [ ] Write failing test: CRLF input → LF output, byte-identical on Windows and Linux
- [ ] Implement CRLF→LF normalization on read; emit LF unconditionally
- [ ] Implement `harness deploy` (offline render to per-agent `$HOME`) + `harness check` (offline drift)
- [ ] Write failing test: `check --against-vault` mutates a vault skill → non-zero exit
- [ ] Implement `check --against-vault` (sha(vault skill) != committed record)
- [ ] Refactor for clarity; wire `dotf harness` noun in `cli/internal/cmd/`

### PR-B — strangle bash (gated on PR-A parity proven)

- [ ] Rewire `setup-windows.ps1` to call `dotf harness deploy`; delete the `Deploy-SkillRecord` block
- [ ] `git rm scripts/compile-harness.sh`; repoint `setup-*.sh` / `ci.yml` / profile
- [ ] Guard-grep: no remaining references to the old block or script

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

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-026-dotf-harness-engine/features.json`):

```json
[
  {
    "id": "CLI-026-dotf-harness-engine-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
