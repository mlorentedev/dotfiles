---
tags: [spec, verification, templates]
created: "2026-10-07"
---

# Verification - CI-012-dedupe-ci-tests

## Evidence

This spec tracks #2059 across several PRs; the entries below are the rows landed so far (N4, N5, N7).

- [x] AC1 -> `tests/dotf-bin-helper.bats` ("a build that fails fails, locally as well as in CI", "no toolchain in CI fails rather than skipping") and the guard "no bats file skips when a build fails". Observed on a tree with a syntax error in `cli/cmd/dotf/`: on `main`, `compile-harness-real.bats` reports `ok 1 ... # skip dotf failed to build` for all 9 cases; on this branch the same files report `not ok` with `go build ./cmd/dotf failed`.
- [x] AC2 -> `DOTF_BIN=/nonexistent bats tests/compile-harness-real.bats` fails with `DOTF_BIN is set to /nonexistent, which is not an executable file`; `DOTF_BIN=/usr/bin/true` is honoured (the case fails on its output, proving the build was not used).
- [x] AC3 -> mutation table in the PR body: 13 one-line production mutations, each turns its Go twin red (`TestAgentRun_*` in `cli/internal/cmd`).
- [x] AC4 -> `tests/dotf-bin-helper.bats` "ci: ..." cases pin the build-before-bats order and the junit/timing/upload flags; `bats --jobs 4 --no-parallelize-within-files --report-formatter junit --timing --output <dir>` writes a `report.xml` with a `time=` per file (checked locally). Whether the artifact appears is only observable on the first CI run of this PR.

## Test status

- `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` -> all packages ok; `golangci-lint run` (pinned 2.12.2) -> 0 issues.
- Full bats suite, branch vs a clean `origin/main` worktree on this Mac: 61 failing on `main`, the same 61 on the branch plus one that was mine (`stub-real-pairing` wanted an exemption for the suite that stubs `go`), fixed in this change. The 61 are macOS environment failures (`/private` path prefix, GNU userland, PyYAML), a separate piece of work.
- No regressions in existing test suite: yes (after the fix above).

## Decisions made during implementation

- `harness-suggest.bats` and `dotf-search.bats` were not trimmed. The audit marks them "probable"; `harness suggest` asserts the shipped trigger data and `search` the CLI wrapper's `--type`/`--json` output, and no Go test pins either. Deleting without a twin would lose coverage.
- Three of the eleven `dotf-agent-run.bats` cases had no Go twin (top tier through the command, saturated pool through the command, probe reaching a stub harness) and a fourth was weaker (the `exit` key). The twins were written first, and mutated red, rather than keeping the bats cases.
- A missing Go toolchain still skips on a developer machine: the file headers promise a shell-only checkout runs the rest of the suite. In CI it fails.
- The spec was scaffolded with `--over-wip-limit` (19 active against a limit of 10): the rows are independent of the other specs and a block in an unrelated one would have mislabelled both.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-343-a-build-that-skips-on-failure-turns-a-compile-error-into-a-green-job.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a test-harness rule with no architectural consequence
- [x] New pattern candidate for `00_meta/patterns/`? no: one project, one occurrence

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CI-012-dedupe-ci-tests/` -> `specs/archive/CI-012-dedupe-ci-tests/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
