---
id: "CLI-041-skill-deps-and-triggers-verification"
type: spec-verification
status: active
created: "2026-08-18"
template_version: "1.0"
---

# Verification — CLI-041 Skill Dependency Resolution & Full Trigger Catalog

## Test Evidence

### Go Unit Tests
```bash
cd cli && go test ./... -count=1
```
- `github.com/mlorentedev/dotfiles/cli/internal/harness`: `ok` (0.008s) — including `TestResolveDependencies` and `TestLoadSkillDependencies`.
- All 18 CLI packages passing without error.

### Bats Integration Suites
```bash
bats tests/harness-suggest.bats
bats tests/dotf-search.bats
```
- `tests/harness-suggest.bats`: 6/6 tests passing (prompts, paths, JSON, terraform, helm, transitive skill dependencies).
- `tests/dotf-search.bats`: 3/3 tests passing.

### Schema & Compilation Check
```bash
./scripts/compile-harness.sh --check
```
- Validated all 37 skills frontmatters against `harness/skill-frontmatter.schema.json`.
- Zero harness drift.

### Interactive Live Runs
- `dotf harness suggest --prompt "we need to configure terraform for kubernetes"`: Resolves `terraform` + `helm`.
- `dotf harness suggest --prompt "crea una sesion de arquitectura"`: Resolves `architecture-session` and transitively includes `read-all-adrs`, `spec`, `adversarial-review`, `verification-before-completion`.
