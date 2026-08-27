---
tags: [spec, verification, templates]
created: "2026-08-27"
---

# Verification - HARNESS-086-cyclomatic-complexity-evals

## Verification Plan

1. Run `./scripts/compile-harness.sh --check` to assert zero harness drift and valid skill frontmatter.
2. Run `bats tests/triggers-registry.bats tests/skills-pipeline.bats` to verify trigger registry integrity and cross-agent deployment.
3. Run `go test ./...` in `cli/` to assert Go tests pass.

## Verification Evidence

- `bats tests/triggers-registry.bats tests/skills-pipeline.bats tests/compile-harness.bats tests/agents-md.bats`: 118 tests, 0 failures.
- `go test ./...`: all CLI packages passed (exit 0).
- `./scripts/compile-harness.sh --deploy`: successfully rendered to `.gemini/skills/`, `.gemini/prompts/`, `.pi/agent/skills/`, `.claude/skills/`, `.config/opencode/commands/`, and `.copilot/copilot-instructions.md`.
