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
second `Quiet` finds them present and adds nothing. A `GIT_CONFIG_COUNT` that is
not a non-negative integer is still refused, because git would read the same
number and fail somewhere the operator cannot see.

## Out of scope

- Rewriting the harness variables, or unsetting them for a clean slate: the
  inherited pairs are exactly what the caller needs.
- Changing the `GIT_CONFIG_SYSTEM` file `Quiet` writes, which is what reaches
  the remote side of a local transport (#3031).
- Making `Quiet` tolerant of `GIT_CONFIG_NOSYSTEM`, which stays a panic: it
  would silently leave remotes unquieted.

## Risks / open questions

- **Index collisions.** `Quiet` never writes at an index the caller uses; it
  writes at `len(caller) + added`. A caller whose count disagrees with the pairs
  actually set is preserved verbatim rather than corrected — git reports that
  case itself, as it does without this package.
- **Duplicate keys.** If the caller already pins `gc.auto`, both pairs are
  present and the later one wins. That is the intended precedence, and it is
  asserted rather than assumed.
- **Long-lived divergence.** `alreadyQuiet()` disappears, so the fast path is
  now "the pair is already in the list", which the re-execution test covers.

## Acceptance criteria

- [x] `go test ./internal/cmd -count=1` starts and runs its tests with
      `GIT_CONFIG_COUNT=3` and three inherited pairs in the environment, where
      it previously aborted in `TestMain`.
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
