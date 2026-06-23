---
id: "CLI-025-dotf-mem-heal-and-session-start"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-22"
issue: "mlorentedev/dotfiles#494"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-025-dotf-mem-heal-and-session-start

> AUDIT-007 Phase B / PR9 — build the `dotf mem` noun so the
> session-start/end hook cluster collapses from twin shell scripts into one Go
> implementation. Parent: `docs/adr/audit-007-cli-convergence-state.md`.

> **Scope change (2026-06-22):** `dotf mem heal` is **removed from this spec**. The
> 2026-06-22 architecture session ratified ADR-016 Q2 (hive) → **claude-mem is being
> retired** (#541), so porting its self-heal is throwaway (ADR-016 "continuous
> simplification"). PR1/PR2 below are **dropped**; the WIP on `origin/feat/dotf-mem-heal`
> is abandoned. This noun is now **`session-start` + `session-end`** (folding
> `session-brief` + `ensure-memory-symlink`).

## Why

<!-- from issue #494: CLI-025: dotf mem — heal + session-start/end, thin the hooks, delete 4+ scripts -->

`mem` is a large unmigrated shell cluster in the CLI convergence:
`claude-session-start.{sh,ps1}` (the SessionStart aggregator), `session-handoff.{sh,ps1}`
(the SessionEnd twin), plus `session-brief.sh` and `ensure-memory-symlink.sh`
(sourced/overlapping singletons). They are cross-OS twins that **drift**. Per
ADR-020/021, every twin pair must converge to one Go noun; building `dotf mem`
session-start/end kills the drift and lets the hooks shrink to thin shims. (The
`claude-mem-heal.{sh,ps1}` pair, formerly the largest piece of this cluster, is
**deleted** by the claude-mem retirement #541 — not ported.)

## What

A new `dotf mem` noun with two subcommands, faithful ports of the existing
behaviour (cross-OS, single implementation):

- **`dotf mem session-start`** — ports the SessionStart aggregator and folds in
  `session-brief.sh` + `ensure-memory-symlink.sh`: SDD-004 config gate, session-brief
  core (vault detect/health/specs/lessons-staleness), silent `dotf doctor --quick`
  drift surfacing, hive project detection, memory-symlink ensure, knowledge health,
  memory-temperature scan, and the SDD-021 `.claude.json` size monitor. Emits the
  SessionStart `hookSpecificOutput.additionalContext` JSON.
- **`dotf mem session-end`** — ports `session-handoff` (SessionEnd).

After this work the SessionStart/SessionEnd hooks (`claude-session-start.{sh,ps1}`
and the SessionEnd registration) are **thin shims** that `exec dotf mem …`, and the
session-start/end script cluster is deleted. (The `claude-mem-heal.{sh,ps1}` pair is
deleted separately by #541, not here.)

## Out of scope

- The memory **content** model — vault writes, MEMORY.md authoring, junction
  topology. That is the substrate epic (#469), not this noun.
- Changing any hook **semantics**. This is a faithful port; observable output
  (the SessionStart `additionalContext`) must stay byte-equivalent.
- Upstream `thedotmack/claude-mem` behaviour. `heal` stays a resilient patcher of
  upstream's shapes (Option A from spec #242), never a self-emitted template.
- The hook **registration/deploy** mechanics in `setup-*` beyond repointing them
  at the shims (full setup→`dotf setup` is CLI-028).

## Decomposition (strangler-fig — this is NOT one atomic PR)

The session-start/end cluster + a per-session hot path → multiple atomic PRs, each
build-then-cutover-then-delete (heal PRs removed — see Scope change above):

1. **`dotf mem session-end` + shim + delete `session-handoff.{sh,ps1}`.** Small and
   independent; can land early.
2. **`dotf mem session-start` (build).** Port the aggregator, folding `session-brief.sh`
   + `ensure-memory-symlink.sh`. Golden-output test for the `additionalContext` JSON.
3. **Session-start cutover + delete.** Thin `claude-session-start.{sh,ps1}` to shims;
   `git rm` session-start + session-brief + ensure-memory-symlink; guard-grep.

## Risks / open questions

- **[OPEN] session-brief core stability.** `session-brief.sh` (ADR-023, HARNESS-026)
  is the richest piece, and HARNESS-026-session-brief-core still carries unresolved
  `[AGENT-DRAFT]` tags (flagged at this session's start). Porting an unstable surface
  invites churn. **Decide:** port it as-is now, or pin HARNESS-026 first and port in
  PR4 against a frozen contract?
- **[OPEN] PR ordering / sequencing vs the substrate epic (#469).** Does any #469
  work touch the same session-start emission, risking a collision? Confirm before
  scheduling PR4/5.
- **Per-session hot path.** The hooks run on **every** session start; a Go binary's
  cold-start must stay under the shell baseline. `dotf doctor --quick` was already
  optimised for this (CLI-013) — reuse that bar; measure.
- **Output equivalence.** The SessionStart `additionalContext` must be byte-equivalent
  to the shell version (Claude consumes it). Capture a golden fixture from the live
  shell hook before porting; diff in the test.
- **Sequencing vs #541.** The claude-mem retirement (#541) edits the same SessionStart
  hook (removing the heal call). Land #541 first, then port the slimmed aggregator —
  avoids porting a hook that #541 is about to change.

## Acceptance criteria

Spec-level (each sub-PR carries its own testable ACs in `tasks.md`):

- [ ] `dotf mem {session-start, session-end}` implemented in `cli/internal/mem`,
  cross-OS, table-tested.
- [ ] The SessionStart/SessionEnd hooks are thin shims (`exec dotf mem …`), no business
  logic.
- [ ] The session-start/end script cluster (`claude-session-start.{sh,ps1}`,
  `session-handoff.{sh,ps1}`, `session-brief.sh`, `ensure-memory-symlink.sh`) is deleted
  and a guard-grep pins that no production caller references any of them.
- [ ] The SessionStart `additionalContext` output is byte-equivalent to the retired
  shell hook (golden-fixture diff).

## References

- Parent roadmap: `docs/adr/audit-007-cli-convergence-state.md` (row 9, completeness-gap rows for session-handoff/brief/symlink)
- EPIPE predecessor (resolved + archived): `specs/archive/2026-05-27-claude-mem-heal-consumer-epipe/` (PR #242) — keep the `sed -n 1p` drain + Option-A sed-patch architecture
- Heal-lineage predecessors: `specs/archive/BUG-016/017-claude-mem-heal-*`
- Related: substrate epic #469 (memory content — distinct from this noun)
