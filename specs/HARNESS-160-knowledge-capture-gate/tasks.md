---
tags: [spec, tasks, templates]
created: "2026-09-25"
---

# Tasks - HARNESS-160-knowledge-capture-gate

> TDD order. One task = one focused commit. `[P]` = no dependency on another unchecked task; `[AC<n>]` = serves acceptance criterion n.

## Setup

- [x] Worktree `../dotfiles-wt-knowledge-gate`, branch `feat/knowledge-capture-gate` from main
- [x] #387 re-scoped, In Progress, P1
- [x] `proposal.md` complete; no open questions left

## Slice 1: the checker (PR 1)

- [x] [AC1] Failing cases in `tests/check-knowledge-gate.bats`, on a real git fixture repo:
  - a valid body passes;
  - each failure fails, naming its cause: the section missing, a line missing, a bare `none`, a path not in the diff, a path outside its kind's directory, `_index.md`, a duplicated line;
  - a section inside a fenced code block does not count;
  - an unset body, or refs that cannot be diffed, exit 2.
- [x] [AC1] `scripts/check-knowledge-gate.sh` makes them pass.
- [x] [AC2] Failing cases:
  - a dependency bot with the `dependencies` label passes;
  - a human with the label is judged;
  - the bot list equals `spec-gate`'s.
- [x] [AC2] Implement the skip.
- [x] `shellcheck` clean; features f1 and f2 recorded.

## Slice 2: wiring (PR 2, which runs the gate on itself)

- [x] [AC3] Failing cases in `tests/spec-gate-pr.bats`:
  - `--gate check-knowledge-gate.sh` execs that script;
  - a `--gate` value with a `/` exits 2;
  - no `--gate` still runs `check-spec-gate.sh`.
- [x] [AC3] Add `--gate` to `scripts/spec-gate-pr.sh`.
- [x] [AC3] Add `.github/workflows/knowledge-gate.yml`. A test asserts its triggers and its `cancel-in-progress` expression equal `spec-gate.yml`'s.
- [x] [AC4] Add the release-please `pull-request-footer`. A case runs the checker on the footer text.
- [x] [AC6] Add the section to the PR template, and to DoD §2: the vault's `pattern-change-lifecycle.md`, then the compiled render.
- [x] [AC7] Write ADR-039 (knowledge is asked for by the PR that produces it) and lesson-301 (knowledge goes where a mechanism asks for it). This PR's own section names them.
- [ ] Before the merge: tell live peers, and add the section to the bodies of open PRs.

## Slice 3: promotions at archive (PR 3)

- [ ] [AC5] Failing Go tests in `cli/internal/spec`:
  - refusals: a line unanswered, the template placeholder left, a `no` without a reason, a `yes` without a path, a `yes` whose path is missing;
  - a pattern `yes` resolves against the vault root, and refuses when the vault root is unresolvable;
  - valid answers archive.
- [ ] [AC5] Implement the pre-flight in `spec.Archive`, next to the review pre-flight.
- [ ] [AC6] Change the answer grammar in `cli/internal/spec/templates/verification.md` to `yes: <path>` or `no: <reason>`.

## Slice 4: required (waits on #1451)

- [ ] Add `knowledge-gate` (app 15368) to dotfiles' required checks in `forge/branch-protection.json`, and apply it with `dotf forge protection apply` (#1746, merged). Its preflight refuses a context that has not reported on one of the last 5 merged PRs, so this waits until the wiring has run. `dotf forge protection check` is then clean.

## Closing

- [ ] Every acceptance criterion has a feature in `features.json` with a command that fails without the change.
- [ ] `verification.md` filled; independent adversarial review before the archive.
