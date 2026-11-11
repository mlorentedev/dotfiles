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
  references. The event's `repository.full_name` is deliberately not a third source: on a runner
  `GITHUB_REPOSITORY` is always set and names the same repository for a `pull_request` event
- Out of scope: `spec-gate-pr.bats` and `spec-gate-prepush.bats` test the `gh` wrappers that resolve the PR
  live; the shared composite replaces them with the event file.

## PR 3: `--adjacency`, `--summary` for `gate spec`

- [x] [AC2] Adjacency report through the shim: `tests/spec-gate-adjacency.bats` 15 through
  `spec_gate_both`, `--adjacency-issues` mapped to `--adjacency`, the step summary compared byte for byte
  (dotf's `--summary` file, appended once per shell). Mutations caught: the `%` escape, the `|` escape,
  the 88-byte clip, the closed-issue exclusion
- [x] A feed whose last line has no newline: the script's `read` drops it, `dotf` reads it. Kept as a fix,
  since the workflow writes every line terminated and no parity case can tell them apart
- [x] `--summary` on both gates: `gate knowledge` has had it since PR 1 (the script writes none, so it has no
  parity case); `gate spec` writes the adjacency table. Every `--summary` writes through one helper

## PR 4: `dotf lint fleet`

- [x] [AC4] Reader over `.github/workflows` and `.gitea/workflows` (every YAML document of a file), first four
  rules (`fleet/timeout`, `fleet/sha-pin`, `fleet/persist-credentials`, `fleet/no-secrets-inherit`), each with a
  good and a bad fixture; the bad one trips its rule once and no other
- [x] [AC4] Exceptions in `.github/lint-exceptions.yml`: rule, path, reason and issue required; an incomplete entry
  is `fleet/exception` and silences nothing; an exception silences its rule on its path only
- [x] Decided: a job calling a reusable workflow is exempt from `fleet/timeout` (GitHub rejects the key there);
  `./` and `docker://` are exempt from `fleet/sha-pin` (a digest pin is a registry rule, not a git one)
- [x] Mutations caught: the caller exemption, the `./` exemption, an explicit `persist-credentials: true`, an
  exception that ignores its path, one that ignores its rule, the `inherit` match
- [x] dotfiles' own `.github/lint-exceptions.yml` lands with the adoption, by the owner (#3336): `.github/**` is
  not an agent's to merge. Run on dotfiles at PR 4: 21 findings, all `fleet/persist-credentials`

## PR 5a: the remaining four rules

- [x] [AC4] `fleet/cancel-in-progress` (workflow-level `concurrency` only; an expression counts as set, `false` does
  not; only where a push to the pull request starts the workflow again, so `types:` without `synchronize` is exempt), `fleet/os-on-pull-request` (no `if:`, and `runs-on` names Windows or macOS literally or through a literal
  matrix, its `include:`, a nested `matrix.a.b` path, less an `exclude:` on that axis alone; a computed matrix is conditional), `fleet/skip-cache` (an explicit switch-off only:
  `cache: false` on a `*/setup-*` step, `skip-cache: true` on golangci-lint-action), `fleet/branch-filter` (push and
  pull_request `branches:` read as GitHub does, in order, last match wins, `**` crossing a slash, `?` and `+`
  quantifying the preceding character, `[...]` a class, `\` escaping; `branches-ignore`
  not read)
- [x] Default branch: `--default-branch`, else what origin/HEAD names. A branch filter with neither is exit 2,
  never a pass
- [x] Pull-request triggers read in all three shapes of `on:` (scalar, sequence, mapping)
- [x] Mutations caught: `cancel-in-progress: false`, the sequence form of `on:`, the `if:` exemption, `include:`,
  the setup-action scope of `cache:`, the `skip-cache` value, `!` patterns, `**`, regexp metacharacters, the missing
  default branch, the pull-request scope of the cancel rule
- [x] Run on dotfiles: 29 findings (26 once #3305 gated two Windows legs and added a checkout). 21 persist-credentials (#3335), 3 os-on-pull-request (`ci.yml` test-windows,
  `cli.yml` test, `cli-smoke.yml` smoke), 1 cancel-in-progress (`repo-hygiene.yml`), 2 skip-cache (`cli.yml`, the
  exception the proposal names with #2834). `add-to-project.yml` runs on `opened` and `reopened` only: it was a
  finding until the rule read `types:`

## PR 5b: `--fleet`, exception hygiene, runbook

- [x] [AC5] `--fleet <dir>` over the git checkouts directly under a directory, each against its own origin/HEAD,
  findings prefixed with the checkout's name; one that cannot be read is reported and the rest still checked
  (exit 2); no checkout at all is exit 2; `--dir` and `--default-branch` refused with it
- [x] Stale exceptions (matching nothing) and unknown rule ids as findings; the fixture test walks the list of
  rules an exception may name, and a test ties that list to the fixture directories
- [x] [AC5] The runbook: `docs/runbooks/guide-fleet-lint.md`
- [ ] [AC5] The link from ci-workflows ADR-001, a pull request in that repository once the runbook is on main
- [x] Run over the owner's local checkouts: 222 findings in 13 repositories, no unreadable one

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Lint passes
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in
