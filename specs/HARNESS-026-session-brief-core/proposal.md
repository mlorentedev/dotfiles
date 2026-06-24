---
id: "HARNESS-026-session-brief-core"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-17"
issue: "dotfiles#405"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-026-session-brief-core

> First implementation slice of ADR-023 (agnostic session-start). Establishes the
> `session-brief` core contract and migrates the first signal under it.

> **RECONCILED WITH CLI-025 (2026-06-23).** This spec is **implemented and on main**:
> `scripts/session-brief.sh` (the agnostic core, its `sb_*` emitters, the
> `--format=stdout|markdown` contract, the 16-test bats suite, and the 3-CWD byte-equivalence
> harness) shipped, and `claude-session-start.sh` sources it. Its **mechanism — a POSIX-`sh`
> core — is interim.** CLI-025 PR2 ports it to Go (`dotf mem session-start`, preserving the
> `--format` contract) and PR3 deletes the shell core, so the agnostic session-brief core
> becomes the `dotf` binary (the "eliminate scripts via the CLI" direction). The **design here
> is the contract CLI-025 PR2 reproduces**; the byte-equivalence harness is reused as PR2's gate.
> **Archive this spec once CLI-025 PR3 deletes `session-brief.sh`** (keeping it readable until
> then so PR2 has the reference). See `specs/CLI-025-dotf-mem-heal-and-session-start/proposal.md`.

## Why

<!-- from issue #405: HARNESS-026: Agnostic session-start: session-brief core + per-agent adapters (ADR-023) -->

`scripts/claude-session-start.sh` (509 LOC) is the Claude `SessionStart` hook. It conflates two responsibilities ADR-023 decided to split: **agent-independent vault signal-gathering** (vault detection, vault-health, spec counts, baseline integrity, memory temperature, crystallize staleness) and **Claude-specific delivery** (the `hookSpecificOutput.additionalContext` JSON envelope, `claude-mem-heal`, and the `~/.claude/projects/<encoded>` path). The other harness agents (opencode, agy, copilot) get none of that brief because the gathering logic is Claude-coupled. This PR is the first strangler slice: stand up the agnostic core and migrate one signal under it without changing what Claude emits.

## What

A new `scripts/session-brief.sh` becomes the owner of the agent-independent vault signals, usable two ways: **sourced as a function library** (each signal is an `sb_*` emitter that writes its block to stdout) and **run standalone** with output modes (per ADR-023): `--format=stdout` (the full brief as raw lines, for runtime consumers) and `--format=markdown` (the same brief as one fenced block, for file-based agents the HARNESS-001 compiler will inject later). `claude-session-start.sh` becomes the **Claude adapter**: it sources the core and calls each `sb_*` emitter at the exact legacy position, so its emitted `additionalContext` stays **byte-for-byte identical**. No user-visible behavior change — a structural extraction that establishes the contract the rest of HARNESS-026 builds on.

This slice migrates the **full agent-independent cluster** (decision below): vault detection + the `Obsidian vault detected` headline, vault-health, active/archived spec counts, and vault-baseline integrity.

## Out of scope

- **Path-coupled signals** (memory-temperature, MEMORY.md staleness / crystallize, `.claude.json` size monitor) — they read Claude's `~/.claude/projects/<encoded>` path; migrating them needs the adapter to pass that path into the core. Deferred to a follow-up PR.
- **Hive project detection** (`detect_hive_project` / `find_work_sdk_project`) — agent-independent but emits MCP-tool guidance; migrated in a later slice to keep this PR within the atomic cap.
- **`claude-mem-heal`, `dotf doctor` drift, the `[sdd]` reminder, and the JSON envelope** — Claude-adapter concerns; stay in the adapter.
- **HARNESS-001 compiler delivery** to opencode/agy/copilot — this PR only *defines and tests* the `--format=markdown` contract; wiring the compiler to inject it is a separate PR (the delivery path).
- **Windows parity** (`claude-session-start.ps1` / a `session-brief.ps1` twin) — tracked follow-up; C5 honored by keeping the core POSIX `sh`.

## Risks / open questions

- **Byte-equivalence (C4) is the gate.** The four migrated signals are *interleaved* with adapter-owned blocks (the headline is prepended; `[specs]`, vault-health and vault-baseline append at different positions). Mitigation: the core is **sourceable**, so the adapter calls each `sb_*` emitter inline at its legacy position — preserving order and bytes — instead of splicing one contiguous subprocess chunk. The byte-equivalence bats test (3 CWDs vs `origin/main`) is the regression net; iterate until it passes.
- **Single vault-root derivation.** `find_vault_root` moves into the core; the adapter sources it and computes `VAULT_ROOT` once, then both the headline and the dependent signals read it — no double-derivation drift.
- **`set -euo pipefail` interaction.** Sourcing the core into the adapter shares the strict-mode shell; `sb_*` emitters must be written so an empty/absent signal returns success and emits nothing (the byte-equivalence test across 3 CWDs, including `/tmp` with no vault, covers this).
- **~RESOLVED — PR1 scope boundary:** full agent-independent cluster (baseline + specs + vault detection/health). Hive deferred to keep the diff near the ~300 LOC atomic cap.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `scripts/session-brief.sh` exists, is executable, POSIX `sh`-compatible, and ShellCheck-clean.
- [ ] `session-brief.sh --format=stdout` emits the migrated signal(s) as raw lines — the same text the old hook produced for that signal, given the same vault state.
- [ ] `session-brief.sh --format=markdown` emits the same signal(s) wrapped in a single fenced markdown block under a stable heading.
- [ ] `session-brief.sh` with an unknown/empty `--format` exits non-zero and prints a usage line to stderr.
- [ ] `claude-session-start.sh` obtains the migrated signal from the core and no longer computes it inline.
- [ ] Claude's emitted `additionalContext` is byte-identical to pre-PR output for the same inputs (existing byte-equivalence test still passes — no regression).
- [ ] New bats tests exercise both `--format` modes of the core in isolation (no Claude runtime, isolated `HOME`/vault fixtures).

## References

- Related ADR: `docs/adr/adr-023-agnostic-session-start.md` (the decision this implements)
- Sibling first brick: `scripts/ensure-memory-symlink.sh` (MEMORY-002, #404 — already agnostic, the core's first existing member)
- Epic: HARNESS-001 (#162) — cross-agent compiler that will deliver `--format=markdown` to file-based agents
- Issue: dotfiles#405 (HARNESS-026)
