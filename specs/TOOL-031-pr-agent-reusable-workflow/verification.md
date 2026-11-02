---
tags: [spec, verification, templates]
created: "2026-10-21"
---

# Verification - TOOL-031-pr-agent-reusable-workflow

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] Criterion 1 -> commit `<hash>` / test `<name>`
- [ ] Criterion 2 -> commit `<hash>` / test `<name>`
- [ ] Criterion 3 -> commit `<hash>` / test `<name>`

- AC3, partly -> PR #3015, run 41232841929: `Build dotf`, the preflight, the draw, the guard probe and the final guard all ran from the PR and passed; the risk route drew Sonnet, which published. The push gate was skipped on `ready_for_review`. No weight changed in that PR, so the task's run on a weight change is still open.

- AC6, D's half (#3016) -> `cli/internal/prreview/{reviewable,filefilter}_test.go`, `cli/internal/cmd/pr_reviewable_test.go`; the cases of `tests/pr-agent-queue-skip.bats` and the matcher cases of `tests/pr-agent-config.bats` are ported there, and the bats file keeps the wiring.
  - Mutations: 46 of the verb and 5 of the workflow wiring, each applied alone, every one failing a test. The first pass left 4 alive: two test gaps closed (a failed read with a body on stdout, an unpaginated listing), one literal case added, and one equivalent mutation that showed the code was redundant, now deleted.
  - Differential, one-off, before the Python left: `translateGlob` against Python 3.13's `fnmatch.translate` on 240,000 random glob and path pairs. 239,844 agreed, none differed, 156 are globs RE2 refuses, which are dropped and only narrow.

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
- [ ] Folder moved: `specs/TOOL-031-pr-agent-reusable-workflow/` -> `specs/archive/TOOL-031-pr-agent-reusable-workflow/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
