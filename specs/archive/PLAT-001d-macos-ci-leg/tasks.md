---
tags: [spec, tasks, templates]
created: "2026-10-07"
---

# Tasks - PLAT-001d-macos-ci-leg

> Three PRs, in order, because each later one runs what the earlier ones fix: (1) the scripts, hooks and hermetic test fixes (AC4), with this spec; (2) the tag tier, its guard and `scripts/run-bats.sh`, with the Linux `test` job calling it (AC2, AC3, AC6); (3) the non-required `test-macos` job (AC5). Split from #2063, whose single diff was too large for PR-Agent to review. Verification: shell layer `shellcheck`, `bats --jobs 8 tests/*.bats`; Go layer `go build ./... && go vet ./... && GOOS=windows go vet ./... && GOOS=darwin go vet ./... && go test ./...` and the pinned `golangci-lint`.

## Setup

- [x] Branches created from main: `fix/macos-portable-shell`, `test/bats-os-tier`, `ci/macos-test-leg` (cut from `ci/macos-leg`, #2063)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

### Triage (AC4)

- [x] [AC4] Run the full suite on the Mac (88 failures of 1784); cluster by root cause, never by test
- [x] [AC4] Class (b), missing tool: PyYAML and Python 3.11+ for the suite's Python readers. Environment, not code; CI installs them
- [x] [AC4] Class (a), script bugs: BSD `wc` padding, `grep -P` swallowed, no `sha256sum`, `chmod --reference`, physical against logical paths in two git hooks, `pgrep -a` in the stray detector
- [x] [AC4] Class (b), non-hermetic tests: physical sandbox paths, GNU `sed`/`stat` in test helpers, an Apple-signed binary copied and killed, host agy state
- [x] [AC4] Class (c), Linux-only: the release-PR body step runs on ubuntu only, so its tests skip with the reason where `sed` is not GNU

### Tiering and runner (AC2, AC3, AC6)

- [x] [AC3] Failing test, then `tests/guard-bats-tags.bats`: exact tag spelling, known tags only, `test_tags` above an `@test`, non-empty tier
- [x] [AC2] Tag the OS-sensitive tests (`# bats file_tags=os-sensitive` / `test_tags`), each file justified in `verification.md`
- [x] [AC6] Failing tests, then `scripts/run-bats.sh` (`getconf` CPU count, GNU parallel preflight, `--expect-bash`, empty tag selection fails); `tests/run-bats.bats` for the broken environments, `tests/run-bats-real.bats` for the real tools

### Workflow (AC5)

- [x] [AC6] `ci.yml`: `test` calls `run-bats.sh`
- [x] [AC5] `ci.yml`: new non-required `test-macos` on the `code` filter
- [x] [AC5] Assert bash 3.2 and zsh in the job rather than assume them
- [x] [AC5] First green run of `test-macos` on the PR that adds the job (first measured on #2063, run 37590992834); the main-only full-suite step runs after merge (green on main, run 38014809265, 2026-10-10)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] Follow-ups ticketed: #2059 (comment), #2061, #2062
