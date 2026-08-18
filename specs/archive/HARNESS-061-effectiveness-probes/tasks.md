---
tags: [spec, tasks, templates]
created: "2026-08-08"
---

# Tasks - HARNESS-061-effectiveness-probes

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [n/a] Branch `feat/HARNESS-061-effectiveness-probes` — never existed. The work
  landed across several PRs rather than one feature branch; recorded as `n/a`
  instead of ticked, because a ticked box asserting a branch nobody created is
  the representation-over-behaviour failure this spec is about.
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — both
  were resolved in the body (execution-vs-resolution probe split; dispatcher
  fallback covered by its own real-dependency test) and the third (exemption
  table becoming permanent) is mitigated by the staleness test, re-verified by
  mutation 2026-08-18

## Implementation

- [x] `[AC1]` `hookprobe.go`: `effectiveHooksPath` — read the value git resolves for a repo, local beating global
- [x] `[AC1]` `hooksDirFor` — resolve via `--git-common-dir`, so linked worktrees and `--separate-git-dir` behave (BUG-043's lesson)
- [x] `[AC3]` `hookForStage` — executable only; git ignores a hook it cannot run
- [x] `[AC2]` `stageReachesPreCommit` — the generated hook OR a dispatcher with a config to act on
- [x] `[AC1]` `checkGuardHooks`: probe each repo that matters by effect, naming the repo and the remedy
- [x] `[AC2]` `checkVaultHooks`: resolve what git would run instead of testing for `.git/hooks/<stage>`
- [x] `[AC4]` `[P]` `tests/stub-real-pairing.bats`: pairing guard + exemption table
- [x] `[AC5]` `[P]` staleness test for the exemption table
- [x] `[AC6]` `[P]` `checks_dr.go`: escrow freshness + drill marker, WARN not FAIL
- [x] `[AC7]` red-direction tests for every new check
- [x] `[AC5]` doc-table drift guard: the human-readable exemption table is a third
      copy of `EXEMPT_SUITES` and nothing compared them (found 2026-08-18)

## Verification

- [x] `go test ./...` — 12 packages green
- [x] `bats tests/*.bats` — only the pre-existing #807 failure remains
- [x] `shellcheck` clean
- [x] Red-teamed: removing each fix turns its guard red
