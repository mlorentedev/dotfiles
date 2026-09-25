---
tags: [spec, tasks, templates]
created: "2026-09-25"
---

# Tasks - HARNESS-145-doctrine-triggers-ledger-agy

> Retroactive: the work exists and is split into four PRs. Each PR ticks only its own slice below and fills only its own section of `verification.md`, so the slices merge in any order without touching each other's lines.
>
> **Inline markers**: `[AC<n>]` — this task helps satisfy acceptance criterion `<n>` from `proposal.md`.

## Setup

- [x] Ticket `mlorentedev/dotfiles#1682` filed and on the board
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## S1 - doctrine and ASCII (AC1)

- [x] [AC1] Failing budget test on the committed records
- [x] [AC1] Slim `pr-stewardship`, `pr-sizing` and `secrets-never-in-output` to their binding rules in the vault and refresh the records
- [x] [AC1] Fold every capped surface to pure ASCII; name a character the fold does not know by its bytes
- [x] [AC1] Migrate the old em-dash preamble without touching a user's own lines

## S2 - triggers (AC2), after S1

- [x] [AC2] Failing test: a shared pattern must not cross-link skills; an empty pattern is never reported
- [x] [AC2] Point every trigger at a pattern that exists; repair the `pattern-loader` record
- [x] [AC2] `--refresh` refuses a dangling trigger and writes nothing
- [x] [AC2] `dotf doctor` reports a trigger that names an absent pattern

## S3 - ledger (AC3)

- [x] [AC3] Failing end-to-end test: two payloads with no session id share one ledger
- [x] [AC3] No session id means no consumption or dispatch state: allow, and journal `session-unscoped` when a persona would have been enforced or a skill consumption recorded
- [x] [AC3] Reserve the `_unscoped` journal name: a session id equal to it is read as naming no session

## S4 - agy (AC4), after S3

- [x] [AC4] Parse agy's payload and answer in its protocol (`ask` or `deny`, never `allow`)
- [x] [AC4] `hooks-json` bind format under `agents.bind_named`, with the loader refusing a format in the wrong key
- [x] [AC4] Retire the stale `settings.json` entry; keep a person's disable switch
- [x] [AC4] Lessons 292 and 293, ADR-027 amendment, ADR-010 row

## Operational tail (AC5)

- [ ] [AC5] `dotf harness mirror` from a main checkout, install a binary that carries the agy parser, `dotf harness bind`
- [ ] [AC5] One real agy tool call; read `~/.local/state/dotfiles/gate/` for an `agy` record with a real conversation id

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Type checks and lint pass
- [ ] No unrelated changes in any slice
- [ ] `verification.md` filled in
- [ ] Independent review before archiving (a change that closes a spec)

## Machine-readable features

`features.json` (alongside this file) maps each acceptance criterion to one feature with a single shell command whose exit code is the verdict. Only the harness may set `"state": "passing"`.
