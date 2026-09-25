---
tags: [spec, tasks, templates]
created: "2026-09-25"
---

# Tasks - MEMORY-008-handoff-resume

> TDD order. Each slice below is one PR under the 300-line executable cap, on its own branch from main, and opens after the slice it depends on has merged. One task = one focused commit.
>
> **Inline markers**: `[AC<n>]` — this task helps satisfy acceptance criterion `<n>` from `proposal.md`. `[P]` — no dependency on another unchecked task.

## Setup

- [x] Ticket `mlorentedev/dotfiles#1689` filed and on the board
- [x] `proposal.md` reviewed on the spec PR (the design options it takes from the research note are confirmed or overruled)
- [x] #1606, #1620 and #1651 merged (slice 0: they lose or misplace data in the source `resume` reads)

## Slice 1 - thread schema (depends on slice 0)

- [x] [AC2] Failing test: `ParseThread` reads every canonical label, maps `Verify at start` and `Judgment calls left open` to `Verify` and `Awaiting Manu`, and keeps an unknown label rather than dropping it
- [x] [AC2] Implement `ParseThread` and the canonical label set in `cli/internal/mem`
- [x] Failing test, then implementation: `handoff-write` names a missing `Next action` or an unknown label on stderr and still writes

## Slice 2 - per-thread store and derived index entry (depends on slice 1)

- [ ] [AC4] Failing test: a fixture copy of the real dotfiles `MEMORY.md`, with `Verify` and `Awaiting Manu` added to every thread, stays at or under 25,000 B
- [ ] [AC4] `handoff-write` writes `memory/threads/<key>.md` atomically (temp file and rename) and derives the capped index entry in `MEMORY.md`
- [ ] [AC4] Failing test (`TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget`), then implementation: a write that would push `MEMORY.md` past its byte budget fails loudly instead of truncating (#1477)
- [ ] Failing test, then implementation: an old-format block is read as a thread and rewritten in the new shape on its next write, with every other thread byte-identical

## Slice 3 - `dotf mem resume` (depends on slice 2)

- [ ] [AC1] Failing golden test: `resume --format prompt` and `--format context` on a fixture thread; two runs are byte-identical
- [ ] [AC1] Implement `dotf mem resume` as a pure render of the thread file
- [ ] [AC2] Failing test, then implementation: `Verify` and `Awaiting Manu` both render; a `Verify` command outside the read-only allowlist is marked unverified
- [ ] [AC2] Failing test, then implementation: a working directory that names no single thread for this agent lists the candidates and picks none

## Slice 4 - SessionStart by source (depends on slice 3)

- [ ] [AC3] Failing golden tests, one fixture per `source`: `startup`, `clear` and `compact` begin with the full resume; `resume` and `fork` get one line
- [ ] [AC3] `runClaudeHook` parses `source` and `session_id` (only `cwd` today) and passes them to the session-start context
- [ ] [AC3] A flag in `session-start-config.json` turns the injection off, with a test

## Slice 5 - skills and doctrine (depends on slice 3)

- [ ] [AC5] Vault `handoff` skill: the chat summary points the next session at `dotf mem resume`, with no hand-typed prompt; records refreshed
- [ ] [AC5] Vault `catchup` skill: step 1 runs `dotf mem resume` and its `Verify` commands, and an ambiguous thread is listed and asked about; records refreshed
- [ ] [AC6] Fallback sentence in the compact doctrine's vault source; `--refresh`, `--check`, and the 8,000-character budget test pass

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Type checks and lint pass
- [ ] No unrelated changes in any slice
- [ ] `verification.md` filled in
- [ ] Independent review before archiving (a change that closes a spec)

## Machine-readable features

`features.json` (alongside this file) maps each acceptance criterion to one feature with a single shell command whose exit code is the verdict. Only the harness may set `"state": "passing"`. The test names are the ones this plan intends; a slice that names a test differently updates its feature in the same PR.
