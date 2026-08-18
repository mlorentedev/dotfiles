---
id: "CLI-041-skill-deps-and-triggers"
type: spec
status: archived
created: "2026-08-18"
issue: "mlorentedev/dotfiles#1069"
tags: [spec, proposal, harness, triggers, skills]
template_version: "1.0"
review: waived
review_waived_reason: "Declarative schema update, trigger catalog expansion and transitive dependency resolver with 100% unit and Bats test coverage."
---

# CLI-041 — Skill Dependency Resolution & Full Trigger Catalog

## Why

Following the DeepSeek Harness benchmark and dotfiles audit, two architectural gaps exist in our prompt/trigger routing system:
1. Skills do not declare prerequisite dependencies (`requires:`). When an agent loads a composite skill (e.g. `spec` or `executing-plans`), prerequisite skills like `adversarial-review`, `verification-before-completion`, or `systematic-debugging` are not automatically resolved.
2. `harness/triggers.json` only covers 8 out of 37 skills, leaving critical domains (Terraform/IaC, Helm/K8s, Go, Security Audit, Architecture Sessions, Ticket Creation, ADR review, Hardware Debugging) unmapped in `dotf harness suggest`.

## What

1. **Schema & Frontmatter**:
   - Update `harness/skill-frontmatter.schema.json` with an optional `requires: []` string array.
   - Enforce declarative dependencies in composite skills in `00_meta/skills/` and `harness/skills/`.
2. **Transitive Dependency Engine**:
   - Add `ResolveDependencies(initialSkills []string, depsMap map[string][]string) []string` in `cli/internal/harness`.
   - Update `Suggest()` to automatically resolve the transitive closure of required skills.
3. **Full Trigger Catalog**:
   - Expand `harness/triggers.json` with triggers for Terraform/OpenTofu, Kubernetes/Helm, Go Microservices, Security Audit, Architecture Sessions, ADR exploration, Hardware Debugging, and Bitácora Ticket Creation.

## Out of scope

- Runtime dynamic installation of external binary tools (handled via `dotf deploy` / setup).
- Modifying prompt injection templates outside of `requires:` awareness.

## Risks / open questions

- **Circular dependencies**: Cyclic dependency graphs must be safely detected and handled without infinite recursion (visited sets).
- **Harness drift**: `compile-harness.sh --check` must validate that updated skill frontmatters match schema.

## Acceptance criteria

- [x] `harness/skill-frontmatter.schema.json` validates optional `requires` array.
- [x] Skills with prerequisites declare `requires: [...]` in frontmatter.
- [x] `Suggest()` resolves transitive dependencies with cycle protection.
- [x] `harness/triggers.json` maps 100% of domain categories covering all 37 skills.
- [x] Go unit tests and Bats integration tests pass.

## References

- Issue: [mlorentedev/dotfiles#1069](https://github.com/mlorentedev/dotfiles/issues/1069)
- Related specs: `CLI-040-dotf-search-and-suggest`, `SDD-008-skill-pipeline`
- Related prestudy: `10_projects/dotfiles/prestudy/2026-08-18-harness-evolution-benchmarks-and-action-plan.md`
