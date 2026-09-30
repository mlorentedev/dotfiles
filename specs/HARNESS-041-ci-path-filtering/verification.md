---
tags: [spec, verification, templates]
created: "2026-08-20"
---

# Verification - HARNESS-041-ci-path-filtering

## Evidence

- [x] Criterion 1 (SHA-pinned `dorny/paths-filter` in the `changes` job) -> `tests/ci-path-filtering.bats` "HARNESS-041: the changes job runs a SHA-pinned dorny/paths-filter"
- [x] Criterion 2 (filtered jobs need `changes`) -> "HARNESS-041: every filtered job needs the changes job". Mutation: `needs: []` on `test-windows` turns it red.
- [x] Criterion 3 (every step after checkout is guarded) -> "HARNESS-041: every step after checkout in a filtered job is guarded by a changes output". Mutation: deleting the `if:` of "Run bats test suite" turns it red.
- [x] Criterion 4 (suite passes, filter covers the tree) -> `bats tests/ci-path-filtering.bats`, 4/4 on 2026-09-30. Mutations: deleting `'secrets/**'` from the filter, or misspelling `'setup-windows.ps1'`, turns "every top-level entry is in the code filter or declared docs-only" red.

Round 1 ran the same four mutations against the old tests: all four stayed green.

## Test status

- `bats tests/ci-path-filtering.bats tests/workflow-timeouts.bats tests/workflow-job-names.bats` -> all pass
- No regressions: yes

## Decisions made during implementation

- **Step-level conditional guards**: jobs stay active and only their heavy steps skip, so every required check reports on every PR.
- **The filter is an allow-list, widened by additions** (round 1). A deny-list would fail open on one wrong negation. The docs-only side is declared in the test, with a reason per entry, so the tree is classified in full.

## Round-1 review dispositions (FAIL, `nan/mimo-v2.6-flash`)

| # | Finding | Disposition |
|---|---|---|
| 1 | Major: the AC2/AC3 tests survive their own mutations | apply: the tests parse the workflow per job and per step; mutations recorded above |
| 2 | Major: the `code` filter omits paths the suite reads | apply: added `secrets/**`, `ssh/**`, `systemd/**`, `forge/**`, `.claude/**`, `AGENTS.md`, `install.sh`, `env-contract.json`, `machine.json.example`, `session-start-config.json`, `.gitattributes`, `.gitignore`, `.pr_agent.toml`, `.geminiignore` and both `*.local.example`; a coverage test classifies every top-level entry. `docs/` and `specs/` stay docs-only by decision: archive PRs are the common case, and their guards run in pre-commit and on push. Lesson 322 |
| 3 | Minor: AC1 names `@v3`, HEAD runs a pinned `@v4` | apply: AC1 names a SHA-pinned action; test 1 asserts the pin |
| 4 | Minor: "3/3" miscount and an unmeasured "~3-5 seconds" | apply: 4/4, and the number is dropped from the proposal |
| 5 | Minor: ACs unticked, `status: implementing` | apply: ticked, `status: verifying` |
| 6 | Minor: `tasks.md` promises a `features.json` | apply: the promise is replaced by a line saying the criteria are verified by the named tests |
| 7 | Minor (theoretical): the retired-twin step runs on docs-only PRs | apply: guarded on `code`, since retiring a twin deletes a script the filter matches |
| 8 | Minor (theoretical): `pi-nan-package` skips at job level | decline: it is not a required check, and `tests/workflow-job-names.bats` flags a required job with a skipping `if:` |
| 9 | Minor: stray `diff.patch` at the root | defer: #1869 |

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: `docs/lessons/lesson-322-an-allow-list-filter-classifies-every-unlisted-path-as-safe.md`
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a CI filter's contents, within the existing workflow
- [x] New pattern candidate for `00_meta/patterns/`? no: one repo's workflow

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-041-ci-path-filtering/` -> `specs/archive/HARNESS-041-ci-path-filtering/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
