---
id: "HARNESS-051-copilot-native-skills"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-05"
issue: "mlorentedev/dotfiles#753"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-051-copilot-native-skills

> **Naming**: file lives at `<repo>/specs/HARNESS-051-copilot-native-skills/proposal.md`. `HARNESS-051-copilot-native-skills` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #753: HARNESS-051: deploy skills through Copilot native discovery -->

The cross-agent pipeline already carries valuable reusable skills such as `handoff`, but Copilot receives only a generated name-and-description catalog in its instructions file. Current Copilot CLI and Copilot App discover complete personal Agent Skills from `~/.copilot/skills`, so the catalog-only target prevents `/skills` from listing or loading the actual workflow bodies and auxiliary resources. Issue [#753](https://github.com/mlorentedev/dotfiles/issues/753) tracks restoring native discovery without creating another deployment engine.

## What

After setup, every skill whose `targets[]` includes `copilot` is present as a regular directory at `~/.copilot/skills/<name>/`, including its rendered `SKILL.md` and auxiliary files. Copilot CLI and Copilot App can discover those skills natively; target filtering and generated-output pruning follow the same behavior already used for Claude, Agy, and Pi.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- Porting `compile-harness.sh` and the PowerShell deploy twin to `dotf harness`; that remains tracked by #495.
- Rewriting skill bodies or translating provider-specific `allowed-tools` metadata.
- Removing the generated Copilot catalog. It remains as a compatibility index while native discovery becomes the authoritative execution surface.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- **Resolved - user-managed skills:** stale-output pruning already requires the `generated: true` provenance marker, so unrelated personal skills under `~/.copilot/skills` remain untouched.
- **Resolved - cross-platform parity:** both deploy engines consume the same `skills.deploy[]` manifest matrix; adding one declarative target reaches Linux and Windows without changing either engine.
- **Resolved - Copilot support:** GitHub documents `~/.copilot/skills/<name>/SKILL.md` as the personal skill location for Copilot CLI and Copilot App.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] **AC1 - Native discovery:** `compile-harness.sh --deploy` writes Copilot-targeted records to `~/.copilot/skills/<name>/SKILL.md` as regular copies.
- [ ] **AC2 - Complete and filtered render:** auxiliary files are copied for Copilot, while a skill with `targets: [claude]` is absent from the Copilot directory.
- [ ] **AC3 - Safe convergence:** a stale generated Copilot skill is pruned, but an unmarked user-managed skill is preserved.
- [ ] **AC4 - Product recognition:** the installed Copilot CLI lists `handoff` after deployment into an isolated Copilot home.

## References

- Bitácora: [#753](https://github.com/mlorentedev/dotfiles/issues/753)
- Original pipeline: [#141](https://github.com/mlorentedev/dotfiles/issues/141), `specs/archive/SDD-008-skill-pipeline/`
- Future Go migration: [#495](https://github.com/mlorentedev/dotfiles/issues/495)
- GitHub documentation: `Adding agent skills for GitHub Copilot CLI`
