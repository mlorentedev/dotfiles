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
- winget prints a fixed-width table, and a Name cell can contain text that looks like a catalog id. The id match therefore reads the Id column alone. Its start is taken from the header line, and the columns are counted in runes, so a non-ASCII name does not shift the slice. The release needs this fix as well as #3287: it landed separately in PR 1b, after #3287 merged at 6debd6ae.
