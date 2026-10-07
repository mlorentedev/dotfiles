---
id: "lesson-346-a-ci-cache-other-prs-can-read-is-one-main-wrote"
type: lesson
status: active
title: "A CI cache other PRs can read is one main wrote"
created: "2026-10-07"
---

# A CI cache other PRs can read is one main wrote

## Context
`test-windows` reinstalled the pinned pi CLI from npm on every PR: ~130 s of a job whose PR p50
is 508 s (#2047, measured 2026-10-07). An earlier attempt, #1480, had added an npm cache and
was removed for buying nothing.

## The Trap
Two separate ways a cache step looks right and saves nothing:

1. **Caching the download, not the result.** #1480 cached npm's download cache (`~/.npm`). The
   install still ran on every job, resolving, extracting and linking, so the network-bound part
   moved but the cost did not. A cache saves time only if a hit lets the job **skip** the step.
2. **Saving from a PR.** GitHub scopes a cache to the ref that saved it. A cache saved on a PR
   branch is visible to that PR's own re-runs and to nothing else. The base branch's caches are
   visible to every PR. A cache only PRs write is re-created by every PR and shared with none.

## The Solution
For the pi CLI (#2050):
- **Cache what the install writes**, the package directory under `npm prefix -g` plus its three
  shims, not npm's download cache and not the whole prefix. On a hit, setup's existing
  idempotent branch runs (`pi --version` against the pin), and the install is skipped.
- **Key on every input of the install, exact match only:** OS, the runner `ImageVersion`,
  `PI_VERSION` and `hashFiles('setup-windows.ps1')`, with no `restore-keys`. A PR that changes any
  input misses and runs the fresh install itself, which keeps coverage where a change could break
  it.
- **Only main saves, as the job's last step.** `actions/cache/restore` runs with `lookup-only:`
  on push (it downloads nothing), and `actions/cache/save` runs on a push that missed. So a cached
  install is always a fresh one from a fully green run. A test rejects `always()`, `failure()` or
  `cancelled()` in the save's `if:`, because a status function replaces the implicit `success()`.
- **Print the resolved inputs** (`prefix=C:\npm\prefix image=... version=...`). A PR's run can
  show only a miss. The hit is proven on the first PR after main saves, so the log has to say
  what was keyed.

Pester is the small case of the same rule (#2046): a per-PR save is fine there, because the
cost is a 405 KB module, but the version still comes from `versions.conf`, never the Gallery's
latest.
