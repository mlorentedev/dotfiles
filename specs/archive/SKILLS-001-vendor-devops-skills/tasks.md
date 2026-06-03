---
id: "SKILLS-001-vendor-devops-skills"
type: tasks
status: draft
created: "2026-06-03"
template_version: "1.0"
---

# SKILLS-001 — Tasks

## T1 — Audit (avoid duplication) ✅
- [x] Compare the 10 issue candidates against the 25 existing vault skills.
- [x] Drop 3 duplicates (architecture-patterns, skill-development, python-testing-patterns); collapse terraform 3→1.

## T2 — Source + license due-diligence ✅
- [x] Identify a permissive (MIT/Apache-2.0) source repo per skill; verify each by reading its LICENSE.
- [x] terraform=Apache-2.0, helm=MIT, mcp-builder=Apache-2.0, golang-pro=MIT, async-python-patterns=MIT.

## T3 — Vendor into the vault SSOT ✅
- [x] Write `00_meta/skills/<name>/SKILL.md` for all 5 (clean frontmatter name+description+source+license; body adapted; broken `references/` links neutralized; attribution footer).

## T4 — Refresh + validate ✅
- [x] `compile-harness.sh --refresh` → committed `harness/skills/` records (+ vault-sync drift-fix).
- [x] `compile-harness.sh --check` → exit 0, "no harness drift".

## T5 — Attribution ✅
- [x] `harness/skills/ATTRIBUTION.md` NOTICE table (source/license/copyright per skill).

## T6 — Commit + PR
- [ ] Vault: commit the 5 `00_meta/skills/*` sources to vault `master` (direct, per vault discipline).
- [ ] dotfiles: PR with the 6 new `harness/skills/` records + ATTRIBUTION.md + spec. `Closes #158`. No AI attribution; English; no phase refs.
- [ ] Verify CI green (test/lint/spec-gate/integration).

## T7 — Close-out
- [ ] Tick HARNESS-001 epic (#162) consumer-3 (#158) on merge.
- [ ] (optional) Deploy locally: `setup` to render the new skills into ~/.claude/skills, ~/.config/opencode/commands, etc.
