---
tags: [spec, verification, templates]
created: "2026-10-14"
---

# Verification - CLI-119-lane-next

## Measurements

2026-10-14, macOS, herdr 0.9.3: `dotf lane exec --agent <a> --prompt-file p.md` typed into a fresh split pane by `herdr pane run`, prompt "Reply with exactly the word LANE-READY and do nothing else."

- claude (Sonnet 5.5): started interactive, answered `LANE-READY`, pane `agent: claude`.
- agy 1.3.5 (`agy -i`): started interactive, answered `LANE-READY`, pane `agent: agy`.
- pi: answered `LANE-READY` only after the launch carried its keys. The shell's `pi` is a function that wraps `dotf secrets run`, and exec skips functions, so the table launches `dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY -- pi` (same key list as the bash and PowerShell wrappers).

These prove the launch half only. The finisher -> successor chain (AC6, `RESULT <agent> PASS` lines) lands with `lane next`.

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] Criterion 1 -> commit `<hash>` / test `<name>`
- [ ] Criterion 2 -> commit `<hash>` / test `<name>`
- [ ] Criterion 3 -> commit `<hash>` / test `<name>`

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

-
-

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-119-lane-next/` -> `specs/archive/CLI-119-lane-next/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
