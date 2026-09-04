---
id: "CI-002"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-03"
issue: "mlorentedev/dotfiles#1478"
tags: [spec, proposal]
template_version: "1.0"
---

# CI-002

## Why

`test-windows` is a required check whose cost is **unbounded**, and every PR pays it
whether or not its diff can change the outcome. On 2026-09-04 it cancelled two unrelated
PRs at a 30-minute ceiling; the ceiling was raised to 45 and **#1468 then exhausted 45
minutes too** (02:58:19 → 03:43:24, cancelled). Four PRs were open at once with none
mergeable.

The cost is the pi package reconcile — nine serial `pi install` calls, measured at
**883s, 1331s, 2201s and 2269s** across four runs of the same nine *version-pinned*
packages. A Go-only PR (#1471) paid 23m2s of it for a manifest it never touched.

## What

Two changes, one mechanism and one policy.

1. **`DOTFILES_SKIP_PI_PACKAGES`**, honoured by both setup twins as the **first** branch
   of the reconcile guard chain, logging what was *not verified* rather than merely that
   it skipped.
2. **A `pi` path filter** in the `changes` job, and one `env:` line on `test-windows`
   that sets the variable when — and only when — the event is `pull_request` **and** the
   filter is false.

This is PR 1 of CI-002. PR 2 is the wider filter audit (splitting `code`, re-gating each
job), which is deliberately separate: it touches four jobs at once on a workflow whose
only end-to-end verification is the job we are trying not to run.

## Out of scope

- **Why an install costs what it costs.** Seven observations at **421 ±1s across five
  different packages** say it is a fixed ~7-minute timeout landing on whichever install
  hits it, and how many hit it per run is what moves the phase. Identifying it needs the
  `pi install` output, which both twins discard (stdout *and* stderr) on the package
  loop. Tracked under #1472, which this does not close.
- The wider filter audit (PR 2).
- Branch protection settings.

## Risks / open questions

- **After this, the reconcile runs only on pushes to `main`.** That is intended — the
  coverage does *not* move to a schedule nobody reads — but it means `main` becomes the
  only place a reconcile regression can surface, so the guard's value rests entirely on a
  red `main` being noticed. Stated in the workflow comment rather than left implicit,
  because an unread signal is how this defect stayed invisible: `test-windows` was
  `skipped` on 6 of the last 6 main runs before #1475.
- **This is a real coverage reduction, not a relocation.** TEST-003 exists so a PR proves
  setup works end to end on Windows, and the reconcile is a genuine part of setup.
  Detection moves from pre-merge to the next push to `main`. Judged the right trade at
  883–2200s of a 1348–2602s job, recorded here as a decision with its latency named.
- **Not a config key or a `packages.json` entry.** A durable switch would let a real
  machine end up permanently unconverged, which is the opposite of what setup is for. An
  environment variable dies with the process that sets it.
- **`setup-linux.sh`'s reconcile has never run in CI** — the `integration` container has
  no npm, so it logs `npm not found` and skips. Pre-existing gap; the Linux guard is
  therefore verified structurally here, not end to end. Not fixed in this PR.
- Rejected: pre-seeding the live `settings.json` `packages` array in CI. It would make
  the job fast by making the block report as verified without ever executing — a false
  green, the failure class this repository keeps cataloguing.

## Acceptance criteria

- [ ] AC1 — Both twins honour `DOTFILES_SKIP_PI_PACKAGES`.
- [ ] AC2 — The guard is the **first** branch, before the `pi`/`npm`/`jq` probes, in both
      twins.
- [ ] AC3 — The skip logs what was **not verified**, not merely that it skipped, in both
      twins.
- [ ] AC4 — CI never sets the variable on a `push`, so pushes to `main` always reconcile.
- [ ] AC5 — The `pi` filter covers the manifest **and both twins**, so a PR that can
      change the reconcile's behaviour still runs it.
- [ ] AC6 — Every assertion above is shown to fail against a mutated tree, not merely to
      pass against the real one.

## References

- Issue: https://github.com/mlorentedev/dotfiles/issues/1478
- CI-001 (#1472) — the job-duration ticket this does not close.
- #1480 — removes the npm cache, measured to buy nothing.
- HARNESS-041 (#552) — introduced the `changes` job. Its question was *"does this diff
  need CI?"*; this asks *"which part of CI?"*, which one bucket cannot answer.
