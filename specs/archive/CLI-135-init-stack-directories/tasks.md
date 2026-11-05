---
tags: [spec, tasks, templates]
created: "2026-11-04"
id: "CLI-135-init-stack-directories"
type: spec-tasks
status: active
owner: manu
---

# Tasks - CLI-135-init-stack-directories

## Setup

- [x] Branch created from main: `feat/init-stack-directories`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"
- [x] Coordinated with the ci-workflows sessions: dotfiles-only, reference never copy, CI shape deferred to #3163

## Implementation

- [x] [AC1] [AC2] [AC3] [AC4] Write failing tests: `stacks_test.go` (per-stack directory table, `dotf init --stack workflows`, `StackInit` no-op)
- [x] [AC1] [AC3] Add the stack table (`stacks.go`), fold `ciTemplates` into it, split the code directories out of `scaffoldDirs`, and pass the stack through `ScaffoldOpts`
- [x] [AC2] Add the `workflows` stack: `tests/` only, no ci template
- [x] [AC4] `StackInit` treats `workflows` as a no-op; `--stack` help lists `ts` and `workflows`
- [x] Mutation check: each of the three behaviours, when reverted, fails the suite

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder
