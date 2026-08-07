---
tags: [spec, tasks, templates]
created: "2026-08-06"
---

# Tasks - BUG-036-precommit-under-global-hookspath

> TDD order. One task = one focused commit. Tick as you go.
>
> `[P]` = no dependency on another unchecked task. `[AC<n>]` = satisfies acceptance criterion #n in `proposal.md`.

## Setup

- [x] Branch created from main: `fix-precommit-under-global-hookspath`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"
- [x] Work-gate verified: issue #748 is OPEN (`dotf spec init --issue 748`)

## Implementation

> Every test drives a stub `pre-commit` on `PATH`: CI installs only age/zsh/bats,
> and a stub is what makes the *invocation* assertable (real pre-commit would
> only tell us pass/fail, not what it was asked to do).

- [x] [AC1] Write failing test: no local hook + `.pre-commit-config.yaml` present -> dispatcher invokes `pre-commit hook-impl --hook-type pre-push`
- [x] [AC1] Extend `git-hooks/lib/chain-local-hook.sh` with the pre-commit fallback
- [x] [P] [AC2] Write failing test: a non-zero pre-commit exit propagates out of the dispatcher (the push is blocked)
- [x] [P] [AC3] Write failing test: stdin (the pre-push ref list) reaches pre-commit unmodified
- [x] [P] [AC4] Write failing test: an executable repo-local hook still wins and pre-commit is NOT also invoked (no double execution)
- [x] [P] [AC5] Write failing test: missing `.pre-commit-config.yaml` -> clean exit 0, pre-commit never called
- [x] [P] [AC5] Write failing test: `pre-commit` absent from PATH -> clean exit 0 (a global dispatcher must not break `git commit` everywhere)
- [x] [P] [AC6] Write failing test: the fallback is stage-generic (`pre-commit` and `commit-msg` stages, with their own arguments)
- [x] [AC7] Confirm `tests/guard-memory-sink.bats` still passes: the memory-sink guard runs first and can still veto

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Lint passes (`shellcheck -x`)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Follow-up (out of scope here, tracked separately)

- [ ] `dotf doctor`: stop emitting the `pre-commit install` remedy it cannot apply (issue #748 task 4)
- [ ] `dotf doctor`: assert the gate is *effective* — a hook actually fires — not merely that a file exists (issue #748 task 5)
- [ ] `scripts/install-precommit.sh`: same blocked `pre-commit install` call

## Machine-readable features

See sibling `features.json`.
