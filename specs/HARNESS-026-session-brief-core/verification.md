---
tags: [spec, verification, templates]
created: "2026-06-17"
---

# Verification - HARNESS-026-session-brief-core

## Evidence

Each acceptance criterion from `proposal.md`, mapped to its proof. Test runs below
are from `bats tests/session-brief.bats tests/session-start-config.bats tests/session-start-false-positives.bats` (34/34 ok, 0 not-ok) and `shellcheck`.

- [x] `scripts/session-brief.sh` exists, executable, POSIX `sh`, ShellCheck-clean -> `shellcheck scripts/session-brief.sh` (clean) + test `core: session-brief.sh exists and is executable`
- [x] `--format=stdout` emits the migrated signals as raw lines -> test `standalone --format=stdout emits a non-fenced brief` + `sb_*` unit tests (`sb_vault_detect`, `sb_specs`, `sb_vault_baseline`, `sb_vault_health`)
- [x] `--format=markdown` wraps the brief in one fenced block -> test `standalone --format=markdown fences the brief`
- [x] Unknown/empty `--format` exits non-zero with usage on stderr -> tests `standalone rejects an unknown format with usage`, `standalone rejects a missing format`
- [x] The adapter obtains the signals from the core, no longer computes them inline -> `claude-session-start.sh` sources the core; inline `find_vault_root`/`detect_repo_specs`/`check_vault_baseline`/vault-health block removed (185-line net reduction)
- [x] Claude's emitted `additionalContext` is byte-identical to pre-PR output -> test `byte-equivalence: refactor preserves output (3 CWD scenarios)` (compares POST vs `origin/main` across dotfiles / outside-vault / inside-vault CWDs)
- [x] New bats cover both `--format` modes + the emitters in isolation -> `tests/session-brief.bats` (16 tests, isolated HOME/vault fixtures)

## Test status

- Test suite: `bats tests/session-brief.bats tests/session-start-config.bats tests/session-start-false-positives.bats` -> 34 ok, 0 not-ok
- `shellcheck scripts/session-brief.sh` -> clean (default severity); `shellcheck scripts/claude-session-start.sh` -> clean at `--severity=error` (CI gate), the only delta vs origin/main is SC1090 on the new `source`, silenced with a `# shellcheck source=` directive; SC2317/SC2016 are pre-existing (present on origin/main).
- Manual smoke: `SESSION_BRIEF_CWD=~/Projects/knowledge ./scripts/session-brief.sh --format=markdown` -> fenced block with the live vault headline, health summary, and spec counts.
- No regressions: yes — the byte-equivalence test is the regression net for the adapter; the hermetic false-positives suite was updated to copy the core (now a required dependency of the hook).

## Decisions made during implementation

- **Sourceable-library core, not subprocess-per-signal.** ADR-023 asks for a standalone with `--format` modes, but the 4 migrated signals are interleaved with adapter-owned blocks (the headline is *prepended*; specs/health/baseline append at different positions). A single contiguous subprocess chunk would change byte order. Making the core sourceable lets the adapter call each `sb_*` emitter inline at its legacy position — byte-equivalent — while the standalone `--format` runner still serves file-based agents. Emitters use the `func() ( … )` subshell form (no `local`, no variable leakage) and emit content via `printf %s` args so vault text with `%`/`\` is literal.
- **Hermetic test now copies the core.** `session-brief.sh` is a required dependency of the hook, not an optional sibling, so the false-positives suite copies it alongside the hook; vault-health.sh stays absent so `sb_vault_health` takes its deterministic "not found" path.
- **Scope = full agent-independent cluster** (user decision): vault detection/headline + vault-health + spec counts + baseline. Hive detection and the Claude-path-coupled signals (memory-temperature, crystallize, `.claude.json`) deferred to follow-up slices.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? yes — the `$()`-strips-trailing-newline + sourceable-emitter technique for byte-equivalent strangler extraction of an interleaved script. (Capture on archive.)
- [ ] ADR-worthy? no — implements existing ADR-023.
- [ ] New pattern for `00_meta/patterns/`? no — single-repo technique for now; revisit if the same strangler shape recurs (e.g. the PS twin).

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-026-session-brief-core/` -> `specs/archive/HARNESS-026-session-brief-core/`
- [ ] Backlog entry (#405) ticked with PR link
- [ ] Promotions above executed (if any)
