---
tags: [spec, verification, templates]
created: "2026-06-16"
---

# Verification - GUARD-001-memory-sink-precommit

> Status: **draft** — this PR ships the spec (design). Evidence is filled during the implementation PR.

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] AC1 (reject in non-vault repo) -> bats `guard-memory.bats::rejects MEMORY.md in non-vault repo`
- [ ] AC2 (allow in vault) -> bats `guard-memory.bats::allows MEMORY.md in vault (sentinel)`
- [ ] AC3 (chaining preserves local hook) -> bats `guard-memory.bats::runs guard and local pre-commit`
- [ ] AC4 (dotf init gitignore block) -> Go `initrepo.TestScaffoldGitignoreMemoryBlock`
- [ ] AC5 (idempotent install) -> bats `guard-install.bats::idempotent + preserves existing hooksPath`
- [ ] AC6 (AGENTS.md single-sink) -> grep assertion in CI / bats

## Test status

- Test suite: `<command> -> <output / coverage %>` (filled at implementation)
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work.

- R1 (chaining): **DECIDED 2026-06-16 — Option A, multi-hook chaining dispatcher.** Global `core.hooksPath` (dotfiles `.gitconfig`) → tracked `git-hooks/` dir with one dispatcher per hook type (`pre-commit`/`commit-msg`/`prepare-commit-msg`/`pre-push`); each runs its global concern then execs the literal `<toplevel>/.git/hooks/<type>`. Per-repo layer installed as a stable `.git/hooks/<type>` calling `pre-commit run` (not `pre-commit install`, whose location the global hooksPath would hijack).
- R2 (vault detection): recommendation = sentinel (`.obsidian/` at repo root) + `$VAULT_PATH` fallback — confirm as built.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? Likely yes — "per-repo hooks can't enforce a machine-wide invariant; `core.hooksPath` + a chaining dispatcher is the agnostic keystone" (the ts-bridge leak root cause).
- [ ] ADR-worthy decision for `docs/adr/adr-XXX.md`? Candidate — the single-sink enforcement mechanism (global hooksPath dispatcher) is an architecture choice; consider an ADR if R1's chaining design proves non-obvious.
- [ ] New pattern for `00_meta/patterns/`? Only if the global-dispatcher-with-chaining recurs beyond memory (e.g. reused for secret-scanning). Defer.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/GUARD-001-memory-sink-precommit/` -> `specs/archive/GUARD-001-memory-sink-precommit/`
- [ ] Backlog entry (issue #398) ticked / closed with PR link
- [ ] Promotions above executed (if any)
