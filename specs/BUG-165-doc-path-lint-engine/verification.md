---
tags: [spec, verification, templates]
created: "2026-11-05"
id: "BUG-165-doc-path-lint-engine"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-165-doc-path-lint-engine

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> `TestCheckFiles*`, `TestDiscoverGovernedFiles`, `TestDiscoverFromLinkedWorktree`, and `TestLintDocPaths*` cover token outcomes, native glob behavior, linked worktrees, explicit paths and Git environment sanitization.
- [x] Criterion 2 -> Windows `TestCurrentRepository` measured 0.22 s in-process. After the same `version` probe used by `install.ps1`, two full CLI scans measured 485 ms and 473 ms with identical 14-line output (`sha256:98b1e2b30578c16ce3b735aa1408575b60d4cb3dd519cd44b7263c7cfd7cdcaf`).
- [x] Criterion 3 -> `TestLintDocPaths*`; the diff guard confirms `.pre-commit-config.yaml`, the shell guard and both BATS files remain unchanged in PR 1.
- [ ] Criterion 4 -> release-gated PR 2 remains unpublished until PR 1 is merged, released and installed canonically.
- [x] Criterion 5 -> `.github/workflows/cli.yml` has no event-level `paths:` filter, and `TestCurrentRepository` runs in the unfiltered Go test job.
- [x] Criterion 6 -> policy, scanner, discovery and command failure tests all exit 0; Git stderr is retained and filesystem traversal errors are no longer collapsed into empty globs.
- [x] Criterion 7 -> `TestLoadPolicy` rejects malformed, unknown and unsupported policy data; `TestCurrentPolicyLoads` pins the exact governed basenames.
- [x] Criterion 8 -> behavioral fixtures use `t.TempDir`; `TestCurrentRepositoryReadOnly` proves the live checkout is unchanged.

## Test status

- Focused engine suite: `go test ./internal/docpaths` -> PASS.
- Command regressions: `go test ./internal/cmd -run '^TestLintDocPaths'` -> PASS in 3.280 s.
- Vet: `go vet ./internal/docpaths ./internal/cmd` -> PASS.
- Formatting: `gofmt -l` over all changed Go files -> no output.
- Feature commands f1, f2, f3, f5, f6, f7 and f8 -> exit 0 with uncached runs; f4 is correctly pending behind the release gate.
- Windows smoke with incomplete `GIT_CONFIG_COUNT=1`: install-style `version` probe -> 10.092 s; following full scans -> 485 ms / 473 ms, deterministic 14-line output, exit 0.
- Fresh-build startup observation: endpoint security inspects each newly built unsigned EXE on first execution. `install.ps1` deliberately pays that one-time cost in its executable probe before placement, and its standalone path then runs the installed target through `converge`; hook use occurs after inspection rather than on the commit path.
- Lesson hygiene: `go run ./cmd/dotf lessons fmt --check --dir ../docs/lessons` -> `[OK] lessons are in their canonical form`.
- CI on `3ce302cc`: Linux/Windows tests passed; `cli-lint` found `filepath.Glob` and a dead helper, and hygiene found duplicate lesson 553. All three causes were removed in `17d943af`.
- Hosted CI on `17d943af`: Windows exposed an 8.3 working-directory alias (`RUNNER~1`) and macOS exposed `/var` versus `/private/var`; string-relative path resolution rejected files inside the checkout. `RepositoryLocation` now gets both Git's physical root and its working-directory prefix in one invocation, while tests compare checkout identity with `os.SameFile`. Focused Windows tests pass locally; replacement-head CI is pending push.
- Commit hook: historical bypasses are not counted as evidence. #3199 now permits one final reviewer/CI-fix commit after the evidence above; PR 1 makes no permanent hook configuration change.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Performance alone did not choose Go; an optimized shell could also be fast. ADR-020 ownership and retiring incident-encrusted dual-shell logic are the architectural reasons.
- `cli.yml` already runs Go tests on every pull request with no paths filter, so PR 1 needs no workflow edit for current-repository coverage.
- Explicit file arguments resolve from the caller's working directory by default and from `--repo` when supplied, then normalize to repository-relative report names.
- Git root discovery and tracked-file discovery share the same `GIT_CONFIG_*` sanitization boundary; failures retain Git's stderr instead of reporting a false "not inside a checkout" diagnosis.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: `docs/lessons/lesson-555-a-token-loop-is-a-process-loop-on-windows.md`
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-020 already assigns user-facing cross-OS tooling to the Go CLI.
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: the process-startup evidence is currently specific to this repository and workstation.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-165-doc-path-lint-engine/` -> `specs/archive/BUG-165-doc-path-lint-engine/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
