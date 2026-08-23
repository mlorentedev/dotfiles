---
tags: [spec, verification, templates]
created: "2026-08-23"
---

# Verification - CLI-042-dotf-agent-run

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] AC1 (JSON contract on stdout, logs on stderr) -> commit `<hash>` / test `<name>`
- [ ] AC2 (pool-unavailable advances the chain, task-failed does not) -> commit `<hash>` / test `<name>`
- [ ] AC3 (timeout kills the worker and frees the slot before reaping) -> commit `<hash>` / test `<name>`
- [ ] AC4 (fails closed on an unreadable counter and an unidentifiable machine) -> commit `<hash>` / test `<name>`
- [ ] AC5 (the top tier escalates, never degrades) -> commit `<hash>` / test `<name>`
- [ ] AC6 (the hive backend answers through NaN and reports pool + model) -> commit `<hash>` / test `<name>`
- [ ] AC7 (`dotf secrets run -- hive serve`; no credential in a deployed file) -> commit `<hash>` / test `<name>`
- [ ] AC8 (Ollama and OpenRouter fully removed from deployed config) -> commit `<hash>` / test `<name>`
- [ ] AC9 (`dotf doctor` fails a backend that can serve nothing) -> commit `<hash>` / test `<name>`

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

-
-

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-042-dotf-agent-run/` -> `specs/archive/CLI-042-dotf-agent-run/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
