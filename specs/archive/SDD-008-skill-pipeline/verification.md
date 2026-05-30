---
tags: [spec, verification, sdd-008]
created: "2026-05-26"
updated: "2026-05-30"
---

# Verification - SDD-008-skill-pipeline

> **Complete — all phases done; PR open.** Design = option A (committed records + render-at-deploy). The vault `00_meta/skills/<name>/SKILL.md` is the SSOT; `compile-harness.sh --refresh` compiles each into a committed record under `harness/skills/`; `--deploy` renders each record to its per-agent `$HOME` path as a regular copy (never a symlink); `--check` validates records render offline. Machine-readable contract: `features.json` (8 entries, one per AC).

## Evidence (per AC)

- [x] **AC1** (zero symlinks in deployed paths) -> `bats tests/skills-pipeline.bats -f 'symlink'` + `bats tests/compile-harness.bats -f 'replaces a pre-existing vault symlink'`. --deploy `rm`s any pre-existing symlink before writing a regular copy; `healthcheck.sh` asserts deployed skill paths are symlink-free on-machine. Full setup exercised by the integration container.
- [x] **AC2** (single SSOT) -> 17 skills migrated to the vault in Phase 4 (22 there now); Phase 5 removed `ai/skills/*` (README stub remains), `scripts/skills-to-opencode.sh`, and `ai/opencode/commands/`. Verify: `test ! -d ai/skills || [ "$(ls -1 ai/skills | grep -vc '^README')" -eq 0 ]`.
- [x] **AC3** (`--check` detects drift, offline) -> `bats tests/compile-harness.bats -f 'AC3'` — clean tree passes offline; an invalid committed record fails. Wired into CI lint + `healthcheck.sh`.
- [x] **AC4** (provenance) -> `bats … -f 'renders records to .HOME with provenance'` — `generated_from` + `generated_sha:` on claude/opencode/agy outputs; prompt carries a leading `<!-- generated: … sha256:… -->`. **Reconciled:** provenance is added at --deploy (records stay verbatim); fields not a leading comment (a comment would break the YAML frontmatter agents parse).
- [x] **AC5** (frontmatter schema) -> `bats … -f 'schema:'` — missing `name` + unterminated frontmatter fail with file context. **Reconciled:** required = `name`+`description` (universal), not the proposal's `id`/`type`/`allowed-tools`.
- [x] **AC6** (per-skill targets[] + agy + copilot catalog) -> `bats … -f 'targets|catalog|agy'` + `bats tests/skills-pipeline.bats -f 'Claude-only'` — opt-out honored at deploy; agy native skill + flat prompt; copilot catalog injected into `$HOME`. Real records: claude 22, opencode/agy 17 (5 Claude-only opt out via `targets:[claude]`).
- [x] **AC7** (vault `pattern-cross-agent-skill-pipeline.md`) -> file exists, bidirectional cross-link with `pattern-spec-driven-development.md` (both `[[wikilinks]]` verified). Vault-dependent, so not run in dotfiles CI.
- [x] **AC8** (post-deploy smoke: /spec discoverable) -> `bats tests/skills-pipeline.bats -f 'smoke'` — renders the committed records to a throwaway HOME, asserts /spec at the claude/opencode/agy discovery paths. The integration container runs full `setup-linux.sh`.

## Test status (targeted; NOT `tests/*.bats` — see #167)

- `bats tests/compile-harness.bats` -> **27/27** (12 ENGINE-001 enforced + 15 SDD-008 records/deploy/check/schema/wiring).
- `bats tests/skills-pipeline.bats` -> **5/5** (AC8 smoke + AC1/AC6 on real records).
- `bats tests/setup-linux.bats` -> **78/78**; `bats tests/setup-windows.bats` -> **85/85** (static).
- `bats tests/healthcheck.bats` -> 22/22; `docs-drift` 5/5; `find-polluter` 4/4; `migrate-to-vault` 4/4; `opencode` 39/39.
- `shellcheck scripts/compile-harness.sh scripts/healthcheck.sh` clean; `bash -n` pass; PowerShell additions ASCII-only.
- Real-repo `compile-harness.sh --refresh` (vault present) -> 22 records; `--check` offline green.

## Decisions made during implementation

- **Option A (committed records + render-at-deploy)** — the record under `harness/skills/` is the one committed artifact; agent outputs render per-machine at deploy time, not committed. Deploy is a copy, never a symlink (ends BUG-100). `--refresh`/`--check` Linux-first; outputs not committed avoids a 4x diff and manual-edit drift.
- **Consume ENGINE-001** — skills are a manifest `skills` block with a `deploy[]` matrix + `catalog`, on the same `compile-harness.sh` `--refresh`/`--check` surface (new `--deploy` mode added). One engine.
- **opencode skip-list -> vault `targets[]`** — the five Claude Code-only skills (`creating-skills`, `crystallize`, `dispatching-parallel-agents`, `executing-plans`, `insights`) declare `targets:[claude]` in the vault, replacing the hard-coded skip-list in the removed `skills-to-opencode.sh`. Side effect (intended): agy no longer receives those five (it never could run them).
- **Windows = replicated deploy in PowerShell** — `Deploy-SkillRecords` ports `--deploy` (reads the same manifest + records). ENGINE-001's `--refresh`/`--check` stay Linux-only; Windows consumes committed records, the same model as the committed override blocks. Validated statically (bats); functional validation is Windows-empirical follow-up.
- **copilot catalog injected into `$HOME` only** (option-A pure) — `ai/copilot/copilot-instructions.md` + `.github/` carry empty markers (parity kept); the catalog is rendered into `~/.copilot/copilot-instructions.md` at deploy, never committed.
- **Provenance in frontmatter / schema required = name+description** (AC4/AC5 reconciliations, above).

## Promotion candidates

- [x] **New pattern** -> `00_meta/patterns/pattern-cross-agent-skill-pipeline.md` created (AC7), cross-linked with the SDD pattern.
- [ ] **Lesson?** Candidate — "reconcile a pre-existing spec to a newly-shipped engine before implementing" + "verify the plan's file map against the code (the symlink loop the plan missed)". Capture at archive.
- [ ] **ADR-worthy?** No new ADR — ADR-012 (deploy = copy + drift assertion) + ADR-013 (deploy engine) cover the substrate; this is an extension within them.

## Archive checklist (after merge)

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-008-skill-pipeline/` -> `specs/archive/SDD-008-skill-pipeline/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Lesson(s) above captured if promoted
