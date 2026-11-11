---
tags: [spec, verification, templates]
created: "2026-11-10"
id: "TOOL-036-system-package-upgrade"
type: spec-verification
status: active
owner: manu
---

# Verification - TOOL-036-system-package-upgrade

## Evidence

- [x] AC1 -> `TestSystemUpgrade*Darwin*Plan` (74192f21); on the owner's Mac: `dotf converge --plan --only system-upgrade` -> `index refreshed (brew update); catalog entries up to date`, `0 to change`
- [x] AC2 -> `TestSystemUpgrade*Darwin*Apply` (74192f21), second run asserted changed=0 with no upgrade or housekeeping argv; on the Mac: apply -> `0 changed, 1 ok`, then plan -> `0 to change`
- [x] AC3 -> `TestSystemUpgrade*OutsideCatalog` (table over darwin/linux/windows). Mutation `isBehind(t) || true` failed 9 tests, reverted
- [x] AC4 -> `TestSystemUpgrade*Linux*` (no sudo: skip with reason; sudo: `--only-upgrade` on catalog packages; root: bare)
- [x] AC5 -> `TestSystemUpgrade*Windows*` (per-id upgrade, one failure isolated and named in the step error, detail names only the upgraded; `TestSystemUpgradeWindowsMatchesOnlyTheIdColumn` reads the Id column by header position)
- [x] AC6 -> `TestSystemUpgrade*Refresh*` on all three OSes
- [x] AC8 -> `TestRunUpgradesSystemPackagesOnAQuietDay` (current runs the step once; fast-forward, reinstall, dirty, offline and ahead do not; a failure is `upgrade-failed` with an error; a skip is `upgrade-skipped` with the reason and no claim the step converged, and a mutation that drops the skip branch fails it) and `TestUpdate_ConvergesAfterAFastForward` (a quiet day's report counts one step, runs no setup script, and records the fixture's skip as `upgrade-skipped`). On the owner's Mac, 2026-11-11, `DOTFILES_REPO_DIR=<scratch clone> go run ./cmd/dotf update --state-home <scratch>` -> exit 0, `[ OK ] system-upgrade 5.1s index refreshed (brew update); catalog entries up to date`, `last.json` status `current`
- [x] AC7 -> af71fcc7: `packages.json` `$comment`, ADR-036 amendment 2026-11-10; 02586d37: `catalog.go`, `system.go`, runbook

## Test status

- `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./...` -> ok; `go test ./...` -> every package ok; `golangci-lint run` -> `0 issues.`
- Manual smoke test (owner's Mac, 2026-11-10): plan, apply and re-plan of `--only system-upgrade`. Nothing in the catalog (or outside it) was outdated, so the apply path upgraded nothing. The upgrade-with-packages path is covered by the fake-runner tests only until the next outdated formula appears.
- `brew outdated --json=v2` shape checked against the real brew (`formulae`/`casks` arrays). The `apt list --upgradable` and `winget upgrade` listings are unit-tested against assumed formats; Linux and Windows acceptance is a real run after the release (`dotf vm windows`, a Linux box with `sudo -n`).
- No regressions in the existing suite: yes

## Decisions made during implementation

- Production runner `converge.ExecManager` returns stdout only (stderr appended to the error on failure): `tools.ExecRunner` merges stderr into stdout, and a brew `Warning:` or apt's "no stable CLI interface" notice would corrupt the parse.
- A failed winget id fails the step at Reconcile (Changes = the ones upgraded); Probe re-lists independently.
- Housekeeping failure is a note in Detail, not a step failure.
- Linux skips the whole step (refresh included) without root, since `apt-get update` needs root too.
- `needs_sudo` casks and brew-pinned formulae are excluded per entry and named, so Probe can converge.
- winget prints a fixed-width table, and a Name cell can contain text that looks like a catalog id. The id match therefore reads the Id column alone. Its start is taken from the header line, and columns are counted as display cells, the way winget pads them (`golang.org/x/text/width`): a CJK or fullwidth character fills two cells, so a wide name or a localized header does not shift the slice. The release needs this fix as well as #3287: it landed separately in PR 1b, after #3287 merged at 6debd6ae.
- A failed `system-upgrade` step on a quiet day exits non-zero under its own status, `upgrade-failed`. After a fast-forward the same step failure already fails the run as `converge-failed`, and the same failure should not be softer on a quiet day. The separate status lets doctor's Self-deploy line tell a failed package upgrade from a broken deploy. It is not a stuck status: it clears when the package upgrades. A refresh failure, or apt without `sudo -n`, is a skip inside the step and exits 0 (AC6, AC4). `dotf update` reads the step's entry from the run's report (`runConvergeReport`) and records `upgrade-skipped` with the reason, because converge's exit status alone cannot tell a skip from an upgrade (pr-agent on #3312). It is not a stuck status: an offline day clears by itself.
- The quiet-day run is `dotf converge --only system-upgrade`, not a full converge. A full converge on a day nothing moved would re-run the setup script daily, which is the cost the "ONLY when HEAD moved" rule exists to avoid.

## Closing verification (main at af73427a, after #3312, 2026-11-11)

- Every `features.json` gate (f1-f8) run from the repository root: exit 0; each entry's `evidence` names the tests that passed.
- `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./...` -> ok; `go test ./...` -> exit 0, 56 packages ok; `golangci-lint cache clean && golangci-lint run` -> `0 issues.`
- Linux and Windows acceptance against a real package manager remains a post-release run (see the smoke-test line above); the parsing for both is unit-tested against recorded listings.

## Review dispositions (review.md, nan/deepseek-v4-flash, PASS-WITH-GAPS)

| Finding | Disposition |
|---|---|
| Major, THEORETICAL: winget "nothing outdated" detected by English text | Ticketed #3343: needs a real Windows host to confirm the exit code before the fix. |
| Minor: Linux plan without root previews nothing, yet the runbook advises the preview | Applied: `guide-self-deploy-timer.md` scopes the preview to macOS and Windows and names `apt list --upgradable` for Linux. |
| Minor: a permanently skipping machine renders as a doctor pass | Already ticketed: #3321. |
| Minor, REAL: no cmd-level test for `upgrade-failed` on a quiet day | Ticketed #3344 (the fixture needs a per-OS stub manager; the wiring is generic). |
| Minor, REAL: no lesson for the #3295 Id-column defect | Applied: `docs/lessons/lesson-583-a-fixed-width-table-cell-can-impersonate-an-id.md`. |
| Post-release Linux and Windows acceptance | Stays the declared deferral; #3343 carries the Windows run. |

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-583-a-fixed-width-table-cell-can-impersonate-an-id.md (and lesson 580 from PR 2)
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-036-install-channels.md (amended 2026-11-10: unpinned system entries upgrade, pinned entries move by PR)
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: parsing a tool's human table by header-measured columns is recorded in lesson 583; it has not recurred in another project
