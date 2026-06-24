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

### `dotf mem session-start` — decomposed PR2a / PR2b / PR3 (architecture decision 2026-06-23)

> **Architecture (ratified in-session 2026-06-23):** the session-start hook splits into an
> **agnostic core** + a **thin Claude adapter**, with OS-variance localized in a shared Go
> primitive. This maximizes the north star (everything in `dotf`, max OS/agent agnosticism):
>
> - **Agnostic core** — the `sb_*` brief signals (vault detect/health/specs/lessons/baseline),
>   rendered via `--format=stdout|markdown`. Serves opencode/agy/copilot (HARNESS-001 compiler)
>   AND feeds the Claude adapter. This is the port of `session-brief.sh`.
> - **Claude adapter (thin)** — wraps the core in the `additionalContext` JSON envelope and runs
>   the Claude-only injectors (SDD-004 config gate, doctor-drift, hive/work-sdk detect, knowledge
>   health, memory-temperature, `.claude.json` monitor) + the junction ensure.
> - **`memlink` primitive** — junction(Windows)/symlink(POSIX) ensure, one OS-agnostic Go helper,
>   shared by the adapter (per-session ensure, surfaces a notice if it repaired — kills #551's
>   "silent hook" complaint) AND by `dotf doctor --fix` (#551, on-demand repair). One impl, two
>   consumers — no Go-level duplication.
>
> The ~720 LOC port exceeds the ~300 LOC atomic cap, so it splits along the architecture seam:

#### PR2a — agnostic core (this branch, `feat/CLI-025-mem-session-start-core`)

- [ ] Failing tests (table-driven, mirror `tests/session-brief.bats`): `vaultDetect`, `specs`
      (active/archived counts + AGENT-DRAFT flagging), `lessonsStaleness`, `vaultBaseline`,
      `vaultHealth` (against a stub script), brief assembly + leading-blank drop, `--format` render
- [ ] Implement `cli/internal/mem/session_start.go` — port `session-brief.sh`'s `find_vault_root`
      + `sb_*` emitters + the assemble/render runner; `vaultHealth` shells out to the same
      `vault-health.sh` (scriptsDir injected for tests, resolved from the env-contract in prod)
- [ ] Wire `dotf mem session-start --format=stdout|markdown` in `cli/internal/cmd/mem.go`
- [ ] **Gate:** Go byte-equivalence harness vs `session-brief.sh --format=…` across the 3 CWDs
      (dotfiles repo / outside-vault / inside-vault), reusing the `session-start-config.bats` pattern
- [ ] `go build ./...` + `go test ./...` green

#### PR2b-1 — `memlink` primitive (PR #557)

- [x] Extract `ensure-memory-symlink.sh` into an OS-agnostic Go `memlink` primitive; the Windows
      junction branch also closes the MEMORY-002 R4 gap the shell twin deferred. Standalone package
      `cli/internal/memlink` (resolution + symlink/junction create), consumed next by the adapter
      and by `dotf doctor --fix` (#551).

#### PR2b-2a — Claude adapter injectors (this PR)

- [x] Port the Claude-only injectors as pure functions, each returning its exact CONTEXT_LINES
      contribution: SDD-004 config reader, `claude.json-size`, `knowledge-health`,
      `memory-temperature`, `doctor-drift`, `hive`/`work-SDK` detect, `auto-memory` link
      (delegates to `memlink.Ensure`). Table-tested byte-exact.

#### PR2b-2b — Claude adapter assembly + gate

- [x] Assemble the injectors + the `mem.Brief` (PR2a) `sb_*` blocks in the exact CONTEXT_LINES
      order, wrap in the `additionalContext` JSON envelope (jq-equivalent: ordered keys, no HTML
      escaping), wire the default `dotf mem session-start` (no `--format`) = the Claude hook path
- [x] Golden-fixture byte-equivalence diff vs the live `claude-session-start.sh` across 3 CWDs
      (POSIX-only gate; runs on the Linux test job)

#### PR3 — cutover + delete

- [ ] Repoint the SessionStart hook to `dotf mem session-start` directly (no shim);
      `git rm` `claude-session-start.{sh,ps1}` + `session-brief.sh` + `ensure-memory-symlink.sh`;
      guard-grep that no production caller references them; then HARNESS-026 (#405) can be archived
- [ ] Wire `dotf doctor --fix` to the shared `memlink` primitive (closes the #551 junction half)

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
