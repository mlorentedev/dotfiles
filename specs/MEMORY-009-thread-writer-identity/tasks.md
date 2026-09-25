---
tags: [spec, tasks, templates]
created: "2026-09-25"
---

# Tasks - MEMORY-009-thread-writer-identity

> TDD order, two slices. **Inline markers**: `[AC<n>]` — this task helps satisfy acceptance criterion `<n>` from `proposal.md`.

## Setup

- [x] Ticket `mlorentedev/dotfiles#1690` filed and on the board
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] MEMORY-008's slice 0 (#1606, #1620, #1651) merged: this change builds on `WriteThread` as #1703 left it

## Slice 1 - the stamp and the fork (one PR)

- [x] [AC4] Pin today's output first: a write with no agent is byte-identical to `WriteThread`'s
- [x] [AC3] Failing test, then implementation: a write by an agent stamps `(writer: <agent>)`, and a rewrite by the same agent replaces its own block in place
- [x] [AC1] Failing test, then implementation: another agent's stamped block is kept, and the write lands in `<key>+<agent>`
- [x] [AC1] Failing test, then implementation: an unstamped block whose `Journal:` line names another agent is kept the same way
- [x] [AC3] Failing test, then implementation: an unstamped block with no known writer is replaced in place and stamped
- [x] [AC2] Failing test, then implementation: `handoff-write --agent` names the key and both agents on stderr when it forks

## Slice 2 - the skill passes the flag (after a release carrying `--agent` is installed)

- [x] [AC5] Vault `handoff` skill: its command passes `--agent`; the record is refreshed

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test (AC5 by f5's check of the record)
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks and lint pass
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [ ] Independent review before archiving (a change that closes a spec)

## Machine-readable features

`features.json` (alongside this file) maps each acceptance criterion to one feature with a single shell command whose exit code is the verdict. Only the harness may set `"state": "passing"`.
