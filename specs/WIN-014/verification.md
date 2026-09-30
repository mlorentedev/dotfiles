---
tags: [spec, verification, templates]
created: "2026-09-28"
---

# Verification - WIN-014

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> `TestHarnessMirrorCmd_UsesExplicitRepoOutsideTheCheckout` and `TestHarnessMirrorCmd_ExplicitRepoWinsInsideAnotherRepository`
- [x] Criterion 2 -> `TestMirror_PreservesTheSourceMode` and
  `TestMirror_ReplacesReadOnlyDestinationOnWindows`
- [x] Criterion 3 -> setup BATS cases named `executes harness mirror with its checkout path`
  and `warns when dotf cannot mirror the harness` for both Linux and Windows

## Test status

- Windows regression: `go test ./internal/harness -run '^TestMirror_ReplacesReadOnlyDestinationOnWindows$' -count=1` -> pass.
- Test suite: `go test ./internal/cmd ./internal/harness` -> pass on Windows.
- Static checks: `go vet ./internal/cmd ./internal/harness`, `gofmt -l`, PSScriptAnalyzer, and `git diff --check` -> pass.
- Changed-code lint: `golangci-lint run --new-from-rev=main` -> 0 issues.
- Full Go suite: all packages pass except the pre-existing Windows symlink
  privilege failure in `internal/doctor` (`A required privilege is not held by
  the client`, tracked by #1804); the changed `internal/harness` package passes.
- Setup behavior: `bats --filter 'WIN-014' tests/setup-linux.bats tests/setup-windows.bats`
  -> 4 passed. Each test extracts and executes the real harness-mirror block;
  fake `dotf` captures the `--repo` argument, while an empty command path
  exercises the non-fatal warning branch.
- TDD red evidence: with temporary local mutations that replaced both checkout
  arguments and both warning strings, the same command failed all 4 cases.
  Restoring the production scripts returned the suite to 4 passing cases.
- Manual smoke test: a source-built `dotf harness mirror --help` lists `--repo string`.
- Full setup BATS run: 190 passed, 3 dependency skips, and 5 environment failures because WSL lacks `zsh`, `jq`, and `pwsh`; all WIN-014 cases passed.
- Linux mode test: committed for Linux CI; local WSL has no Go toolchain, while Windows cannot expose POSIX executable bits.
- No regressions in targeted suites: yes.

## Prior-review dispositions

- **AC3 setup coverage — resolved.** The former source-text-only assertions were
  replaced with focused execution of the Linux and Windows mirror blocks. The
  tests prove the checkout path reaches `dotf harness mirror --repo` and that a
  missing `dotf` emits the documented warning without running either full setup.
- **Review-base mis-scope — confirmed; fresh review required.**
  `review-request.json` names `b15ad970f98c1ec6de1a2e51defc944cbbb0280a`,
  which is 19 commits behind the PR merge base
  `d691f613a7f0f6d79870ab318853046696069078`. That expands review scope from 6
  branch files to 92 files and makes the existing request unsuitable as
  WIN-014 evidence. This branch does not redesign review-base selection. Before
  archive, regenerate the independent review request from the then-current
  `git merge-base origin/main HEAD` and require the resulting review to cover
  only the PR diff plus these remediation changes.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Explicit checkout selection wins over cwd/environment inference because setup
  already knows the checkout it is configuring.
- A byte-identical destination with the wrong mode is drift and is rewritten;
  idempotence requires both content and mode convergence.
- Windows refuses an atomic rename over an existing read-only file. The mirror
  clears that attribute only on the old destination immediately before rename;
  if installation fails it restores the old mode, and a successful replacement
  retains the source mode already applied to the temporary file.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? no: issue #1751, the spec, and focused regression tests already preserve the operational finding.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: this applies ADR-020's existing CLI ownership boundary.
- [x] New pattern candidate for `00_meta/patterns/`? no: explicit setup inputs and mode convergence are already covered by the setup idempotence pattern.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/WIN-014/` -> `specs/archive/WIN-014/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
