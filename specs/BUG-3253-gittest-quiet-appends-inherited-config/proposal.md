---
id: "BUG-3253-gittest-quiet-appends-inherited-config"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-07"
issue: "mlorentedev/dotfiles#3253"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# BUG-3253-gittest-quiet-appends-inherited-config

## Why

`cli/internal/cmd`'s `TestMain` calls `gittest.Quiet()`, and `Quiet` panicked on
any `GIT_CONFIG_COUNT` it had not written itself. Every agent harness that pins
git config through that variable — Copilot CLI exports three pairs, to force
non-interactive credentials and `safe.bareRepository=explicit` — therefore
aborted the whole `internal/cmd` test binary at startup. The tests were fine and
CI was green, because CI has a clean environment; the failure existed only where
agents are told to run the suite, and nothing told the operator to scrub the
environment first, so the suite read as broken and the evidence for every
`internal/cmd` change had to come from a workaround nobody wrote down.

## What

`Quiet` merges instead of refusing. It reads the count the environment already
carries, keeps every inherited `GIT_CONFIG_KEY_i`/`GIT_CONFIG_VALUE_i` pair, and
appends `gc.auto=0` and `maintenance.auto=false` at the next free indices, so
git still reads the caller's pairs and still applies `Quiet`'s (the later index
wins). A test binary that re-executes itself inherits its parent's pairs, so a
second `Quiet` finds its own value already last for each key and adds nothing. A
`GIT_CONFIG_COUNT` that is not a non-negative integer, or that promises a pair
the environment does not carry, is still refused, because git would read the
same number and fail somewhere the operator cannot see.

## Out of scope

- Rewriting the harness variables, or unsetting them for a clean slate: the
  inherited pairs are exactly what the caller needs.
- Changing the `GIT_CONFIG_SYSTEM` file `Quiet` writes, which is what reaches
  the remote side of a local transport (#3031).
- Making `Quiet` tolerant of `GIT_CONFIG_NOSYSTEM`, which stays a panic: it
  would silently leave remotes unquieted.

## Risks / open questions

- **Index collisions.** `Quiet` never writes at an index the caller uses; it
  writes at `len(caller) + added`. A count that promises a pair the environment
  does not carry is refused, the way git refuses it, rather than corrected on
  the caller's behalf: writing over an index git cannot read would hide the
  caller's own bug behind ours.
- **Duplicate keys.** git keeps the last pair for a key, so `Quiet` appends
  unless the caller's *last* word on that key is already the value it wants. The
  distinction matters: a caller that pins `gc.auto=0` at one index and
  `gc.auto=1` at the next ends on `1`, so "the pair is somewhere in the list" is
  not the same test as "the pair wins". Both shapes are asserted, not assumed.
- **Long-lived divergence.** `alreadyQuiet()` disappears, so the fast path is
  now "the caller's last pair for the key already carries our value", which the
  re-execution test covers.

## Acceptance criteria

- [x] `go test ./internal/cmd -count=1` starts and exits cleanly with
      `GIT_CONFIG_COUNT=3` and three inherited pairs in the environment, where
      it previously aborted in `TestMain` before running a single test. The
      criterion is startup, not the whole suite: on a host whose global
      `core.hooksPath` makes every fixture commit run a full hook chain, the
      suite exceeds Go's per-binary timeout, so CI is the authority for it.
- [x] The inherited pairs are still readable by a child `git config`, and
      `Quiet`'s own pairs apply on top of them.
- [x] A second `Quiet` call adds nothing, so a re-executed test binary does not
      grow the list on every generation.
- [x] The malformed-count refusal survives, and a pre-populated environment is
      covered by a regression test rather than by the removed panic path.

## References

- Bitácora board: mlorentedev/dotfiles#3253
- Related lesson: `docs/lessons/lesson-517-git-background-work-is-stopped-in-the-environment-not-the-fixture-repo.md`
- Related issue: #3031 (the remote side of a local transport strips `GIT_CONFIG_COUNT`)
