---
tags: [spec, tasks, templates]
created: "2026-08-07"
---

# Tasks - BUG-047-commit-msg-hook-windows

> TDD order. One task = one focused commit. Tick as you go.
>
> Scoped to findings **3 and 4** of #794. Findings 1 and 2 (`scripts/test.sh` resolving `DOTFILES_DIR` to the deploy target; deployment assertions inside a repo-integrity suite) stay open on the issue as a separate PR.

## Setup

- [x] Worktree created from `origin/main`: `dotfiles-wt-bug047-commit-msg`, branch `fix/validate-commit-msg-accepts-scopes`
- [x] Gating issue open: dotfiles#794
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC1] Add failing tests for scoped subjects, including the four real subjects on `main` as a regression guard, plus breaking-change `!` and separator-bearing scopes.
- [x] [P] [AC2] Add failing tests for malformed subjects: missing space after the colon, empty subject (capitalised type and no-type already covered).
- [x] [P] [AC3] Add failing test: a conforming body line does not rescue a malformed subject.
- [x] [P] [AC4] Add failing tests: `Merge`, `Revert "`, `fixup!`, `squash!` subjects are exempt.
- [x] [AC1] [AC2] [AC3] [AC4] Rewrite the matcher: validate `head -n 1` only, against `^[a-z]+(\([a-zA-Z0-9._/-]+\))?!?: .+`; add the git-generated exemptions; keep the type permissive.
- [x] [AC5] Shebang `#!/bin/sh` -> `#!/usr/bin/env sh`, keeping the body POSIX so the cross-shell contract holds. Add a test asserting the shebang.
- [x] [AC6] `scripts/check-spec-gate.sh` shebang `#!/bin/bash` -> `#!/usr/bin/env bash`, so the pre-push gate executes on Windows.
- [x] [AC7] Guard the three pre-existing zsh cases with `require_zsh` so they skip where zsh is absent instead of failing.
- [x] Error output to stderr, showing the accepted grammar with examples; a missing message-file argument fails loudly rather than silently.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Lint passes (`tests/check-bats-names.bats` 7/7 — it flags non-ASCII `@test` names, which caught an em-dash during development)
- [x] No unrelated changes in the diff (no scope creep) — findings 1/2 deliberately excluded
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

`features.json` is not emitted for this spec: every acceptance criterion maps 1:1 onto a named bats case in `tests/validate-commit-msg.bats`, and the single command `bats tests/validate-commit-msg.bats` is the whole verification surface. A JSON restating that would add a second place to drift.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state.
