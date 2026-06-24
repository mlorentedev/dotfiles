---
tags: [spec, tasks, templates]
created: "2026-06-22"
---

# Tasks - CLI-025-dotf-mem-heal-and-session-start

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [ ] Branch created from main: `feat/CLI-025-dotf-mem-heal-and-session-start`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> Strangler-fig: 3 PRs (proposal "Decomposition"). **PR1 = `session-end`** (this branch,
> `feat/CLI-025-dotf-mem-session-end`). PR2/PR3 (`session-start`) are blocked on pinning
> HARNESS-026 — see the RESOLVED open questions in `proposal.md`.

### PR1 — `dotf mem session-end` (port `session-handoff.{sh,ps1}`)

- [ ] Failing test: empty stdin / non-JSON / missing `cwd` → clean no-op, no file, exit 0
- [ ] Failing test: project MEMORY.md absent → no-op
- [ ] Failing test: `## Session Handoff` block absent or whitespace-only → no-op (trivial session)
- [ ] Failing test: happy path → writes `<vault>/10_projects/<project>/sessions/<date>-<project>-claude.md`
      with the SessionEnd frontmatter + extracted block
- [ ] Implement `cli/internal/mem/session_end.go` — resolve vault via `vault.ResolveVault()`
      (retires the hardcoded `$HOME/Projects/knowledge` literal, #463), UTC date stamp
- [ ] Wire `cli/internal/cmd/mem.go` (`newMemCmd` + `session-end` subcommand reading stdin),
      register in `root.go`; `go build ./...` + `go test ./...` green
- [ ] Cutover: repoint SessionEnd registration in `setup-{linux.sh,windows.ps1}` at
      `dotf mem session-end`; `git rm scripts/session-handoff.{sh,ps1}`
- [ ] Guard test (bats): no production caller references `session-handoff`

### PR2/PR3 — `dotf mem session-start` (DEFERRED — blocked on HARNESS-026)

- [ ] (PR2) Capture golden `additionalContext` fixture from the live shell hook; port the
      aggregator folding `session-brief.sh` + `ensure-memory-symlink.sh`; byte-equivalence diff
- [ ] (PR3) Thin `claude-session-start.{sh,ps1}` to shims; `git rm` the cluster; guard-grep

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

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-025-dotf-mem-heal-and-session-start/features.json`):

```json
[
  {
    "id": "CLI-025-dotf-mem-heal-and-session-start-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
