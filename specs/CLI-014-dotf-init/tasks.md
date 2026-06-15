---
tags: [spec, tasks, templates]
created: "2026-06-14"
---

# Tasks - CLI-014-dotf-init

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/dotf-init`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> TDD order, mirroring the ADR-022 build sequence. Each twin is deleted on contact (strangler-fig). Stage as 1-4 PRs (the package + each subcommand + the orchestrator) or one large PR.

### Step 1 — package + embed + drift guard
- [x] `cli/internal/initrepo/` package + `cli/internal/cmd/init.go`; wire `newInitCmd()` into `root.go`
- [x] Embed templates under `cli/internal/initrepo/templates/`; failing drift-test mirroring `spec/drift_test.go` (embedded == vault SSOT)
- [x] Implement template loading to make the drift-test pass

### Step 2 — `dotf init agents` (port init-repo-agents)
- [x] Failing test: bootstrap `AGENTS.md`+SDD, idempotent re-run, **no `$VAULT_PATH` leak**
- [x] Implement; idempotent regenerate-between-markers + `--force` (+ self-contained vault SSOT fix + render guard, per ADR decision)
- [x] Migrate the bats cases to `go test` (no dedicated init-repo-agents bats existed; coverage was via init-project.bats — moved to `agents_test.go` + `init_test.go`). **`git rm scripts/init-repo-agents.sh` DEFERRED to Step 4** (decided): its caller `init-project.sh` is ported there, so the whole init-twin set + bats die together — avoids a regressed/churned intermediate. `.ps1` stays orphan (#380).

### Step 3 — `dotf init github` (port init-repo-github-defaults)
- [x] Failing test: derive owner/name from `origin`, `--dry-run`, auto-skip without remote/`gh`
- [x] Implement `gh api` PATCH `delete_branch_on_merge`. **Twin + bats deletion DEFERRED to Step 4** (same as agents: `init-project.sh` calls it; the whole init-twin set dies with the orchestrator). `.ps1` stays orphan (#380).

### Step 4 — `dotf init` orchestrator (port init-project)
- [ ] Failing tests: structure + `.gitignore` + pre-commit + stack init + `git init` + CI scaffold + env-contract + AGENTS/github steps + vault entry (auto-skip / `--skip-vault`)
- [ ] Implement; CLAUDE.md thin pointer; host-coupled steps degrade to `[WARN]`
- [ ] Delete `scripts/init-project.sh` + **`scripts/init-repo-agents.sh` (deferred from Step 2)** + **`scripts/init-repo-github-defaults.sh` (deferred from Step 3)** + `scripts/init-repo-standards.{sh,ps1}` (dropped) + their bats (the whole init-twin set dies together with its orchestrator caller; `.ps1` twins stay orphan per #380)

### Step 5 — repoint + guard
- [ ] Guard-grep `init-(project|repo)` returns only provenance (CHANGELOG / ADRs / `specs/`)
- [ ] Repoint `setup-linux.sh`, docs, `AGENTS.md` to `dotf init`

### Step 6 — close out
- [ ] Close #248, #299; tick #275 (HARNESS-013) standalone-AGENTS half
- [ ] Smoke `dotf init` on a throwaway checkout with no vault present

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

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-014-dotf-init/features.json`):

```json
[
  {
    "id": "CLI-014-dotf-init-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
