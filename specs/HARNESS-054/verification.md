---
tags: [spec, verification, templates]
created: "2026-08-08"
---

# Verification - HARNESS-054

## Evidence

- [x] AC1 (every declared surface receives rules + presence) -> test `HARNESS-054: --deploy creates the doctrine file for a surface that has none`
- [x] AC2 (a missing region fails a test) -> test `HARNESS-054: every declared agent surface carries a generated region`, which enumerates the manifest's `presence[].file` and `doctrine.deploy[].file` rather than a hardcoded list
- [x] AC3 (user content preserved, idempotent) -> test `HARNESS-054: doctrine injection preserves user content and is idempotent`
- [x] AC4 (cap warning) -> test `HARNESS-054: a file over the platform's documented cap warns`
- [x] AC5 (shadow warning) -> test `HARNESS-054: a shadow file that wins at read time warns`
- [x] AC6 (rationale next to the manifest rows) -> `jq` assertion in `features.json` `HARNESS-054-f6`, run green

## Test status

- Test suite: `bats tests/compile-harness.bats` -> 44/44 pass (39 before this change, 5 added)
- Lint: `shellcheck scripts/compile-harness.sh` -> clean
- Manual smoke test: **not possible on this machine.** Neither Antigravity nor codex is installed, so the deploy path was exercised against a fake `$HOME` in the test fixture only. What was observed live: the same injection primitive already produces a valid region in four other harness instruction files.
- No regressions in existing test suite: yes — the 39 pre-existing cases pass unchanged.

## Decisions made during implementation

- **Compact payload rather than a full `AGENTS.md` copy.** Both platforms document a size limit the 21851-character file cannot satisfy (Antigravity 12000 characters per rules file; codex 32 KiB across the whole global+project chain). Shipping the full file would either be rejected outright or crowd out the repository's own `AGENTS.md`, which is the more specific file in the chain.
- **Injection, never overwrite.** `~/.gemini/GEMINI.md` is shared with the Gemini CLI, so the file may legitimately hold content we do not own.
- **No codex skills row.** No primary source documents its skill-discovery path; inferring one would reintroduce the guesswork this ticket exists to remove.
- **The curator persona lost its `targets[]` enumeration.** It listed four agents while depending on nothing agent-specific — the same "enumeration instead of universal" shape corrected in the skills fence earlier the same day.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — platform caps decide the render, and a deploy target that another tool also writes must be injected into, never overwritten.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — this implements ADR-027's model rather than changing it.
- [x] New pattern candidate for `00_meta/patterns/`? **update, not new** — `pattern-cross-agent-agent-pipeline.md` should record that a surface may receive a reduced render when a documented platform limit forbids the full one.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-054/` -> `specs/archive/HARNESS-054/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
