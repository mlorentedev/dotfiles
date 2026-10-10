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

## A defect this PR's own evidence had, found in review

`features.json` `f2` asserted `bats … | grep -c '^ok ' | grep -qx 22`. Adding the 23rd
test — the derived `PI_VERSION` assertion, itself a review fix — made that command
**fail**, and it is the command the archive gate reads as evidence.

A hard-coded count is brittle in the one direction the spec is guaranteed to move:
adding tests. Rewritten to assert the suite passes **and** that each named guarantee is
present by name, then proven both ways:

| | |
|---|---|
| add a 24th test | f2 still passes — no longer brittle |
| rename a guarantee out of the file | f2 fails — not vacuous either |

Surfaced by PR-Agent quoting the stale command back in its review of `46af234`. It
reported no blocking issues and `#1478 fully compliant`; the defect was visible in what
it quoted rather than in what it said.

## What is NOT verified here

- **The Linux guard end to end.** `integration`'s container has no npm, so
  `setup-linux.sh`'s reconcile block has never executed in CI at all — it logs
  `npm not found — skipping pi package reconcile` and stops. Pre-existing gap, recorded in
  the proposal, not introduced or fixed by this PR. The Linux guard is verified
  structurally and by mutation only. *(Historical, written before #2285: the image has
  carried npm since #2254, and #2285 (closing #1484) now runs the reconcile in
  `integration` and asserts its convergence. See the archive review dispositions below.)*
- **The end-to-end effect on job duration.** That is measured by the first `pull_request`
  run after this lands, and it is owed rather than claimed.
- **Why an install costs ~421s.** Out of scope; #1472.

## Closing status (2026-10-10)

The skip guard moved out of the setup twins into `dotf pi packages apply` (HARNESS-139, #1628). Both twins call that command, so AC1-AC3 now hold through one implementation instead of two. The bats suite pins the CI side (AC4, AC5) and the wording (AC3), and the Go test pins the ordering (AC2). `features.json` f1-f5 were rewritten to point there: the old f1, f2 and f5 grepped and mutated shell blocks that no longer exist.

**A test that could not fail, found closing.** `TestPiPackagesApplySkipIsFirstAndLoud` claims the skip comes before any probe. But its `piRepo` fixture answers every `piLookPath` call, so a mutation that probes for pi first still passed (f5 exited 1 on its first run). The test now fails on any probe, and f5 kills that mutation.

## PR 2: the wider filter audit, declined on measurement

The plan was to split the `code` filter so that a PR touching only Go skips the Windows bootstrap. Measured over the 300 PRs merged since 2026-07-12:

| Changed paths | PRs |
|---|---|
| `cli/**` only | 106 |
| `cli/**` + `tests/**` | 6 |
| Linux shell only | 0 |
| Windows only | 0 |
| Mixed | 161 |
| No code | 27 |

- **The conservative split** skips Windows only for Linux-shell-only PRs, and the reverse. It saves nothing, because neither class occurs.
- **The synthesis's split** skips Windows for Go-only PRs. That drops `test-windows` from about a third of PRs, and the trend widens the cost: setup logic is moving into `cli/**` (setup-script LOC went from 3,907 on 2026-10-03 to 3,069 on 2026-10-10). That job is the coverage that caught hive#484 on dotfiles#2255 this week.
- **Decision (owner, 2026-10-10):** no split. The `pi` filter from PR 1 stays the only path gate on `test-windows`.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? no: the fixture-answers-every-probe defect is an instance of lesson 267 (a mutation harness must prove the mutation landed); the fix is in the test itself.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: declining the split adds no contract; the measurement and decision are recorded here, beside the filter they concern.
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: path-filtered CI is decided per repo from its own PR mix.

## Archive review dispositions (2026-10-10)

The archive review (`PASS WITH GAPS`) is committed unchanged with this PR.

| # | Finding | Disposition |
|---|---|---|
| 1 | `integration` does not receive `DOTFILES_SKIP_PI_PACKAGES`, so it reconciles on every PR | **Declined, measured.** This is now real rather than theoretical: since #2285 (which closes #1484), the image carries npm and the reconcile runs on purpose. `verify-setup.bats` asserts that it converges and that the second run is a no-op. On main run `13:45–13:50Z`, the reconcile line (`changed=9 … 0 failed`) lands 91 s into the docker build, inside a 4m44s job. The 15-minute cost this spec removed was the Windows leg's. Passing the skip to `integration` would turn #1484's convergence test red. |
| 2 | Any non-empty value skips, `"0"` included | **Declined.** "Set means skip" is the contract the workflow is written against: its expression yields `'1'` or `''`. The warning line names the variable, so a mistaken skip is loud and not silent. Parsing truthy strings would add a second spelling of the same switch. |
