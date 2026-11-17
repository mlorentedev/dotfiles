---
spec: "TOOL-036-system-package-upgrade"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "98d9acf5a7cce28adff9fb01c1fa46c0b04ad5ae"
reviewer: "nan/deepseek-v4-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: TOOL-036-system-package-upgrade — the four spec commits in
`2102d867a55b309634d4a6a8e006c6050f6b97c1...HEAD` (`fd6bd4d6` converge step,
`952f4f60` winget Id column, `af73427a` quiet-day update, `98d9acf5` closing
verification). The rest of that range is other work that merged in between and was
read only where it interacts (converge runner, update record, doctor self-deploy).
**Sources**: `specs/TOOL-036-system-package-upgrade/{proposal,tasks,verification,features}.md`,
`cli/internal/converge/sysupgrade.go` + `sysupgrade_test.go`,
`cli/internal/cmd/{converge,update}.go`, `cli/internal/update/{update,record}.go`,
`packages.json`, `docs/adr/adr-036-install-channels.md`,
`docs/runbooks/{guide-self-deploy-timer,tool-installation}.md`.

### Spec and task alignment

- **AC1–AC6** are implemented in one reconciler (`systemUpgrade`) registered after `tools`
  in `converge.Registry`, exactly as `proposal.md` "What" describes: refresh → limited
  outdated listing → upgrade only catalog targets → housekeeping, with the refresh
  reported as info and never counted (`upgradeDetail`).
- **AC7** holds: `packages.json`'s "it is never upgraded" sentence is gone, replaced by the
  `system-upgrade` rule, and `adr-036` carries an amendment dated 2026-11-10 that also
  answers the ADR's own ban on unattended upgrades (pin vs no-pin).
