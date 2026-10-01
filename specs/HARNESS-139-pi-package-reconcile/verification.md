---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - HARNESS-139-pi-package-reconcile

## Evidence

Built in two PRs: #1754 (`e4b39f55`, PR-A, the reconciler) and #1755
(`4dad5ead`, PR-B, `retire` and `requires`). Each criterion's command is in
`features.json`; every Go command names its tests with `-run '^(...)$'`, and
`go test -v` confirms each named test ran (a `-run` that matches nothing exits
0, so the count was checked, not assumed).

- [x] AC1 -> `TestNewPlan`, `TestApplyConvergesAndASecondRunCallsNothing`
  (fake pi editing a real `settings.json`), `TestPiPackagesApplyRemovesThenInstalls`,
  `TestPiPackagesApplyDryRunCallsNothing`
- [x] AC2 -> `TestPiPackagesCheckExitsNonZeroNamingDrift`,
  `TestPiPackagesCheckRefusesAnUnreadableManifest`, `TestLoadManifestRefusesWhatItCannotRead`
- [x] AC3 -> `TestApplyLogsTimeAndFencesTheOutputOfAFailure`,
  `TestApplySlowSuccessIsFencedAndStillCounted`, `TestPiPackagesApplyFailureExitsNonZero`
- [x] AC4 -> `TestPiPackagesApplySkipIsFirstAndLoud`,
  `TestPiPackagesApplyWithoutPiWarnsAndExitsZero`, `TestPiPackagesApplyWithoutNpmWarnsAndExitsZero`
- [x] AC5 -> `TestApplyMovesRetiredPathsIntact`, `TestApplyNeverOverwritesAnArchive`
- [x] AC6 -> `TestPiPackageRequirements_*` (missing fails, resolved passes, the
  shipped manifest passes, an unreadable manifest warns)
- [x] AC7 -> `tests/pi-packages.bats`: "both twins reconcile through dotf pi
  packages apply", "neither twin carries the reconcile loop any more", "the CI pi
  filter covers the manifest and BOTH twins"; `ci.yml` filters `cli/internal/pi/**`
- [x] AC8 -> measured live on msi, 2026-09-30, below

## Test status

- Test suite: `cd cli && go build ./... && go vet ./... && go test ./...` -> every
  package ok; `GOOS=windows go vet ./internal/pi ./internal/cmd ./internal/doctor`
  ok; `golangci-lint run` -> 0 issues; `bats tests/pi-packages.bats` -> 14/14.
- Manual smoke test (AC8, msi, `dotf` 0.61.0):

```
$ jq '.packages' ~/.pi/agent/settings.json | grep -c pi-memory
0
$ test -e ~/.pi/agent/memory || echo absent; ls -d ~/.pi/agent/archive/memory-*
absent
/home/manu/.pi/agent/archive/memory-20260928
$ printf '{"type":"get_state"}\n' | pi --mode rpc --no-session 2>&1 | grep -c memory_search
0
$ dotf pi packages check; dotf pi packages apply --dry-run; dotf pi packages apply
pi packages already reconciled (10 declared, 0 changed)   (x3, each exit 0)
```

  The `memory_search` count is not vacuous: the same output carries
  `{"type":"response","command":"get_state","success":true,...}`, so pi started
  and answered.
- No regressions in existing test suite: yes.

## Decisions made during implementation

- The first real `apply` was not a separate, announced step. It ran inside the
  owner's `setup-linux.sh` on 2026-09-28, the date the archive directory is
  named after. No record shows peers were told first, so the AC8 task says so
  rather than claiming it; `check` and `--dry-run` are recorded here after the
  fact, at `changed=0`.
- Without `NAN_API_KEY` in its environment, pi starts with model `unknown` and
  warns that no `nan/*` pattern matches. That is the key's absence in an agent
  shell, not a reconcile defect: under `dotf secrets run --only NAN_API_KEY` the
  same probe loads `nan/deepseek-v4-flash` with no warning.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no - the one non-obvious point (a
  `go test -run` that matches nothing exits 0) is a check made here, already
  common practice in this repo's specs.
- [x] ADR-worthy decision? no - the declaration semantics were the owner's
  decision on epic #1625, recorded there.
- [x] New pattern candidate? no: the reconcile is specific to pi's package manager.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-139-pi-package-reconcile/` -> `specs/archive/HARNESS-139-pi-package-reconcile/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
