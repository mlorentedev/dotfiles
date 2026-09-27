---
tags: [spec, tasks, templates]
created: "2026-09-25"
---

# Tasks - HARNESS-139-pi-package-reconcile

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/pi-package-reconcile` (worktree `dotfiles-wt-pi-reconcile`)
- [x] `proposal.md` is complete, and the pi behaviours it relies on are measured (Risks)

## Implementation

### The reconciler (`cli/internal/pi`)

- [x] [P] [AC2] Failing tests: the manifest loader refuses an unreadable or empty manifest; live entries are read in both the string and the object form
- [x] [AC2] Manifest and live-settings loaders
- [x] [AC1] Failing tests: the plan removes an undeclared package, installs a missing one, treats a version bump as an install, and is empty when live matches
- [x] [AC1] `Plan`: identity by package, and the apply order (remove, then install)
- [x] [AC1] [AC3] Failing tests against a fake pi that edits a real `settings.json`: apply converges, a second run makes no call, and every call's elapsed time and fenced output on failure or slowness are logged
- [x] [AC1] [AC3] `Apply`, through `pi remove` and `pi install` only
- [x] [AC4] Failing tests: `DOTFILES_SKIP_PI_PACKAGES` skips before any probe; a missing pi or npm warns and exits 0
- [x] [AC4] Environment guards and pi resolution (`--pi`, `~/.local/bin/pi`, `PATH`)

### Retired paths and the doctor check (PR-B)

- [x] [P] [AC5] Failing tests: a declared `retire` path moves under `archive/` intact, a second run moves nothing, and nothing is ever deleted
- [x] [AC5] `retire` in the manifest schema and in `Apply`
- [x] [P] [AC6] Failing tests: a package whose `requires` does not resolve FAILs; the shipped manifest PASSes
- [x] [AC6] The doctor check, on the reconciler's loader

### The command and the twins

- [x] [AC2] `dotf pi packages check`, with its exit contract tested
- [x] [AC1] `dotf pi packages apply [--dry-run]`, with its exit contract tested
- [x] [AC7] Replace both setup blocks with one `dotf pi packages apply` call; retire the shell-block tests in `tests/pi-packages.bats` (keeping the manifest-level ones) and add the twin-calls-the-command test
- [x] [AC7] Extend the CI pi path filter to `cli/internal/pi/**`
- [x] Remove `npm:pi-memory@0.4.2` from `ai/pi/packages.json` (PR-A); declare `retire: memory` (PR-B)

### Live (msi)

- [ ] [AC8] Peers told first; `check` and `apply --dry-run` recorded in the PR; the first real `apply` agreed with the owner

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-139-pi-package-reconcile/features.json`):

```json
[
  {
    "id": "HARNESS-139-pi-package-reconcile-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
