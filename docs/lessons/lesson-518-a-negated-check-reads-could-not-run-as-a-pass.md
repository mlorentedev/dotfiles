---
id: "lesson-518-a-negated-check-reads-could-not-run-as-a-pass"
type: lesson
status: active
title: "A negated check reads could-not-run as a pass"
created: "2026-10-22"
---

# A negated check reads could-not-run as a pass

## Context

`dotf pr reproduce` runs a check and records one of four states. A start failure, a shell's 126 or 127, a signal
or a timeout is `could-not-run`; any other non-zero exit is `reproduced`; zero is `did-not-reproduce`. Some
findings claim a guard can be evaded. For those the defect exists when the guard's own test passes, so the
check has to be inverted.

## Problem

The #1059 replay inverted it the usual way, with `sh -c '! bats tests/ci-age-pin.bats'`. `!` maps every non-zero
exit to 0, including 127. On a machine without `bats`, the shell exits 127 and `!` turns it into 0. The harness
then sees a check that ran and passed, and records `did-not-reproduce`: the REAL finding is refuted by a check
that never ran. The replay itself was correct, because `bats` was installed. The wrapper was wrong on any host
where it is not.

## Solution

`--defect-when-passes` moves the inversion into the harness, after the could-not-run cases are classified, so a
check that never ran stays `could-not-run` in both directions (#2926). The row says `defect when the check
passes`, so a reader knows which way exit 0 was read. The help text names the `sh -c '!'` wrapper as the thing
not to use.

## Rule

Never invert a check's exit code in the shell when its result is classified afterwards. `!` erases the
difference between "failed" and "did not run". Invert after that difference has been read, where the
classifier can still see it. The same holds for `|| true`, `; echo $?` wrappers, and any pipe whose last stage
owns the exit status.
