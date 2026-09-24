---
tags: [spec, tasks, templates]
created: "2026-09-04"
---

# Tasks - CI-003

> TDD order. One task = one focused commit.
>
> `[P]` — no dependency on another unchecked task. `[AC<n>]` — satisfies acceptance
> criterion `<n>` from `proposal.md`.

## Setup

- [x] Branch created from main: `ci/ci-003-observable-bounded-reconcile` off `9c2758a`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] Verify the block anchors are UNIQUE before writing any assertion on them —
      `pi install` appears in pi's own install block ~130 lines earlier in the
      PowerShell twin, so a file-wide search reads the wrong code. Done first, on
      purpose: this is the step lesson 267 says gets skipped.
- [x] [AC1][AC2] `setup-linux.sh` — capture the install's output and its elapsed time
      instead of `>/dev/null 2>&1`
- [x] [AC3][AC4][AC5] `setup-linux.sh` — log elapsed on all three leaves; emit the
      captured output, fenced, on failure or over the verbosity threshold; drop the
      "run it by hand to see why" advice
- [x] [AC1][AC2][AC3][AC4][AC5] `setup-windows.ps1` — the same, as the twin
- [x] [AC6] `setup-windows.ps1` — pin `$ErrorActionPreference` to `Continue` around the
      native call and restore it in a `finally`. The hazard arrives WITH the fix: the
      redirect being replaced never met it because it discarded stderr
- [x] [AC7] Both twins — the threshold is a literal named as a VERBOSITY knob, used only
      by the two comparisons that decide whether to print
- [x] Seven assertions in `tests/pi-packages.bats`, sliced from the reconcile block to a
      file so `lib/refute`'s negative assertions apply (`! grep` mid-body cannot fail a
      bats test)
- [x] [AC8] `specs/CI-003/mutate-assertions.py` — ten mutations, each located by line
      position inside the block, each printing its own diff before any verdict

## Closing

- [x] Every acceptance criterion covered by at least one test
- [x] Every acceptance criterion has a `features.json` entry with a non-vacuous,
      **executed** verification command
- [x] Lint passes: `bash -n`, `zsh -n`, `shellcheck --severity=error`, ASCII-only `.ps1`
- [x] No unrelated changes in the diff — `ci.yml` is deliberately untouched
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Deliberately NOT done

- **No bound.** #1486 §What.2 orders it: the bound comes from the data this produces.
  Chosen today it would be a guess between "35–345s is normal" and "421s is the
  anomaly", with no gap between them.
- **No ceiling change.** #1475's raise was a loan and this is the night it came due;
  raising it again is the same loan.
- **No `ci.yml` change at all.** `setup-*.{sh,ps1}` are already in the `pi` filter, so
  this PR's own `test-windows` runs the full reconcile and produces the data. Nothing to
  add.
