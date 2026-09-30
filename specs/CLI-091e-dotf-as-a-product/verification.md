---
tags: [spec, verification, templates]
created: "2026-09-29"
---

# Verification - CLI-091e-dotf-as-a-product

## Evidence

- [x] AC1 -> test `TestHelpTextHasNoInternalReferences` (49 failures before the rewrite, 0 after)
- [x] AC2 -> `cli/README.md` rewritten; the `features.json` f2 command checks the sections and the absence of ids

## Test status

- `cd cli && go build ./... && go vet ./... && go test ./...`: green. `GOOS=windows go vet ./...`: green. golangci-lint 2.12.2: 0 issues.
- No test or script matched the old help strings (searched `tests/`, `scripts/`, the setup scripts).

## Decisions made during implementation

- The examples used real spec ids (`AI-001-ollama-public`, `HARNESS-071-reviewer-pool`), which the guard flags. They now use a neutral `FEAT-001-dark-mode`, which shows the id format without pointing at this repository's history.
- Error messages keep their ADR references for now (see Out of scope).

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-091e-dotf-as-a-product/` -> `specs/archive/CLI-091e-dotf-as-a-product/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
