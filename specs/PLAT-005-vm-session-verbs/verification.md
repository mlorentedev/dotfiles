---
tags: [spec, verification, templates]
created: "2026-10-22"
---

# Verification - PLAT-005-vm-session-verbs

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] Criterion 1 -> commit `<hash>` / test `<name>`
- [ ] Criterion 2 -> commit `<hash>` / test `<name>`
- [ ] Criterion 3 -> commit `<hash>` / test `<name>`

## Test status

- Test suite: `go -C cli test ./... -count=1` -> exit 0, 50 packages ok (Linux exec and cp, 2026-10-30).
- Manual smoke test, Linux exec and cp, Lima 2.2.1 on macOS, session `plat005-probe`:
  - AC2: `printf` with `a b`, `$HOME`, `;id` came back literal, `ok`, `exit: 0`; `exit 3` gave `ok`, `exit: 3`, process code 1.
  - AC2: `--script` ran from a temporary guest path and left none behind.
  - AC3: piped stdin reached the guest, on argv and on `--script`.
  - AC4: a file went host to guest to host byte-identical through `/tmp/a b`; `findmnt` in the guest listed no share.
  - AC7: the wire protocol ran on the guest with synthetic values only. The masking is proved by unit tests, never against a live store (lesson 261).
  - AC11: `--timeout 2s` on `sleep 20` gave `timeout`, process code 1, in 2068 ms, with no `sleep` left in the guest; `no-session` exit 3; a stopped guest `stale` exit 2.
  - Windows is not covered until PR 1c.
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
- [ ] Folder moved: `specs/PLAT-005-vm-session-verbs/` -> `specs/archive/PLAT-005-vm-session-verbs/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
