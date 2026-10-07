---
id: "lesson-339-a-setup-script-does-what-the-scripts-it-calls-do"
type: lesson
status: active
title: "A setup script does what the scripts it calls do"
created: "2026-10-06"
---

# A setup script does what the scripts it calls do

## Context
While designing the instructions-first step of `dotf converge` (PLAT-001b, #2016), the two
setup scripts were compared to find which steps one ran and the other did not. `setup-windows.ps1`
calls `dotf harness presence`; a grep of `setup-linux.sh` for the same command found nothing.

## The Trap
That absence was filed on epic #2013 as a finding (F-061: "Linux never injects the persona
presence region"), and a second one (F-062: "the setup's CLAUDE.md copy drops the region") was
built on it. Both were wrong. `setup-linux.sh` runs `scripts/compile-harness.sh --deploy`, which
injects the region through `dotf harness presence`, after the setup's own copy. The grep answered
"does this file contain the string", and the finding claimed "does this script do the thing".
The design built on it would have added a second presence writer on Linux and treated a
churn-only defect as data loss.

## The Solution
Before filing a twin-drift finding, or designing around one, follow the calls: list what the
script invokes (`grep -nE '\./scripts/|dotf |bash ' setup-linux.sh`) and read the callee for the
behaviour. A Windows script with an inline port and a Linux script that delegates are the normal
shape here, so "absent from the Linux script" is a lead to check, not a finding. Both findings
were corrected on #2013 the same day, and the reconciler now runs `compile-harness.sh --deploy`
itself instead of duplicating its steps.