- **AC8** holds in `update.Run`: `StatusCurrent` now calls `upgradeQuietDay`, which runs
  `dotf converge --only system-upgrade` and returns `upgrade-failed` (with the error) or
  `upgrade-skipped` (with the step's reason from the report, `systemUpgradeSkip`) or
  `current`. The pin branch precedes it, so a running dotf older than the checkout is
  reinstalled (and converges in full) instead of hitting `Select`'s "unknown reconciler".
- Every `[x]` in `tasks.md` has matching code and a named test, except where the findings
  below say otherwise. `verification.md` evidence reproduces: I ran the named tests and the
  package suites in this worktree (see "What I ran").
- `proposal.md` states Linux/Windows real acceptance is post-release by design; that is a
  declared deferral, not silent drift — but it is also why the Windows findings below are
  THEORETICAL rather than closed.

### What I ran (this worktree, `98d9acf5`)

- `cd cli && go build ./...` → exit 0.
- `go test -count=1 -run TestSystemUpgrade ./internal/converge/ -timeout 5m -v` → 16/16 PASS
  (including `…DarwinPlanListsOnlyOutdatedCatalogEntries`,
  `…DarwinApplyUpgradesCleansAndConverges`, `…OutsideCatalogIsNeverUpgraded`,
  `…LinuxWithoutSudoSkips`, `…LinuxWithSudoUpgradesCatalogPackages`,
  `…WindowsUpgradesPerIdAndIsolatesAFailure`, `…RefreshFailureSkipsWithTheManagersError`,
  `…WindowsMatchesOnlyTheIdColumn`, `…WindowsIdColumnCountsDisplayWidth`).
- `go test -count=1 ./internal/converge/ ./internal/update/ -timeout 8m` → both `ok`
  (no regression in the two touched packages).
- `go vet ./internal/converge/ ./internal/update/ ./internal/cmd/` → exit 0;
  `GOOS=windows go vet ./internal/converge/` → exit 0.
- Not run: `golangci-lint` (not installed in this worktree) and a real Linux/Windows
  manager run — marked UNVERIFIED where they matter.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | Resilience (Windows) | `wingetBehind` treats a non-zero `winget upgrade` exit as "nothing outdated" only when stdout contains the English sentence `no installed package`. winget localizes its console messages, so on a non-English Windows every steady-state run (nothing outdated — the normal state) becomes a step error: the daily `dotf update` exits non-zero with `upgrade-failed`, and `Probe` fails the same way after a successful upgrade. The language-independent signal is winget's exit code `0x8A150014` (`APPINSTALLER_CLI_ERROR_NO_APPLICATIONS_FOUND`), which the code ignores. | Code read of `cli/internal/converge/sysupgrade.go` `wingetBehind`; no Windows host in this worktree, not reproduced | `TestSystemUpgradeWindowsMatchesOnlyTheIdColumn` covers only `wingetIDs("No installed package …") == 0`, which bypasses the exit-status branch → **UNTESTED** for the branch itself | code + tests (name it `TestSystemUpgradeWindowsNoUpgradesIsNotAFailure`, asserting a `0x8A150014` exit with a translated/empty listing is not an error) |
| Minor | THEORETICAL | Spec drift / usability | `linux()` takes the sudo branch *before* the refresh and the listing, so `dotf converge --plan --only system-upgrade` on a Linux box without passwordless sudo prints only `needs root: …` — while `docs/runbooks/guide-self-deploy-timer.md` tells the owner to "Preview that first with `dotf converge --plan --only system-upgrade`". `apt list --upgradable` needs no root, so a plan could name what is outdated and skip only the refresh. `verification.md` records the whole-step skip as a deliberate decision, and AC4 does not ask for a plan, so this is a gap against the runbook's advice rather than against an AC. | Code read of `linux()`; runbook text above | `TestSystemUpgradeLinuxWithoutSudoSkips` (asserts the skip, not the plan preview) — the plan-without-root path is **UNTESTED** | code (list before the sudo gate under `dryRun`) or docs (scope the runbook's preview advice to darwin/Windows) |
| Minor | THEORETICAL | Handoff / observability | A permanently-skipping machine is rendered as a doctor **pass**. On Linux without `sudo -n` the step skips forever and records `upgrade-skipped`, which is deliberately not in `stuckStatuses`; `checkSelfDeployRun` therefore prints `pass("daily from …; last `dotf update` … (upgrade-skipped): …needs root…")`. The reason is visible in the message (the runbook's claim holds literally) but nothing escalates it, unlike the apt-sudo remedy that `System packages` keeps warning about until fixed. | Code read of `cli/internal/update/record.go` (`stuckStatuses`) and `cli/internal/doctor/checks_self_deploy.go:102` (`case r.Stuck()`) | `TestRunUpgradesSystemPackagesOnAQuietDay` proves the record, nothing proves the doctor rendering → **UNTESTED** | code (doctor) or docs |
| Minor | REAL | Tests | AC8's failure half is recorded but not asserted at the cmd layer: `cli/internal/cmd/update_test.go:121` asserts `last.json` ends with `upgrade-skipped`, and no cmd-level test asserts `upgrade-failed` reaches `last.json` *and* exits non-zero. The wiring is generic (`saveRecord` on any Outcome), so this is a coverage gap, not a defect. | `grep -n upgrade-failed cli/internal/cmd/*_test.go` → no hit; `TestRunUpgradesSystemPackagesOnAQuietDay` proves the status at the `update` package only | `TestUpdate_ConvergesAfterAFastForward` covers only the skip status → second half **UNTESTED** at the cmd layer | tests |
| Minor | REAL | Knowledge (Definition of Done) | No repo lesson was written for the #3295 defect this spec carries a follow-up PR for: a fixed-width table cell can impersonate an ID, and an ellipsis/`Name` truncation must not become a match. That is exactly `lesson-113`'s class (winget output shape assumed, not observed) yet it lives only in `verification.md`'s decisions and the code comment. `verification.md` is excluded from the staleness check, so this can be added without invalidating the verdict. | The range adds no `docs/lessons/*` entry naming the Id column; `git log` of the spec's commits shows only the four listed | n/a (docs) | docs (`docs/lessons/`) |

No Blocker found. I specifically tried, and could not make these fail:
- **Injection / privilege**: no shell is involved — `r.run(bin, argv...)` is `exec.Command` with an
  argv slice; package names come from the catalog, apt names must lack spaces/tabs, and
  `sudo -n` cannot prompt. `--only` selects by registry name, so a typo errors rather than
  passing silently.
- **Probe after a skip**: `runStep` returns before `Probe` when `res.Skip != ""`, so a linux
  or refresh skip cannot fail as a post-condition — the mechanism that would have made
  AC4/AC6 self-contradicting.
- **Pinned / out-of-catalog packages**: filtered twice (catalog targets in `outdated`, plus
  `pinned` in `brewBehind`), and `TestSystemUpgradeOutdatedPackageOutsideCatalogIsNeverUpgraded`
  tables it over all three OSes.
- **Second run**: housekeeping is gated on `len(due) > 0`, so `changed=0` is reachable; the
  darwin apply test asserts the refresh still runs and no upgrade/cleanup argv does.
- **AC8 re-entry**: fast-forward, reinstall, dirty, offline and ahead all assert
  `Upgrades == 0` in the table test.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Every AC is met and the negative paths (pinned, out-of-catalog, no sudo, per-id failure, failed refresh) are tested; the localized-winget path and the plan-without-root path are uncovered. |
| Verification       | B | `features.json` gates are reproducible and I reproduced the named tests, the package suites and both vets; Linux/Windows real acceptance is declared post-release, and `golangci-lint` was not available here. |
| Scope              | A | The four spec commits stay inside `proposal.md`; the wide diff range is other work merged in between and was not judged. |
| Reliability        | B | Refresh→skip, housekeeping failure→note, per-id isolation and never-prompting are all handled; the English-message dependence is the one silent failure mode left. |
| Maintainability    | B | Table-driven tests, sub-40-line functions in `sysupgrade.go`, comments explain why (stderr, display width, no `--greedy`); one 542-line file mixes runner, parsers and reporting but follows the package's existing shape. |
| Handoff-readiness  | B | ADR-036 amended, two runbooks and `packages.json` updated, `verification.md` records the decisions; the #3295 parsing defect got no lesson. |

### Verdict
PASS WITH GAPS

No Blocker and no REAL Major: the one Major is THEORETICAL (a localized Windows host, never
observed — the repo's own Windows acceptance is a post-release `dotf vm windows` run), so it is
tracked rather than blocking. No dimension grades C or D.

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is **closed** by this verdict —
do not edit it. Disposition these in `verification.md` (excluded from the staleness check) or
carry them into a follow-up ticket:

- **Major (code + tests)** — decide the winget "nothing outdated" signal by exit code
  (`0x8A150014`) instead of the English sentence, and add
  `TestSystemUpgradeWindowsNoUpgradesIsNotAFailure` covering a translated listing and that exit
  code. Until then a non-English Windows fails its daily update on every quiet day.
- **Minor (code or docs)** — either list outdated packages before the sudo gate under `dryRun`
  so the runbook's `dotf converge --plan --only system-upgrade` preview works on Linux without
  root, or scope that runbook sentence to the platforms where it does.
- **Minor (code or docs)** — decide whether `upgrade-skipped` on a machine that will never have
  `sudo -n` should stay a doctor pass; if it stays one, say so in
  `guide-self-deploy-timer.md` so the next reader does not read the pass as "upgrading".
- **Minor (tests)** — add the cmd-level `upgrade-failed` half of AC8: non-zero exit and
  `last.json` status on a quiet day whose step fails.
- **Minor (docs)** — write the repo lesson for #3295 (a fixed-width cell can impersonate an ID;
  truncation must not become a match), the class `lesson-113` already opened for winget output.
- Track the post-release acceptance that `proposal.md` declares: a real `apt-get` and `winget`
  run, which is the only thing that can turn the Major above from THEORETICAL into evidence.
