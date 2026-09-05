---
tags: [spec, verification]
created: "2026-09-04"
---

# Verification — CI-003

All commands run on this branch (`ci/ci-003-observable-bounded-reconcile`, off `9c2758a`),
in this session. `features.json` entries stay `pending`: the agent may not write the
terminal `passing` state, so the executed output lives here.

## The measurement this spec answers to

Two pushes to main, same nine version-pinned packages, same runner image. Per-package
durations derived from the runner timestamps of consecutive `Installing pi package`
lines — the only place they exist, because neither twin logs them:

| package | `33847200968` (cancelled) | `33930483404` (success) |
|---|---:|---:|
| `npm:pi-effort@0.0.8` | **421.4s** | 1.4s |
| `npm:pi-web-access@0.24.2` | 195.2s | 42.0s |
| `npm:@ayulab/pi-rewind@0.4.6` | **421.7s** | **1.8s** |
| `npm:@narumitw/pi-plan-mode@0.53.0` | 309.6s | 13.2s |
| `npm:@narumitw/pi-goal@0.53.1` | **421.5s** | 2.3s |
| `npm:pi-subagents@0.56.0` | 35.2s | 12.7s |
| `npm:pi-memory@0.4.2` | 42.2s | 1.7s |
| `npm:pi-cc-extensions@0.8.67` | 345.0s | 25.2s |
| `npm:pi-add-dir@1.3.1` | ≥238.4s (killed) | 1.6s |
| **total** | **≥2430s** | **101.8s** |

The right-hand run reports `pi packages: 9 installed, 0 already present, 0 failed`, so
101.8s is a *complete* reconcile — not one that skipped work. **24x on identical input.**

The rows a package-shaped explanation cannot survive: `@ayulab/pi-rewind` at **421.7s and
1.8s** — the package #1472 singled out as the slow one — and `pi-memory` at **421.5s,
42.2s and 1.7s**, three draws on one pinned version. Ten observations in a 421 ±1s band
across seven packages.

Between two consecutive `Installing pi package` lines the log holds **nothing** — no
success, no failure, no output. That is what this PR changes, and it is why no bound is
chosen here: what settles the bound is whether those 421s installs succeed or fail, and
that is precisely what is unlogged.

## AC-by-AC

| AC | Covered by | Result |
|---|---|---|
| AC1 | `neither twin discards the install's output` | pass |
| AC2 | `the install's output is CAPTURED, not merely streamed` | pass |
| AC3 | `every install logs its elapsed time, on EVERY outcome` | pass |
| AC4 | `a failure emits what happened instead of telling the reader to redo it` | pass |
| AC5 | `the captured output is FENCED, so empty output is legible` | pass |
| AC6 | `the PowerShell capture cannot be terminated by a noisy install` | pass |
| AC7 | `the slow threshold gates VERBOSITY only, never the install` | pass |
| AC8 | `specs/CI-003/mutate-assertions.py` | 10/10 caught |

## AC8 — the mutation run

`python3 specs/CI-003/mutate-assertions.py` → exit 0. Every case prints the mutated line,
before and after, at its line number inside the reconcile block, **before** the suite runs:

```
CAUGHT  AC1  neither twin discards          [setup-linux.sh]
CAUGHT  AC1  neither twin discards          [setup-windows.ps1]
CAUGHT  AC2  output is CAPTURED             [setup-linux.sh]
CAUGHT  AC3  elapsed time, on EVERY outcome [setup-linux.sh]
CAUGHT  AC3  elapsed time, on EVERY outcome [setup-windows.ps1]
CAUGHT  AC4  emits what happened            [setup-linux.sh]
CAUGHT  AC5  FENCED                         [setup-linux.sh]
CAUGHT  AC5  FENCED                         [setup-windows.ps1]
CAUGHT  AC7  VERBOSITY only                 [setup-linux.sh]
CAUGHT  AC6  terminated by a noisy install  [setup-windows.ps1]
```

Sample, showing the diff that makes the verdict a fact rather than a hypothesis:

```
== AC1  neither twin discards  [setup-linux.sh]
    @ setup-linux.sh:926
      -if pi_pkg_out=$("$PI_BIN" install "$pi_pkg" 2>&1); then
      +if pi_pkg_out=$("$PI_BIN" install "$pi_pkg" >/dev/null 2>&1); then
      1 test(s) selected -> CAUGHT
```

## Both directions, on the two features that print nothing when they pass

A command whose passing output is empty is exactly the shape that passes for the wrong
reason, so each was run with its guarantee removed:

- **f4** (shell syntax + shellcheck + ASCII-only `.ps1`): appended a single `é` to
  `setup-windows.ps1` → **exit 1**. Restored.
- **f6** (the harness must report a HARNESS FAULT rather than a verdict): replaced one
  mutation target with a string that does not exist → the harness printed
  `HARNESS FAULT: target not found in setup-linux.sh at/after line 866` and exited **1**.
  Restored.

Without these two runs, both entries were green commands that had not been shown capable
of being red.

## The harness's own defect, and why it is called out

The previous mutation harness (lesson 267) did a repo-wide `replace(x, '', 1)`, deleted
an identically-spelled line from a *different* filter block, and reported four sound
assertions as vacuous. The one committed here refuses that outcome by construction:

- mutations are located **at or after the reconcile block's anchor**, never repo-wide;
- the mutated line is printed before and after, so "the mutation landed" is observed;
- an identical file, a missing target, or a `-f` filter selecting zero tests is reported
  as **HARNESS FAULT** and never as `NOT CAUGHT` — the two are no longer spelled the same.

The same reasoning drove a decision *before* any assertion was written: the block anchors
were checked for uniqueness first. `pi install` appears in pi's **own** install block ~130
lines earlier in `setup-windows.ps1`, carrying its own "run: npm install -g …" advice, so
a file-wide assertion for AC4 would have compared against the wrong code and passed.

## One test's first draft was wrong, and the failure is kept in the comment

`the slow threshold gates VERBOSITY only` first searched for the words
`break|continue|exit|return` unanchored. It matched the literal `exit` inside the fence's
own `(exit $pi_pkg_rc, …)` label and reported control flow that is not there — a false
finding, which is the same disease as a false pass. Now anchored at the start of a
statement, with the reason recorded in the test.

## Shell + lint layer

```
bash -n setup-linux.sh                          OK
zsh  -n setup-linux.sh                          OK
shellcheck --severity=error setup-linux.sh      OK
setup-windows.ps1                               ASCII-only (0 non-ASCII chars)
git diff --check                                OK
tests/pi-packages.bats                          30/30 pass (23 -> 30)
tests/*.bats (full suite)                       see PR body
```

The `.ps1` check is not decoration: `.gitattributes:36` marks `*.ps1 text eol=crlf`, and
PSScriptAnalyzer fails CI on a non-ASCII `.ps1` without a BOM.

## Not verified here, and stated rather than implied

- **The Linux twin's reconcile has never executed in CI** (#1484): the `integration`
  container has no npm, so the block logs `npm not found` and skips. The Linux half of
  this change is therefore verified structurally and by mutation, not end to end.
- **The Windows twin runs for real on this PR.** `setup-windows.ps1` is in the `pi`
  filter, so `test-windows` performs the full reconcile against the new code — which is
  both the behavioural test for AC1–AC6 and the run that produces the data #1486 §What.2
  needs. If it is cancelled at the ceiling again, the captured output up to the kill is
  still in the log; a cancelled run keeps its log, which is the property that made this
  spec's own measurement possible.
