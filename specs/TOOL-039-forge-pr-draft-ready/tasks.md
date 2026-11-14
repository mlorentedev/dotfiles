---
tags: [spec, tasks, templates]
created: "2026-11-13"
id: "TOOL-039-forge-pr-draft-ready"
type: spec-tasks
status: active
owner: manu
---

# Tasks - TOOL-039-forge-pr-draft-ready

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/3406-forge-pr-ready`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC1] Failing tests: Gitea `CreatePR` with `Draft` sends `WIP: <title>` and returns the URL when the answer has `draft: true`; an answer with `draft: false` is an error naming the URL.
- [x] [AC1] Implement the Gitea draft create with the read-back.
- [x] [P] [AC2] Failing tests: Gitea `ReadyPR` reads the pull request, strips `WIP:`/`[WIP]` (any case, trailing spaces) through PATCH, succeeds on `draft: false`, fails on `draft: true`, and writes nothing for a ready one.
- [x] [AC2] Implement Gitea `ReadyPR`.
- [x] [P] [AC3] Failing tests: GitHub `ReadyPR` reads `pulls/<n>` through `gh api`, sends the mutation with the `node_id` through `gh api graphql`, and skips it when `draft` is false.
- [x] [AC3] Implement GitHub `ReadyPR`, and add `ReadyPR` to the `Backend` interface.
- [x] [AC4] Failing test, then the `dotf forge pr ready <number>` verb and the help text change for `--draft`.

## Closing

- [x] `go build ./... && go vet ./... && go test ./...`, `golangci-lint run`, and both again with `GOOS=windows`
- [x] Every acceptance criterion has a test and a `features.json` entry with a non-vacuous verification command
- [ ] One manual check on gitea.kubelab.live: a scratch draft opened, marked ready, closed; recorded in `verification.md`
- [x] Follow-up ticket for the harness text (`gh pr ready` to `dotf forge pr ready`), number written in `verification.md`: #3477
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/TOOL-039-forge-pr-draft-ready/features.json`):

```json
[
  {
    "id": "TOOL-039-forge-pr-draft-ready-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
