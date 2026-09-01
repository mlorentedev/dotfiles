---
tags: [spec, verification]
created: "2026-08-28"
---

# Verification - CLI-064-doctor-profile-heal

## Evidence

Run on the Windows work box (go1.26, windows/amd64), 2026-08-28, worktree
`dotfiles-wt-doctor-cluster`, branch `fix/doctor-profile-heal`.

- [x] **AC1** → `TestCheckProfileFiles_DetectsBUG020Corruption` (healthy → PASS;
  1 MB + 1 byte → FAIL "larger than 1 MB"; two marker pairs → FAIL "marker
  pairs"; message names the heal path).
- [x] **AC2** → `TestCheckProfileFiles_HealPathFollowsTheContract` (`SCRIPTS_DIR`
  set → that dir; unset → the contract default; never `~\scripts` literal).
- [x] **AC3** → `TestCheckProfileFiles_FixRunsTheHealAndVerifiesByConsequence`
  (fake `System` records the `pwsh -NoProfile -File <heal>` invocation and
  rewrites the fixture; a heal that leaves the signal in place stays FAIL).
- [x] **AC4** → `scripts/profile-heal.ps1` threshold `-gt 1` (was `-gt 2`),
  comment records why the two must agree.
- [x] **AC5** → `TestCheckProfileFiles` (pre-existing) unchanged and green.

### Mutation check (revert the fix, watch the guard fail, restore)

`profileCorruption` forced to return nil (corruption never detected):
`--- FAIL: TestCheckProfileFiles_DetectsBUG020Corruption`,
`--- FAIL: TestCheckProfileFiles_FixRunsTheHealAndVerifiesByConsequence`.
Restored; both green.

## Test status

```
go build ./... && go vet ./... && GOOS=windows go vet ./... && GOOS=linux go vet ./... && go test ./...
ok  	github.com/mlorentedev/dotfiles/cli/internal/doctor
(all other packages ok)
golangci-lint run ./...   → 0 issues.   (2.12.2, the versions.conf pin)
```

- No regressions in the existing suite: yes.
- `profile-heal.ps1` was NOT run against the real profile on this box (the
  profile is healthy: one marker pair, ~7 KB); the shell-out is exercised
  through the seam in AC3.

## Decisions made during implementation

- **Detection is the defect, heal is convenience.** The corruption source is
  gone (BUG-022), so the FAIL is what matters; `--fix` is the retired
  `doctor.ps1 -Fix` behaviour restored on the doctor path.
- **Verify by consequence.** The heal script's exit code is not the evidence;
  the re-read profile without the two signals is.
- **Contract path, not literal.** WIN-013 (#1310) moves `~\scripts` to
  `~\.dotfiles\scripts`; a literal here would break the remedy the day it merges.
- **Thresholds agree.** The script used `>2` markers; doctor uses `>1` (the SSOT
  contains none, so one pair is healthy). The script was aligned rather than
  doctor loosened.

## Promotion candidates

- [ ] Lesson: no (the pattern — verify by consequence, contract over literal — is already lessons 235 and 240).
- [ ] ADR-worthy decision: no.

## Archive checklist

- [ ] `dotf spec review CLI-064-doctor-profile-heal` PASS
- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/CLI-064-doctor-profile-heal/`
- [ ] Bitácora #531 closed with the PR link
