# Lesson 266 — Two measurements agreeing is not reproducibility

**Date:** 2026-09-04
**Context:** CI-001 (#1472), measured on #1475 across two sessions

## What happened

`test-windows` was cancelling at its 30-minute ceiling. The log showed the pi
package reconcile consuming most of the job, with one package standing out:

| Sample | `@ayulab/pi-rewind` |
|---|---|
| 2026-09-03 | **421s** |
| 2026-09-04, cold run | **422s** |

Two independent sessions, on different runners on different nights, one second
apart across seven minutes. Both sessions read that as reproducibility and
started reasoning about mechanism.

The evidence got better, which made it worse. `npm view @ayulab/pi-rewind@0.4.6`
returns **0 dependencies, empty `scripts`, 148 KB, 5 files** — the smallest,
simplest package in the manifest, taking 8x longer than the largest (7.5 MB,
69 files, 50s). A tiny package with nothing to build, deterministically eating
seven minutes, is a genuinely interesting fact and it demanded an explanation.

Both sessions produced one, and they disagreed:

- *Retry ladder.* A constant is the signature of timeouts, which are constants.
  Therefore it is on the network path, and `prefer-offline` against a warm cache
  removes the lookup that triggers it.
- *Fixed cost.* A cost that reproduces to the second is fixed, and fixed costs of
  that size are not on the download path.

Both sessions recorded predictions before the data, agreed a falsification
condition, and refined the experiment for an hour. The care was real. It was
spent entirely inside a premise nobody had tested.

Then the warm run landed:

| package | cold | warm | ratio |
|---|---:|---:|---:|
| pi-effort | 136s | 272s | 2.0x |
| pi-web-access | 50s | 255s | 5.1x |
| **@ayulab/pi-rewind** | **422s** | **269s** | **0.6x** |
| @narumitw/pi-plan-mode | 123s | **422s** | 3.4x |
| **@narumitw/pi-goal** | **3s** | **268s** | **89.3x** |
| pi-subagents | 60s | 167s | 2.8x |
| pi-memory | 38s | 141s | 3.7x |
| pi-cc-extensions | 37s | 251s | 6.8x |
| pi-add-dir | 13s | 224s | 17.2x |

Read the warm column as a set: 272, 255, 269, 422, 268, 167, 141, 251, 224.
**There is no slow package.** pi-rewind came in at 269s, indistinguishable from
pi-goal's 268s — and pi-goal was **three seconds** cold.

The outlier did not disappear. It **moved**. The ~422s now sits on pi-plan-mode.
Same magnitude, different package, next run.

## The lesson

**n=2 agreeing is not reproducibility. It is two samples that agreed.**

Two points cannot distinguish "constant" from "high-variance distribution that
happened to be sampled twice near the same value". The tighter the agreement, the
more convincing the coincidence looks — 421 vs 422 read as *more* rigorous than
421 vs 460 would have, when it carried no more information about the underlying
distribution.

Three failure modes stacked, and the third is the one that made it expensive:

1. **Agreement was read as low variance.** Nobody asked what the spread was,
   because two points cannot show a spread.
2. **Corroborating evidence was mistaken for confirmation.** The `npm view` data
   was real, precise, independently verified, and answered a question that did not
   exist. Evidence for a phenomenon cannot be evidence that the phenomenon exists.
   It was the most convincing part of the case and the most misleading.
3. **Rigour was applied inside the premise instead of to it.** Predictions before
   data, an agreed falsification condition, a named dead zone, an explicit refusal
   to claim a mechanism without log support. Every one of those disciplines is
   correct. All of them operated on "why is pi-rewind slow" and none on "is
   pi-rewind slow". Careful reasoning downstream of an untested premise is more
   dangerous than sloppy reasoning, because it produces confidence.

The prior lesson (265) is the same family: a correct measurement answering the
wrong question. This is its sharper case — a correct measurement answering a
question about a phenomenon that was never established.

## What to do instead

- **State n out loud whenever a number is called reproducible.** "Reproducible
  (n=2)" would have stopped both sessions in one line. Reproducibility is a claim
  about a distribution; two points do not describe one.
- **Before explaining a pattern, get a third sample.** Cheaper than an hour of
  mechanism. In this case the third sample was a single CI rerun, available from
  the start.
- **Watch for the outlier moving, not just shrinking.** A per-item anomaly that
  relocates between runs is per-invocation variance, not an item property. If the
  identity of the extreme value is not stable, no explanation of that item can be
  right.
- **Treat strong corroborating evidence as a reason to re-check the premise, not
  as permission to stop.** The moment the story got compelling was the moment to
  ask whether the thing being explained was real.

## What survived, and it was never the interesting part

The measurement did answer the question it was run to answer, which is the only
reason the night was not wasted: a confirmed 310 MB cache hit made the package
phase **2.6x slower** (882s to 2269s), nine of ten packages regressed, and the
job was **cancelled at 45 minutes** on a ceiling the uncached run cleared by
seven. The npm download cache does not fix CI-001; on the only warm run ever
measured it is the difference between a job that completes and one that dies.

Note that this conclusion never needed a mechanism, and both sessions nearly
traded it for one.

## Coda: the instrument was blindfolded the whole time

Both setup twins discard `pi install`'s output entirely —
`setup-windows.ps1:1326` (`2>$null | Out-Null`) and `setup-linux.sh:893`
(`>/dev/null 2>&1`), stdout and stderr, on the package loop specifically. So
"does npm emit retry warnings during those seven minutes" had no answer in the
artifact and never did. Both mechanisms were unfalsifiable against the only
evidence available, and neither session checked that before designing an
experiment around it.

Timings survived because they come from the script's own `Write-Info` lines,
which is the only reason the table above exists. **Before designing an experiment
around an artifact, confirm the artifact can distinguish the outcomes.**
