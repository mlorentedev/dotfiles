---
tags: [spec, verification, templates]
created: "2026-06-16"
---

# Verification - GUARD-001-memory-sink-precommit

> Status: **verifying** — #409 shipped AC1/AC2/AC3 (the dispatcher + chaining). This follow-up PR completes AC4 (gitignore), AC5 (idempotent `dotf doctor` install), AC6 (AGENTS.md rule) and closes #398.

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (reject in non-vault repo) -> bats `tests/guard-memory-sink.bats::"AC1: rejects MEMORY.md committed to a non-vault repo"` (+ `memory/` path + normal-file-allowed variants)
- [x] AC2 (allow in vault) -> bats `tests/guard-memory-sink.bats::"AC2: allows MEMORY.md inside the vault (sentinel .obsidian/)"` + `"AC2: allows MEMORY.md when repo root == VAULT_PATH"`
- [x] AC3 (chaining preserves local hook) -> bats `tests/guard-memory-sink.bats::"AC3: chains to repo-local .git/hooks/pre-commit (both run)"` (+ failure-propagation + no-local-hook variants)
- [x] AC4 (dotf init gitignore block) -> Go `initrepo.TestScaffoldGitignoreHasMemorySinkBlock` (scaffolded `.gitignore` ignores `MEMORY.md` + `memory/`)
- [x] AC5 (idempotent install) -> Go `doctor.TestCheckGuardHooks_*` (5 cases: missing-dispatcher FAIL · unset→FAIL in check / FIX-wires under `--fix` · already-wired no-op · unrelated preserved-not-clobbered)
- [x] AC6 (AGENTS.md single-sink) -> bats `tests/agents-md.bats::"AGENTS.md states the memory single-sink rule (GUARD-001, AC6)"`

## Test status

- AC1-3 suite (from #409): `bats tests/guard-memory-sink.bats` -> 8/8 ok
- AC4: `cd cli && go test ./internal/initrepo/` -> ok (`TestScaffoldGitignoreHasMemorySinkBlock` + existing scaffold tests)
- AC5: `cd cli && go test ./internal/doctor/` -> ok (5 new `TestCheckGuardHooks` cases); `go vet ./...` clean; full `go test ./...` -> all packages ok
- AC6: `bats tests/agents-md.bats` -> ok; `bats tests/compile-harness.bats` -> 0 failures (the AGENTS.md edit did not break the HARNESS SSOT/compiler)
- No regressions: yes — full CLI suite + the affected bats suites all green.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work.

- R1 (chaining): **DECIDED 2026-06-16 — Option A, multi-hook chaining dispatcher.** Global `core.hooksPath` (dotfiles `.gitconfig`) → tracked `git-hooks/` dir with one dispatcher per hook type (`pre-commit`/`commit-msg`/`prepare-commit-msg`/`pre-push`); each runs its global concern then execs the literal `<toplevel>/.git/hooks/<type>`. Per-repo layer installed as a stable `.git/hooks/<type>` calling `pre-commit run` (not `pre-commit install`, whose location the global hooksPath would hijack).
- R2 (vault detection): recommendation = sentinel (`.obsidian/` at repo root) + `$VAULT_PATH` fallback — confirm as built.
- AC5 (install, this PR): the global `core.hooksPath` is **never clobbered** when it already points elsewhere — a machine-wide setting has too much blast radius to overwrite silently. An unrelated value is a WARN (preserve + tell the human); only an unset one is wired, and only under `--fix`; an already-correct value is a no-op. Implemented as a `dotf doctor` check (Go, `System`-injected) rather than a bats end-to-end so CI never mutates a real `~/.gitconfig`.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? Likely yes — "per-repo hooks can't enforce a machine-wide invariant; `core.hooksPath` + a chaining dispatcher is the agnostic keystone" (the ts-bridge leak root cause).
- [ ] ADR-worthy decision for `docs/adr/adr-XXX.md`? Candidate — the single-sink enforcement mechanism (global hooksPath dispatcher) is an architecture choice; consider an ADR if R1's chaining design proves non-obvious.
- [ ] New pattern for `00_meta/patterns/`? Only if the global-dispatcher-with-chaining recurs beyond memory (e.g. reused for secret-scanning). Defer.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/GUARD-001-memory-sink-precommit/` -> `specs/archive/GUARD-001-memory-sink-precommit/`
- [ ] Backlog entry (issue #398) ticked / closed with PR link
- [ ] Promotions above executed (if any)
