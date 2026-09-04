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

## The same error again, one hour later, inside this document

The first draft of this lesson said the cache made the package phase **2.6x
slower** — 882s cold against 2269s warm — and called it the finding that survived.

It does not survive. A third run settled it, and it was already on disk when that
sentence was written: the post-merge run on `main` was **cold** (no `npm-pi` cache
hit in its log; the main-scoped cache was created *by* that run at 03:41:27) and
its package phase was **2201s**.

| run | package phase | cache |
|---|---:|---|
| PR, cold | 883s | no |
| **main, cold** | **2201s** | **no** |
| PR, warm | 2269s | yes |

**Two cold runs differ by 2.5x.** The warm run sits beside the slower cold one,
not beyond it. So "2.6x slower with a confirmed cache hit" was n=1 against n=1
across a distribution whose spread had never been measured — the exact error this
lesson is about, committed in the paragraph claiming to have escaped it.

What survives is narrower and needs no mechanism: **the cache produced no
benefit, so the download leg is not where the time is**, and CI-001 does not close
on the change. What does *not* survive is that the cache is harmful — nor the
mechanism proposed for it (Defender scanning a 310 MB cache directory, predicting
a slowdown proportional to cache size), which the cold `main` run refutes by
reproducing the slowdown with no cache at all. That hypothesis was hedged at
length, labelled explicitly as not a claim, and still wrong; the hedging bought
nothing that the next data point did not buy outright.

## The constant was real. Both sessions attached it to the wrong noun.

Per-package on `main`'s cold run:

| package | time |
|---|---:|
| **pi-effort** | **421.4s** |
| pi-web-access | 63.5s |
| @ayulab/pi-rewind | 122.4s |
| @narumitw/pi-plan-mode | 53.5s |
| @narumitw/pi-goal | 405.7s |
| pi-subagents | 132.2s |
| **pi-memory** | **421.5s** |
| **pi-cc-extensions** | **422.5s** |
| pi-add-dir | 158.2s |

Four packages at ~420s in a single run. Every observation of that band across all
runs:

| run | package | time |
|---|---|---:|
| 2026-09-03 | pi-rewind | 421s |
| PR cold | pi-rewind | 422s |
| PR warm | pi-plan-mode | 422s |
| main cold | pi-effort | 421.4s |
| main cold | pi-memory | 421.5s |
| main cold | pi-cc-extensions | 422.5s |

**Six observations at 421 ± 1s across four distinct packages.** That is a fixed
timeout — seven minutes exactly — landing on whichever install happens to hit it.
How many installs hit it per run is what moves the package phase from 883s to
2200s, and it is why "27 of 30 minutes", "22-23 minutes" and "43 minutes" are all
descriptions of the same job.

So the constant both sessions saw was genuine. It is a property of the
**operation**, not of any package. The retry-ladder hypothesis was directionally
right about a timeout and wrong about what it belonged to — which is not a
vindication. It was right for a reason its author could not have known, defended
with `npm view` evidence that was irrelevant to it, and it would have been
abandoned entirely if the third run had not arrived.

## The compounding lesson

The first correction took an hour and a third data point. The second took ten
minutes and a fourth. Both were available from the start: a CI rerun and a
post-merge run that had already happened.

**When a conclusion is expensive to be wrong about, spend the next sample before
publishing, not after.** Every wrong claim in this document was cheap to falsify
and none of them were falsified before they were written down.

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
