---
id: "CI-003"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-04"
issue: "mlorentedev/dotfiles#1486"
tags: [spec, proposal]
template_version: "1.0"
---

# CI-003

## Why

CI-002 took the pi package reconcile off the PR path and said so plainly: *"the
reconcile now runs ONLY on pushes to main… the guard's value rests entirely on a red
main being noticed."*

**The reconcile's cost varies 24x on byte-identical input**, and it sits inside a job
with a 45-minute ceiling. So whether main verifies Windows setup at all is a coin flip
on registry behaviour rather than a property of the code. Five pushes to main on
2026-09-04, same nine version-pinned packages:

| run | package phase | setup step | outcome |
|---|---|---|---|
| `33845866062` | (killed mid-loop) | 45m06s | **cancelled at ceiling** |
| `33846609699` | — | — | **cancelled before any job started** (#1487) |
| `33847200968` | **≥2430s** | 44m19s | **cancelled at ceiling** |
| `33930483404` | **101.8s** | 6m24s | success |
| `33930629978` | — | 6m57s | success |

Two of the three that reached a verdict produced **no Windows verification at all** —
`Post-setup doctor gate`, `Run Pester suites` and `Run PowerShell bats subset` all
`skipped` — and `cancelled` is neither a pass nor a failure, so nothing goes red about
it. The successful run reports `pi packages: 9 installed, 0 already present, 0 failed`,
so 101.8s is a complete reconcile and not a run that skipped work.

> An earlier draft of this section said main "does not complete, so it verifies
> nothing". That was true of the three morning runs and **false of the current head**;
> a peer session pointed at the two later ones and checking them directly confirmed it.
> Corrected rather than deleted, because three cancellations read as a state is the same
> sampling error the 421s constant is made of — and the replacement is a stronger claim,
> not a retreat.

The obvious response is to bound the reconcile. **It cannot be bounded yet**, and that
is what this spec is for. Both twins discarded the install's stdout *and* stderr and
logged no elapsed time, so between two consecutive `Installing pi package` lines the log
holds literally nothing. A 421s install that **succeeded slowly** and one that **failed
after retries** are indistinguishable, and they demand opposite bounds: a bound that
kills a slow success turns working setup into broken setup; a bound on an install that
was going to fail anyway is free.

## What

Make the reconcile observable, in both twins, and nothing else.

1. **Log elapsed seconds for every install, on every outcome** — fast success, slow
   success, failure. This is the line that answers *"did the 421s end in success?"*, and
   no amount of captured output answers it on its own.
2. **Capture the install's output instead of discarding it**, and emit it, fenced, when
   the install fails **or** crosses a verbosity threshold. The message it replaces
   (`run 'pi install X' to see why`) asked a human to reproduce by hand, later, a
   failure the machine held in a variable at the time — and on the Windows runner that
   reproduction is not available at all.

## Out of scope

- **The bound itself.** Chosen from the data step 1 produces, in the next PR. The
  measured distribution is 35–345s for normal installs and 421 ±1s for the anomaly, so
  there is no clean gap to sit in and any threshold picked today is a guess. #1486
  §What.2 states this ordering.
- **The 45-minute ceiling.** #1475 raised it 30 → 45 as an explicitly temporary loan and
  recorded that the next slow night would eat 45 the way that one ate 30. This is that
  night. Not raised again here.
- **The concurrency hole** — a third push to main discarding the second *pending* run.
  Real, measured on `33846609699`, filed as #1487.
- **The wider path-filter audit** (#1478 PR 2).
- **#1484** — `setup-linux.sh`'s reconcile has still never executed in CI. The Linux
  half of this change is therefore verified structurally and by mutation, not end to
  end. Unchanged by this spec, and stated rather than implied.

## Risks / open questions

- **The fix carries its own hazard, on the Windows side only.** `2>&1` on a *native*
  command turns its stderr lines into `ErrorRecord`s, and under
  `$ErrorActionPreference = 'Stop'` those terminate — so an install that writes a
  warning to stderr and exits 0 would abort setup. The redirect being replaced
  (`2>$null`) never met that hazard because it threw stderr away, which means the hazard
  arrives *with* the fix rather than being pre-existing. The preference is pinned to
  `Continue` around the call and restored in a `finally`, and AC6 asserts all three
  parts.
- **The verbosity threshold is a literal, not a knob.** Crossing it kills nothing; it
  only decides whose output is printed, so being wrong costs log lines rather than a
  broken install. Made explicit in both twins' comments because a number next to a
  duration reads like a timeout.
- **Captured output reaches the transcript.** It is npm's, on a CI runner with no
  registry credentials configured, and it is emitted only on failure or on a slow
  install. Judged acceptable; noted because "capture and print" is a shape that deserves
  the question asked out loud rather than skipped.
- **A real machine's normal run looks the same as before.** Installs under the threshold
  log one extra line each and print nothing. Nine packages on a fresh machine that is
  having a bad night will print nine blocks — which is the point.

## Acceptance criteria

- [ ] AC1 — Neither twin discards the install's output; the call captures it.
- [ ] AC2 — The output is held, not streamed, so a normal run is not interleaved.
- [ ] AC3 — Elapsed seconds are logged on **every** outcome in both twins: fast success,
      slow success, failure.
- [ ] AC4 — A failure emits what happened; the "reproduce it by hand" advice is gone
      from the reconcile block in both twins.
- [ ] AC5 — The captured output is fenced, so an install that printed **nothing** is
      legible as such rather than indistinguishable from "nothing was captured".
- [ ] AC6 — The Windows capture pins and restores `$ErrorActionPreference`, with the
      restore in a `finally`.
- [ ] AC7 — The verbosity threshold controls printing only: nothing skips, kills or
      fails an install for crossing it.
- [ ] AC8 — Every assertion above is shown to fail against a mutated tree, with the
      mutation's own diff as evidence that it landed inside the reconcile block.

## References

- Issue: https://github.com/mlorentedev/dotfiles/issues/1486
- #1472 — the closed predecessor whose diagnosis ("npm registry latency") this corrects.
- #1478 / CI-002 — took the reconcile off the PR path; this closes the hole that opened.
- #1487 — the concurrency hole found in the same measurement.
- #1484 — the Linux reconcile has never run in CI.
- Lesson 267 — a mutation harness must prove the mutation landed. Applied to AC8.
