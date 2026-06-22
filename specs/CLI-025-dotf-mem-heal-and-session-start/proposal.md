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

> AUDIT-007 Phase B / PR9 — build the `dotf mem` noun so the claude-mem heal +
> session-start/end hook cluster collapses from twin shell scripts into one Go
> implementation. Parent: `docs/adr/audit-007-cli-convergence-state.md`.

## Why

<!-- from issue #494: CLI-025: dotf mem — heal + session-start/end, thin the hooks, delete 4+ scripts -->

`mem` is the largest unmigrated shell cluster in the CLI convergence: **8 scripts,
~1615 lines** — `claude-mem-heal.{sh,ps1}` (the claude-mem plugin self-heal),
`claude-session-start.{sh,ps1}` (the SessionStart aggregator), `session-handoff.{sh,ps1}`
(the SessionEnd twin), plus `session-brief.sh` and `ensure-memory-symlink.sh`
(sourced/overlapping singletons). They are cross-OS twins that **drift**: the
Windows `.ps1` lacks the BUG-019 context-hook neuter, so a live Windows gap exists
today. Per ADR-020/021, every twin pair must converge to one Go noun; building
`dotf mem` kills the drift, closes the Windows gap, and lets the hooks shrink to
thin shims.

## What

A new `dotf mem` noun with three subcommands, faithful ports of the existing
behaviour (cross-OS, single implementation):

- **`dotf mem heal`** — ports `claude-mem-heal`: `ensure_marketplace_compat_symlink`,
  `heal_mcp_json`, `heal_zod`, `heal_hooks_json` (preserving the `sed -n 1p` drain
  from the EPIPE fix #242, **not** `head -n1`), `neuter_context_hook` (BUG-019), and
  the v13 cascade over both the cache-version dir and the marketplace junction.
- **`dotf mem session-start`** — ports the SessionStart aggregator and folds in
  `session-brief.sh` + `ensure-memory-symlink.sh`: SDD-004 config gate, session-brief
  core (vault detect/health/specs/lessons-staleness), silent `dotf doctor --quick`
  drift surfacing, hive project detection, memory-symlink ensure, knowledge health,
  memory-temperature scan, and the SDD-021 `.claude.json` size monitor. Emits the
  SessionStart `hookSpecificOutput.additionalContext` JSON.
- **`dotf mem session-end`** — ports `session-handoff` (SessionEnd).

After this work the SessionStart/SessionEnd hooks (`claude-session-start.{sh,ps1}`
and the SessionEnd registration) are **thin shims** that `exec dotf mem …`, and the
8-script cluster is deleted. Building the heal in Go closes the BUG-019 Windows gap
for free (one implementation, both OSes).

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

~1615 LOC across 8 scripts and a per-session hot path → multiple atomic PRs, each
build-then-cutover-then-delete:

1. **`dotf mem heal` (build + test, no deletes).** Port the heal cluster to
   `cli/internal/mem`. Cross-OS table tests over hooks.json/mcp.json/zod fixtures
   incl. the EPIPE `sed -n 1p` guard and the BUG-019 neuter. Closes the Windows gap
   on build.
2. **Heal cutover + delete.** Repoint the self-heal call site + `setup-{linux,windows}`
   to `dotf mem heal`; `git rm claude-mem-heal.{sh,ps1}`; guard-grep.
3. **`dotf mem session-end` + shim + delete `session-handoff.{sh,ps1}`.** Small and
   independent; can land early.
4. **`dotf mem session-start` (build).** Port the aggregator, folding `session-brief.sh`
   + `ensure-memory-symlink.sh`. Golden-output test for the `additionalContext` JSON.
5. **Session-start cutover + delete.** Thin `claude-session-start.{sh,ps1}` to shims;
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
- **EPIPE regression.** Go does not inherit Node's `SIGPIPE→SIG_IGN`, but the
  *deployed* `hooks.json` still needs the `sed -n 1p` drain — `dotf mem heal` must keep
  emitting it (and normalising any `head -n1` left on older machines), per #242.
- **`.mcp.json` race follow-up.** #242 left the `.mcp.json` `head -n1` unfixed (lower
  blast radius). Fold the same drain into the Go `heal_mcp_json` port, or keep deferred?

## Acceptance criteria

Spec-level (each sub-PR carries its own testable ACs in `tasks.md`):

- [ ] `dotf mem {heal, session-start, session-end}` implemented in `cli/internal/mem`,
  cross-OS, table-tested.
- [ ] The SessionStart/SessionEnd hooks are thin shims (`exec dotf mem …`), no business
  logic.
- [ ] The 8-script cluster is deleted and a guard-grep pins that no production caller
  references any of them.
- [ ] BUG-019 context-hook neuter is present on Windows (the current gap is closed),
  proven by a Windows-path test.
- [ ] The SessionStart `additionalContext` output is byte-equivalent to the retired
  shell hook (golden-fixture diff).

## References

- Parent roadmap: `docs/adr/audit-007-cli-convergence-state.md` (row 9, completeness-gap rows for session-handoff/brief/symlink)
- EPIPE predecessor (resolved + archived): `specs/archive/2026-05-27-claude-mem-heal-consumer-epipe/` (PR #242) — keep the `sed -n 1p` drain + Option-A sed-patch architecture
- Heal-lineage predecessors: `specs/archive/BUG-016/017-claude-mem-heal-*`
- Related: substrate epic #469 (memory content — distinct from this noun)
