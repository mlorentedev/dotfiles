---
tags: [spec, verification, templates]
created: "2026-11-11"
id: "CI-018-dotf-gate-and-fleet-lint"
type: spec-verification
status: active
owner: manu
---

# Verification - CI-018-dotf-gate-and-fleet-lint

## Evidence

- [x] AC1 (knowledge half) -> PR 1, `tests/check-knowledge-gate.bats` (45 cases through both gates)
- [x] AC2 -> PR 2a (parsers), PR 2b (`tests/check-spec-gate.bats` 30 and `tests/spec-gate-archive.bats` 35 through the parity shim), PR 3 (`tests/spec-gate-adjacency.bats` 15 through the shim, step summary compared)
- [x] AC3 (knowledge half) -> PR 1, the no-network trap in `_dotf_gate`
- [x] AC4 -> PR 4 (four rules, exceptions: `go test ./internal/fleetlint/`), PR 5a (the other four)
- [x] AC5 -> PR 5b (`go test ./internal/fleetlint/ ./internal/cmd/ -run LintFleet`, the runbook `docs/runbooks/guide-fleet-lint.md`), linked from ci-workflows ADR-001 decision 5 by mlorentedev/ci-workflows#99

## Test status

- PR 1: `bats tests/check-knowledge-gate.bats` -> 1..45, no failures
- PR 1: `go test ./...`, `go vet ./...`, `GOOS=windows go vet ./...`, `golangci-lint run` and the Windows lint -> clean
- PR 2b: `bats tests/check-spec-gate.bats tests/spec-gate-archive.bats` -> 1..65; mutations: a verdict message (24 red), the archived-at-head check removed (6 red)
- PR 1 mutations: a changed problem message (3 cases red), `escaped` inverted (3 cases red)
- Closing, 2026-11-12 on main at 5854b97c: `bats tests/check-knowledge-gate.bats` 1..45, `tests/check-spec-gate.bats` 1..30, `tests/spec-gate-archive.bats` 1..35, `tests/spec-gate-adjacency.bats` 1..15, no failures; `go test ./internal/fleetlint/` 18 tests ok; `go test ./internal/cmd/ -run LintFleet` 7 tests ok
- Adoption, 2026-11-12: the eight rules run against each default-branch tip of the owner's 14 other repositories gave 4 to 57 findings each; one issue per repository, in that repository (ci-workflows#8 lists them)
- PR 5b: `go test ./...`, both vets and both lints clean; mutations, each red: the stale-exception check, the unknown-rule check, the `.git` test in `Repos`, the `[OK]` line on an incomplete run, the checkout prefix, a failed checkout counted, the empty-fleet error

## Decisions made during implementation

- The parity harness compares output except on exit 2: the shell gate reports an unset variable, `dotf` an event with no pull request, and both mean the wiring is wrong.
- The tokenizer for paths is compiled POSIX (leftmost-longest), matching the shell gate's `grep -oE`.
- `--fleet` refuses `--default-branch`: the checkouts it reads do not share one default branch, so each is checked against its own origin/HEAD.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-590-git-archive-drops-what-export-ignore-names.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the decisions (logic in dotf, one lint per standard) are ci-workflows ADR-001's, which now links the runbook
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: the fleet standard's home is ci-workflows ADR-001, not a vault pattern

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CI-018-dotf-gate-and-fleet-lint/` -> `specs/archive/CI-018-dotf-gate-and-fleet-lint/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
