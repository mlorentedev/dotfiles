---
id: "CLI-041-skill-deps-and-triggers-tasks"
type: spec-tasks
status: active
created: "2026-08-18"
template_version: "1.0"
---

# Tasks — CLI-041 Skill Dependency Resolution & Full Trigger Catalog

## Tasks

- [x] **T1: Schema & Skill Frontmatter Updates**
  - Add `requires` property to `harness/skill-frontmatter.schema.json`.
  - Declare `requires: [...]` in composite skills (`spec`, `executing-plans`, `architecture-session`, `adversarial-review`, `vault-doctor`, `project-maturation`, `writing-plans`, `test-driven-development`).
- [x] **T2: Transitive Dependency Resolver in Go**
  - Implement `ResolveDependencies(skills []string, deps map[string][]string) []string` in `cli/internal/harness/triggers.go`.
  - Add `LoadSkillDependencies(skillsDir string) (map[string][]string, error)`.
  - Integrate transitive resolution into `Suggest()` and `SuggestWithDeps()`.
  - Add unit tests in `cli/internal/harness/triggers_test.go`.
- [x] **T3: Expand `harness/triggers.json`**
  - Add triggers for `infrastructure-as-code`, `kubernetes-packaging`, `golang-engineering`, `security-and-quality-audit`, `architecture-and-adr`, `task-and-ticket-tracking`, `vault-and-knowledge-management`, `hardware-and-embedded-debug`, `plan-authoring-and-execution`.
  - Sync with `cli/internal/harness/triggers.json`.
- [x] **T4: Verification & Integration Tests**
  - Update and add test cases in `tests/harness-suggest.bats`.
  - Run `./scripts/compile-harness.sh --check` to verify schema validation and zero drift.
