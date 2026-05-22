---
tags: [spec, tasks, refactor-004]
created: "2026-05-21"
---

# Tasks — REFACTOR-004-init-project-repo-wiring

## Setup

- [x] Branch created: `feat/REFACTOR-004-init-project-repo-wiring`
- [x] `proposal.md` complete; acceptance criteria observable
- [x] No unresolved open questions

## Implementation

- [x] Add `--skip-agents` / `--skip-standards` / `--skip-github` flag parser to `init-project.sh` (preserves existing positional `<project-name> <stack>` signature via `_POSITIONAL` shift)
- [x] Add wiring block after `.gitignore` creation: invoke `init-repo-{agents,standards,github-defaults}.sh` with non-fatal `|| log_error` catches; auto-skip github when no `origin` remote configured
- [x] Mirror in `init-project.ps1`: add `-SkipAgents` / `-SkipStandards` / `-SkipGithub` switches to `param()`; same wiring block before `# SUMMARY`, using `Write-Warn` for non-fatal failures and `Write-Info` for the no-origin auto-skip
- [x] Confirm ASCII-only encoding of `init-project.ps1` (PSScriptAnalyzer rule from prior incident)
- [x] Add 9 bats parity asserts in `tests/init-project.bats`: 3 flag-parser greps, 1 helper-presence assert, 1 origin-check assert, 1 non-fatal assert (Linux); 3 parity asserts (Windows)
- [x] `bash -n` + `shellcheck` clean on `init-project.sh`
- [x] Full bats suite: 774/774 (was 765 pre-PR)

## Closing

- [x] All 8 acceptance criteria covered by tests
- [x] No regressions in existing init-project bats tests (4/4 still pass; the 2 functional tests confirm `|| log_error ... continuing` handles missing-vault-template case correctly)
- [x] The 3 standalone `init-repo-*.{sh,ps1}` scripts are byte-identical to pre-PR state (zero changes to their files)
- [x] `verification.md` filled
- [x] PR opened referencing this spec folder

## Out-of-scope follow-ups

- AUDIT-005 still surfaces REFACTOR-005 (vault tooling unification), CHORE-001 (shell-profile.sh decision), POLISH-001 (utils.sh boilerplate extraction). All independent of this PR.
- Future enhancement: `init-project.{sh,ps1}` could surface a `--with-remote <origin-url>` flag that runs `git remote add origin ...` and then re-runs `init-repo-github-defaults` automatically. Out of scope here.
