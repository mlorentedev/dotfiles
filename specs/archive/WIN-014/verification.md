---
tags: [spec, verification, templates]
created: "2026-09-28"
---

# Verification - WIN-014

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> `TestHarnessMirrorCmd_UsesExplicitRepoOutsideTheCheckout` and `TestHarnessMirrorCmd_ExplicitRepoWinsInsideAnotherRepository`
- [x] Criterion 2 -> `TestMirror_PreservesTheSourceMode`
- [x] Criterion 3 -> setup BATS cases named `passes its checkout explicitly and warns when dotf is unavailable`

## Test status

- Test suite: `go test ./internal/cmd ./internal/harness` -> pass on Windows.
- Static checks: `go vet ./internal/cmd ./internal/harness`, `gofmt -l`, PSScriptAnalyzer, and `git diff --check` -> pass.
- Setup contract: targeted WSL BATS cases for Linux and Windows -> 2 passed.
- Manual smoke test: a source-built `dotf harness mirror --help` lists `--repo string`.
- Full setup BATS run: 190 passed, 3 dependency skips, and 5 environment failures because WSL lacks `zsh`, `jq`, and `pwsh`; all WIN-014 cases passed.
- Linux mode test: committed for Linux CI; local WSL has no Go toolchain, while Windows cannot expose POSIX executable bits.
- No regressions in targeted suites: yes.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Explicit checkout selection wins over cwd/environment inference because setup
  already knows the checkout it is configuring.
- A byte-identical destination with the wrong mode is drift and is rewritten;
  idempotence requires both content and mode convergence.

## Review dispositions (round 1, PASS, `nan/deepseek-v4-flash`, 2026-09-30)

| # | Finding | Disposition |
|---|---|---|
| 1 | Minor (theoretical): the `--repo` tests use an empty manifest, so its target half is not exercised | defer: #1872 |
| 2 | Minor (theoretical): no command-level test for `--repo` at a checkout without a manifest | defer: #1872 |
| 3 | Minor: the Linux half of the setup bats case greps a string that predates the change | defer: #1872 |
| 4 | Minor: `features.json` is still `pending` | recorded below: the three commands were run, and `features.json` stays untouched because it is in the contract set |
| 5 | Minor (pre-existing): a manifest target with `..` escapes the deploy dir | defer: #1872, with its root cause |
| 6 | Question: `os.Chmod` on a filesystem without POSIX modes | no action: no deployment target on such a filesystem is named |

`features.json` commands run on 2026-09-30 at `902f64a`: `go test ./internal/cmd ./internal/harness` exit 0; `go test ./internal/harness` exit 0; `bats tests/setup-linux.bats tests/setup-windows.bats` exit 0, 195/195. The Windows half runs in CI (`test-windows`, green on #1806).

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? no: issue #1751, the spec, and focused regression tests already preserve the operational finding.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: this applies ADR-020's existing CLI ownership boundary.
- [x] New pattern candidate for `00_meta/patterns/`? no: explicit setup inputs and mode convergence are already covered by the setup idempotence pattern.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/WIN-014/` -> `specs/archive/WIN-014/`
- [x] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018): the archive PR carries `Closes #1751`
- [x] Promotions above executed (if any): none, all three answered no
