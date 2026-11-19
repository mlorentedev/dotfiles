---
tags: [spec, tasks]
created: "2026-11-19"
id: "HARNESS-201-force-with-lease-own-branch"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-201-force-with-lease-own-branch

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch `feat/3607-force-with-lease-own-branch` in lane `feat-lane-secondary` (`dotf lane switch`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] Open questions in `proposal.md` are decisions for the owner, named in the PR; none blocks the work

## Implementation

- [x] [AC1] Write the failing command table (`pushguard_test.go`) against a stub guard; 104 subtests failed
- [x] [AC1] Implement the lexer and the guard (`pushguard_lex.go`, `pushguard.go`); table green
- [x] [AC1] Mutation-check the guard: seven mutations (drop the `main`/`master` backstop, ignore abbreviations, ignore an unknown directory, ignore the checked-out branch, ignore `+`, allow unknown options, accept a short sha) each fail the table
- [x] [AC2] [AC3] Hook command `dotf harness push-guard` and its tests against a real temporary repository, `origin/HEAD` set, unset and pointing at another branch; status is 0 or 2 on garbage input
- [x] [AC3] `nativeDir` converts `/c/x`, `/cygdrive/c/x` and `C:\x` for a native git on Windows
- [x] [AC4] Bind the hook beside the gate; test both survive a merge over a foreign `PreToolUse` group and that the manifest command is a registered subcommand
- [x] [AC5] Narrow the deny globs, add `land:push`, pin deny and allow lists, match a command table against the rules
- [x] [AC5] Update `tests/claude-settings-template.bats`, which pinned the old broad glob
- [x] [AC6] Deploy test: the old broad glob is retired through the record; and stays without one
- [x] [AC7] ADR-065 and lesson 614

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test (see `verification.md`)
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass (`go vet`, `GOOS=windows go vet`)
- [x] Lint passes (`golangci-lint run`, `GOOS=windows golangci-lint run`)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [ ] PR opened as a draft, then marked ready after the independent review
