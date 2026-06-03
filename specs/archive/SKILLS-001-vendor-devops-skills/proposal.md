---
id: "SKILLS-001-vendor-devops-skills"
type: spec
status: archived
created: "2026-06-03"
tags: [spec, proposal, harness-001, skills, skills-sh, vendoring, cross-agent]
template_version: "1.0"
---

# SKILLS-001-vendor-devops-skills

> **HARNESS-001 consumer 3 (GH [#158](https://github.com/mlorentedev/dotfiles/issues/158)).** Vendor a curated, license-clean set of DevOps/IaC skills from the skills.sh ecosystem into the vault skill SSOT, so they deploy cross-agent through the existing engine (SDD-008) — not a bespoke per-skill copy.

## Why

skills.sh is a package manager (Vercel) for agent skills (`SKILL.md` = frontmatter + markdown), distributed from GitHub repos via `npx skills add <owner/repo>`. Several skills map to the user's stack (Terraform/Helm/Kubernetes, Go, async Python, MCP authoring) and should be available to **every** agent (Claude/OpenCode/agy/Copilot), through the unified deploy engine.

The repo already has the engine (vault `00_meta/skills/` → `compile-harness.sh --refresh` → committed `harness/skills/` records → `--deploy` per agent). SKILLS-001 just adds source skills to the SSOT; they then flow through the pipeline with zero new machinery.

## What

Observable result:

1. **Five vendored skills** added to the vault SSOT (`00_meta/skills/<name>/SKILL.md`) and refreshed into committed `harness/skills/` records, deploying to all agents via the existing matrix:
   - `terraform` — antonbabenko/terraform-skill (**Apache-2.0**)
   - `helm` — laurigates/claude-plugins · helm-chart-development (**MIT**)
   - `mcp-builder` — anthropics/skills · mcp-builder (**Apache-2.0**, official Anthropic)
   - `golang-pro` — jeffallan/claude-skills (**MIT**)
   - `async-python-patterns` — wshobson/agents (**MIT**)
2. **License attribution** — `harness/skills/ATTRIBUTION.md` (NOTICE-equivalent) + a per-skill footer citing source/license/copyright. Only each `SKILL.md` is vendored; upstream `references/` deep-dives are left upstream (linked, not copied).
3. **Drift fixed in passing** — `--refresh` also generated the missing `vault-sync` record (it was in the vault SSOT but lacked a committed record). Folded in, noted.
4. **CI green** — `compile-harness.sh --check` validates the new records offline; bats suite passes.

## Audit (avoided duplication — issue requirement)

Of the issue's 10 candidates, **3 were dropped as duplicates** of existing skills (audit-first): `architecture-patterns` (≈ `architecture-session`), `skill-development` (≈ `creating-skills`), `python-testing-patterns` (≈ `test` / `test-driven-development`). The 3 terraform candidates (module-library / style-guide / test) collapse into one comprehensive `terraform` skill. `frontend-design` is already present as a plugin.

## Out of scope

- **Vendoring upstream `references/` deep-dive files.** Only `SKILL.md` per skill; the self-contained guidance is the deliverable, depth links to upstream.
- **Windows deploy parity validation.** The skill records are OS-agnostic markdown; `compile-harness.ps1 --deploy` renders them on Windows unchanged. Empirical Windows validation is batched into the dedicated Windows session.
- **`npx skills` runtime integration.** Rejected: it lives outside the vault→engine SSOT model (the issue explicitly wants engine deploy, not a bespoke per-machine npx step).

## Risks / open questions

- **🟡 License compliance in a public repo.** MIT + Apache-2.0 both allow redistribution with attribution; `ATTRIBUTION.md` + per-skill footer preserve copyright + license + source link. We do not copy full LICENSE texts (they live upstream) — acceptable for adapted docs; tighten by copying license files if the repo later needs strict NOTICE.
- **🟡 Upstream drift.** Vendored copies can age vs upstream. Mitigation: each record marks its source; re-vendor + `--refresh` to update. No auto-sync (would couple us to external repos).
- **🟢 No engine change.** Skills are auto-discovered from the vault `skills.vault_subpath`; no `manifest.json` edit needed.

## Acceptance criteria

- [ ] **AC1** — 5 skills exist in vault `00_meta/skills/` with valid `name`+`description` frontmatter (schema-passing).
- [ ] **AC2** — `compile-harness.sh --refresh` generates `harness/skills/<name>` records for all 5 (+ vault-sync drift-fix); committed.
- [ ] **AC3** — `compile-harness.sh --check` exits 0 ("no harness drift").
- [ ] **AC4** — `harness/skills/ATTRIBUTION.md` lists every vendored skill's source/license/copyright; each `SKILL.md` carries a footer citing it.
- [ ] **AC5** — Audit documented: 3 dropped duplicates + the terraform 3→1 collapse.
- [ ] **AC6** — Vault skill sources committed to vault `master`; harness records + spec + attribution shipped via dotfiles PR (closes #158).
- [ ] **AC7** — HARNESS-001 epic (#162) consumer-3 (#158) ticked on merge.

## References

- Issue: GH [#158](https://github.com/mlorentedev/dotfiles/issues/158); epic [#162](https://github.com/mlorentedev/dotfiles/issues/162).
- Engine: `scripts/compile-harness.sh`, `harness/manifest.json`, SDD-008 / ADR-013.
- skills.sh ecosystem: <https://skills.sh>, <https://skills-registry.dev>.
- Sources: antonbabenko/terraform-skill, laurigates/claude-plugins, anthropics/skills, jeffallan/claude-skills, wshobson/agents (see `harness/skills/ATTRIBUTION.md`).
