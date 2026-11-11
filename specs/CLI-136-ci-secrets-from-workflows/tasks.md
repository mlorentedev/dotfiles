---
tags: [spec, tasks, templates]
created: "2026-11-10"
id: "CLI-136-ci-secrets-from-workflows"
type: spec-tasks
status: active
owner: manu
---

# Tasks - CLI-136-ci-secrets-from-workflows

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/ci-secrets-from-workflows`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (each one has its answer or its pinning test)

## Implementation

Package `cli/internal/secrets` (`ciwf.go`): pure functions over workflow bytes, then the fleet walk behind a reader seam, then the command.

- [ ] [P] [AC1] Failing test: `workflowReads` collects `secrets.X`, `secrets['X']`, `secrets["X"]` from a workflow, excluding `GITHUB_TOKEN`; a caller's explicit mapping yields the name it maps from
- [ ] [AC1] Implement `workflowReads`
- [ ] [P] [AC4] Failing test: `toJSON(secrets)` and `secrets[matrix.k]` come back as unknown with the expression, not as names
- [ ] [AC4] Implement the unknown branch
- [ ] [P] [AC2] [AC5] Failing test: a job `uses: owner/repo/.github/workflows/w.yml@sha` with `secrets: inherit` yields the callee's `on.workflow_call.secrets` keys, each marked required or optional; a bare `on:` key parses as the string (go.yaml.in/yaml/v3)
- [ ] [AC2] [AC5] Implement `inheritedCalls` and `calleeSecrets`
- [ ] [AC1] [AC2] [AC3] [AC5] Failing test: `CheckRepo` against a fake reader and a registry diffs the read set with `SelectCI`'s upload list into missing (optional ones marked), excess and unknown
- [ ] [AC1] [AC2] [AC3] [AC5] Implement `CheckRepo` (callee fetched once per workflow and ref)
- [ ] [AC1] Command `dotf secrets ci-check [--repo]...`: per-repository report, names only, non-zero exit on missing or excess; the fleet from `gh repo list` when no `--repo` is given
- [ ] [AC1] Live run over the fleet, with the output recorded in `verification.md`

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

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-136-ci-secrets-from-workflows/features.json`):

```json
[
  {
    "id": "CLI-136-ci-secrets-from-workflows-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
