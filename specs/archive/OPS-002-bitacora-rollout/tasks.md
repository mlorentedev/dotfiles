---
tags: [spec, tasks]
created: "2026-06-09"
---

# Tasks - OPS-002-bitacora-rollout

> One task = one focused change. Tick as you go.

## Setup

- [x] Branch created from `origin/main`: `feat/bitacora-rollout-script` (worktree)
- [x] `proposal.md` complete; decision (per-repo Action) locked with owner in #258
- [x] Pre-flight: PAT coverage + `gh` token `workflow` scope verified

## Implementation

- [x] `scripts/bitacora-rollout.sh` — discovery, link, secret, workflow deploy (diff-aware contents API), backfill, `--check`, change-count summary
- [x] Runbook §7: register the script + record the decision

## Verification

- [x] shellcheck clean
- [x] `--check` dry-run against knowledge + pollex (detects drifted workflow copies, pending link)
- [x] Post-merge: full live rollout; re-run → 0 changes (AC5)
