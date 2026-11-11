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

- [ ] AC1 (knowledge half) -> PR 1, `tests/check-knowledge-gate.bats` (45 cases through both gates)
- [ ] AC2 -> PR 2 and PR 3
- [ ] AC3 (knowledge half) -> PR 1, the no-network trap in `_dotf_gate`
- [ ] AC4 -> PR 4 and PR 5
- [ ] AC5 -> PR 5

## Test status

- PR 1: `bats tests/check-knowledge-gate.bats` -> 1..45, no failures
- PR 1: `go test ./...`, `go vet ./...`, `GOOS=windows go vet ./...`, `golangci-lint run` and the Windows lint -> clean
- PR 1 mutations: a changed problem message (3 cases red), `escaped` inverted (3 cases red)

## Decisions made during implementation

- The parity harness compares output except on exit 2: the shell gate reports an unset variable, `dotf` an event with no pull request, and both mean the wiring is wrong.
- The tokenizer for paths is compiled POSIX (leftmost-longest), matching the shell gate's `grep -oE`.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CI-018-dotf-gate-and-fleet-lint/` -> `specs/archive/CI-018-dotf-gate-and-fleet-lint/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
