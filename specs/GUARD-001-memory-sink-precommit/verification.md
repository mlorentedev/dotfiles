---
tags: [spec, verification, templates]
created: "2026-06-16"
---

# Verification - GUARD-001-memory-sink-precommit

> Status: **implementing** — this PR ships the AC1/AC2/AC3 mechanism + tests; AC4/AC5/AC6 are follow-up (same #398).

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (reject in non-vault repo) -> bats `tests/guard-memory-sink.bats::"AC1: rejects MEMORY.md committed to a non-vault repo"` (+ `memory/` path + normal-file-allowed variants)
- [x] AC2 (allow in vault) -> bats `tests/guard-memory-sink.bats::"AC2: allows MEMORY.md inside the vault (sentinel .obsidian/)"` + `"AC2: allows MEMORY.md when repo root == VAULT_PATH"`
- [x] AC3 (chaining preserves local hook) -> bats `tests/guard-memory-sink.bats::"AC3: chains to repo-local .git/hooks/pre-commit (both run)"` (+ failure-propagation + no-local-hook variants)
- [ ] AC4 (dotf init gitignore block) -> Go `initrepo.TestScaffoldGitignoreMemoryBlock` — **follow-up**
- [ ] AC5 (idempotent install) -> bats `guard-install.bats::idempotent + preserves existing hooksPath` — **follow-up**
- [ ] AC6 (AGENTS.md single-sink) -> grep assertion in CI / bats — **follow-up**

## Test status

- Test suite: `bats tests/guard-memory-sink.bats` -> 8/8 ok (AC1 reject ×2 + allow; AC2 vault sentinel + VAULT_PATH; AC3 chain runs + propagates failure + runs without local hook)
- Drift guard: `bats tests/architecture-md.bats` -> 5/5 ok (`git-hooks/` declared in `docs/architecture.md`)
- No regressions in existing test suite: yes (only the two affected guards run here; full suite runs in CI `test` job)

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
