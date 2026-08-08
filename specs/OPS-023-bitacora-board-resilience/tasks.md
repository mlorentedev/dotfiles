---
tags: [spec, tasks, templates]
created: "2026-08-08"
---

# Tasks - OPS-023-bitacora-board-resilience

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Worktree from `origin/main`: `dotfiles-wt-809`, branch `fix/bitacora-board-resilience`
- [x] Gating issue open: dotfiles#809
- [x] #774 (OPS-022) consolidated into #809 and closed as superseded, with its
      root-cause analysis migrated rather than lost
- [x] `proposal.md` complete, acceptance criteria testable

## Implementation

- [x] [AC1-AC4] Write failing tests against a stubbed `gh` (and a deliberately
      failing `age`, so reaching the decrypt step fails the run loudly)
- [x] [AC3] `--backfill-only` skips the secret resolution entirely
- [x] [AC1][AC4] Extract `backfill_repo()` — the first cut duplicated step 4 into
      the new branch; one definition, two call sites
- [x] [AC2] Guard placed before steps 1-3 so provisioning is unreachable
- [x] [AC5] Three-way classification in `add-to-project.yml`, with a dedicated
      soft-fail exit code so the branches cannot collapse silently
- [x] [AC6] Both events on one GraphQL call, node ID from the payload
- [x] [AC7] Charset validation on the `workflow_dispatch` input before splitting
- [x] [AC8] `bitacora-reconcile.yml`: daily cron, notice on success, dedup'd
      labelled issue + red on its own failure
- [x] [AC5-AC8] Structural cases read the workflows with comment lines stripped —
      the first cut grepped raw text and matched the headers' own prose

## Closing

- [x] Every acceptance criterion covered by at least one test (9 cases)
- [x] `shellcheck` + `bash -n` clean on the modified script
- [x] Workflows parse as YAML; `actionlint` reports only a pre-existing
      info-level SC2016 that `main` already carries on the same construct
- [x] Full suite: no new failures
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder, with `Refs #809` (not `Closes`) —
      the archive lands in a follow-up PR, keeping both halves of the gate
      satisfiable without the #800 fix being merged first
