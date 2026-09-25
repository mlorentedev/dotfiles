---
id: "MEMORY-008-handoff-resume"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-25"
issue: "mlorentedev/dotfiles#1689"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# MEMORY-008-handoff-resume

## Why

<!-- from issue #1689: MEMORY-008: the handoff block and the next-session prompt are two hand-written copies of one fact, so the prompt carries what the block lacks -->

Every session ends with the same manual ritual: /handoff, ask for a paste-ready prompt, /compact or /clear, paste. The handoff block and that prompt are two hand-written copies of one fact, and only the block is durable. It is also the poorer copy: nothing enforces its fields, no code renders a prompt from it, and `dotf mem session-start` ignores the hook's `source`, so after /compact or /clear the session gets a brief with no handoff in it.

## What

- A thread has one canonical field set, defined once in Go: `Updated`, `Last task`, `Decisions`, `Open threads`, `Next action`, `Verify`, `Awaiting Manu` and `Journal`. The labels sessions already improvise (`Verify at start`, `Judgment calls left open`) are read as aliases, so no existing block needs migrating by hand.
- The per-thread file `memory/threads/<key>.md` is the source. The entry in `MEMORY.md` is derived from it and capped at about 350 B (updated, writer, one-line next action, how many items await Manu, pointer), not the 1.5 KB block of today. Old blocks stay readable and move to the new shape on their next write.
- `dotf mem resume [--thread K] [--format prompt|context]` renders the thread. The output depends only on the thread file. When the working directory names no single thread for this agent, it lists the candidates rather than picking one.
- Claude's `SessionStart` hook injects the resume keyed on `source`: the full resume for `startup`, `clear` and `compact`, one line for `resume` and `fork`, where the conversation already holds the history.
- Harnesses with no session hook get one sentence in the compact doctrine: run `dotf mem resume` and follow it.
- `/catchup` step 1 runs `dotf mem resume` and its `Verify` commands; "awaiting Manu's decision" is a valid next action.

## Out of scope

- The three bugs that lose or misplace data in the source `resume` reads, #1606, #1620 and #1651, stay on their own issues and land first (slice 0 in the research note).
- Writer identity on default-branch keys: MEMORY-009 (#1690).
- Archiving threads, on `dotf worktree done` and by age (14 days) for ambient `main@host` threads. The decision is recorded on #1689; the implementation follows once the per-thread store exists.
- Checking a thread's references against GitHub at render time (`Refs:` status, REST, 5 s, fail-open): a later slice once `resume` exists.
- Session-start injection for pi, opencode and Copilot: they get the doctrine sentence until their hooks can emit context.
- The catchup gaps outside step 1 (#1522, #1229, #1654).

## Risks / open questions

- **Claude's JSON envelope is production evidence, not documented contract.** `hookSpecificOutput.additionalContext` is how this session's own context arrived, but the hooks docs show plain stdout. Mitigation: a golden test pins the envelope `ClaudeEnvelope` writes; if it stops working, plain stdout is the documented fallback.
- **A `Verify:` line is an injection surface.** A `MEMORY` file is input the next session did not write. `resume` marks any command outside a read-only allowlist as unverified, and the session still runs it through its normal permission gate.
- **The doctrine sentence must fit.** The compact doctrine is 7,798 characters on main against the 8,000 budget test (HARNESS-145), so there is room for one short sentence (about 40 characters) and no more. The AC1 budget test is part of this spec's verification.
- **Non-hook harnesses see less ambiently.** Moving the rich fields out of the auto-loaded file changes what they read at start. The derived index entry stays in `MEMORY.md`, and the doctrine sentence tells them where the rest is.
- **Whether the lint refuses or warns.** `handoff-write` starts by warning on a missing `Next action` or an unknown label, so no session loses a handoff to the lint. Refusing is decided after the warnings have been counted.
- **Design options taken from the research note.** Per-thread file as the source (rejected: inline fields with a byte cap, which put dotfiles' `MEMORY.md` at 103% of its budget) and the archive lifecycle. Either can be overruled on this PR.

## Acceptance criteria

- [ ] **AC1** — `resume` output is a pure function of the thread file (golden test); two runs are identical.
- [ ] **AC2** — a thread with `Verify:` and `Awaiting Manu:` renders both, and an ambiguous working directory lists the candidate threads instead of choosing one.
- [ ] **AC3** — `SessionStart` output begins with the resume block for `source=compact` and `source=clear` (golden fixture per source), and `resume` and `fork` get one line.
- [ ] **AC4** — no `MEMORY.md` in a fixture copy of the real dotfiles file exceeds 25,000 B after the new fields are added.
- [ ] **AC5** — the handoff skill points the next session at `dotf mem resume` instead of a hand-typed prompt, and catchup step 1 is `dotf mem resume`.
- [ ] **AC6** — the fallback sentence is in the compiled doctrine, `compile-harness.sh --check` passes, and the doctrine stays under the 8,000-character budget.

## References

- Bitácora: `mlorentedev/dotfiles#1689` (see the `issue:` frontmatter field); MEMORY-009 is #1690.
- Research: vault `10_projects/dotfiles/research/2026-09-24-handoff-resume-ritual.md` (method and measurements in sections 1-2, design in 3, slicing in 4).
- Extends spec `HARNESS-088-handoff-threads` (#1278), whose thread model this keeps. Budget and pruning: #1477.
- Code on main: `cli/internal/mem/handoff.go` (`WriteThread`, `ThreadKey`, `isThreadHeading`), `session_start.go`, `session_start_adapter.go` (`ClaudeContext`, `ClaudeEnvelope`), `cli/internal/cmd/mem.go` (`runClaudeHook`).
- Patterns: `pattern-context-orchestration` (a cheap always-on selector gates an on-demand payload), `pattern-ai-memory` section 4 (offload).
