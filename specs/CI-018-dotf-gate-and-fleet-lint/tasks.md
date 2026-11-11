---
tags: [spec, tasks, templates]
created: "2026-11-11"
id: "CI-018-dotf-gate-and-fleet-lint"
type: spec-tasks
status: active
owner: manu
---

# Tasks - CI-018-dotf-gate-and-fleet-lint

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/dotf-gate-commands`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## PR 1: `internal/gate` and `dotf gate knowledge`

- [x] [AC1] Parity harness: `tests/check-knowledge-gate.bats` runs every judged case through the shell gate and `dotf gate knowledge` and requires the same status and output
- [x] [AC1] `gate.ReadEvent`: body, labels, author; a null body is empty; no `pull_request` is `ErrNoPR`
- [x] [AC1] `gate.StripMarkup`: one left-to-right pass over fences, HTML comments and code spans, reporting an unclosed fence
- [x] [AC1] `gate.JudgeKnowledge` and `dotf gate knowledge`, with `--summary` and `--explain`
- [x] [AC3] The shim runs dotf with no token and a PATH holding git alone

## PR 2a: the spec gate's parsers

The script does not fit one PR under the 300-line cap, so PR 2 splits along the pure/impure line.

- [x] [AC2] `StripCode`: the spec gate's own stripper (closes a fence on any long-enough marker, removes
  inline spans), kept apart from `StripMarkup` until both scripts retire (#2407)
- [x] [AC2] `ClosingIssueNumbers`: closing verbs, a qualified reference only for this repository, URL folding,
  line by line
- [x] [AC2] `FrontmatterIssue`, `ExcludedFromLOC`, `IsActiveSpecPath`, `NormalizeRenamePath`, `ParseNumstat`
- [x] [AC2] Table tests; a one-off differential run against the shell functions agreed on every case and went
  red on two planted mutations

## PR 2b: `dotf gate spec` core

- [x] [AC2] Parity harness `tests/lib/spec-gate-parity.bash` over `check-spec-gate.bats` and
  `spec-gate-archive.bats`, PATH holding git alone; the one paragraph that tells a script user how to re-run
  by hand is compared on neither side
- [x] [AC2] Archive-on-merge, dependency and `skip-sdd` escapes, the LOC walk and `--explain`
- [x] [AC2] Repository slug: `GITHUB_REPOSITORY`, then origin, as the script; empty ignores qualified
  references. The event's `repository.full_name` is not read yet: the script does not, and the parity
  suites have no way to tell the two apart
- Out of scope: `spec-gate-pr.bats` and `spec-gate-prepush.bats` test the `gh` wrappers that resolve the PR
  live; the shared composite replaces them with the event file.

## PR 3: `--adjacency`, `--summary` for `gate spec`

- [ ] [AC2] Adjacency rule through the shim

## PR 4: `dotf lint fleet`

- [ ] [AC4] Reader, first four rules, `.github/lint-exceptions.yml` with rule, path, reason and issue

## PR 5: remaining rules, `--fleet`

- [ ] [AC4] Remaining rules
- [ ] [AC5] `--fleet` report and the runbook section

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Lint passes
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in
