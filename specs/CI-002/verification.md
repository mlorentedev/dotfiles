---
tags: [spec, verification]
created: "2026-09-03"
---

# Verification — CI-002 (PR 1: the reconcile skip)

## Commands run, in this session

```
bash -n setup-linux.sh                       OK
zsh  -n setup-linux.sh                       OK
shellcheck setup-linux.sh                    15 pre-existing infos, 0 new
python3 -c yaml.safe_load(ci.yml)            YAML OK
bats tests/pi-packages.bats                  22/22 ok  (17 before, 5 added)
features.json f1..f5                         5/5 PASS, each executed
```

PowerShell is not parseable locally (`pwsh` absent on this machine); the twin's syntax is
covered by CI's `lint-powershell`, which gates on `**/*.ps1`.

## AC6 — the assertions were mutation-tested, not merely run

A test that passes on first write is not evidence. Five mutations, each applied to a real
file and reverted:

| # | Mutation | Result |
|---|---|---|
| M1 | `DOTFILES_SKIP_PI_PACKAGES: "1"` unconditionally (fires on `push`) | **caught** — f21 |
| M2 | drop `setup-windows.ps1` from the `pi` filter | **caught** — f22 |
| M3 | move the Linux guard below the `$PI_BIN` probe | **caught** — f19 |
| M4 | move the Windows guard below the `Get-Command pi` probe | **caught** — f19 |
| M5 | replace the loud message with a bare "skipped" | **caught** — f20 |

## Two defects the mutation pass found in the tests themselves

Both are the failure class this repository keeps cataloguing — a green result that measured
the wrong thing — and neither was visible from a passing run.

1. **`grep -n 'X' file | grep -v '^ *#'` does not filter comments.** `grep -n` prefixes a
   line number, so every line starts with a digit and `^ *#` never matches. The ordering
   assertion was anchoring on the *explanatory comment* above the block, which precedes
   everything, so it passed regardless of where the guard actually sat. M3 did not fail
   against it.
2. **`[ ! -x "$PI_BIN" ]` also appears ~150 lines earlier**, in pi's own install block. A
   file-wide search found that one and compared against the wrong branch, so the corrected
   assertion failed on the *clean* tree. Fixed by slicing the reconcile block with `awk`
   before comparing.

## What is NOT verified here

- **The Linux guard end to end.** `integration`'s container has no npm, so
  `setup-linux.sh`'s reconcile block has never executed in CI at all — it logs
  `npm not found — skipping pi package reconcile` and stops. Pre-existing gap, recorded in
  the proposal, not introduced or fixed by this PR. The Linux guard is verified
  structurally and by mutation only.
- **The end-to-end effect on job duration.** That is measured by the first `pull_request`
  run after this lands, and it is owed rather than claimed.
- **Why an install costs ~421s.** Out of scope; #1472.
