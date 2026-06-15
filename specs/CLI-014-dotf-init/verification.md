---
tags: [spec, verification, templates]
created: "2026-06-14"
---

# Verification - CLI-014-dotf-init

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

- **Step 1 (package + embed + drift guard).** Templates are vendored **whole-file, byte-identical** to the vault SSOT (`embed` + `drift_test.go` does pure `bytes.Equal`), exactly like `cli/internal/spec`. Per-output transforms (snippet extraction, `$VAULT_PATH` stripping for #248) happen at *render* time in later steps, not in the embedded bytes — keeping the drift guard a clean equality check.
- **`init` parent is runnable via `RunE: cmd.Help`** (the same idiom `root.go` uses) so it is listed under "Available Commands" rather than demoted to cobra's "Additional help topics" — a non-runnable, childless command is treated as a help topic. Step 4 swaps this `RunE` for the orchestrator.
- **Vault-content drift flagged for Step 2 (not silently dropped):** the embedded `agents-spec-section.md` snippet (a) carries `$VAULT_PATH` literals — the #248 leak, stripped at render time per AC #1 — and (b) still references the deprecated `init-spec`/`archive-spec` shell scripts instead of `dotf spec`. (b) is stale *vault* content; updating the vault SSOT (then re-vendoring) is Step 2 work done via the isolated-worktree vault-commit discipline.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons.md`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-014-dotf-init/` -> `specs/archive/CLI-014-dotf-init/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
