---
tags: [spec, tasks, templates]
created: "2026-11-12"
id: "CI-017-gate-required-check"
type: spec-tasks
status: active
owner: manu
---

# Tasks - CI-017-gate-required-check

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## PR 1: the `ci-gate` job

- [x] Branch created from main: `feat/ci017-gate-job`
- [x] `proposal.md` complete; the open questions are the owner's and do not block PR 1
- [x] [AC1] Failing test: `tests/workflow-job-names.bats` "ci.yml: ci-gate runs under always(), needs every other job, and uses the pinned gate composite"
- [x] [AC1] The `ci-gate` job in `.github/workflows/ci.yml`; the test and `tests/lib/check-workflow-contexts.py` pass
- [x] [AC2] On the PR: `ci-gate` reports and its result follows its needs (6929dd90, success)

## PR 2: protection and the Windows skip (owner, after `ci-gate` reports on main)

- [x] [AC3] `forge/branch-protection.json`: `ci-gate` replaces `changes`, `lint`, `lint-powershell`, `test`, `test-windows`
- [ ] [AC3] Owner, after merge: `dotf forge protection apply`, then `dotf forge protection check` clean
- [x] [AC4] `test-windows` takes `if: github.event_name == 'push' || needs.changes.outputs.code == 'true'`; the CI-004 checker passes

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Lint passes
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in
