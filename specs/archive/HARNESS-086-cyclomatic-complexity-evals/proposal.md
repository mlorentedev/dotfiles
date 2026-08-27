---
id: "HARNESS-086-cyclomatic-complexity-evals"
type: spec
status: archived
created: "2026-08-27"
issue: "mlorentedev/dotfiles#1245"
review: waived
review_waived_reason: "Direct vendoring of upstream skill and benchmark evals dataset with passing Bats test suite (118 tests) and Go CLI unit tests."
tags: [spec, proposal, harness, skills, evals]
template_version: "1.0"
---

# HARNESS-086: Cyclomatic Complexity Skill and Harness Evaluation Benchmarks

## Why

AI-generated code frequently suffers from excessive branching, deep indentation, and monolithic functions. Integrating a vendor-agnostic cyclomatic complexity skill and a standardized harness evaluation dataset empowers agents and developers to systematically measure and refactor complex logic across all agent runtimes (Antigravity, Pi, Claude, OpenCode, Copilot) while providing reproducible benchmarks for harness quality.

## What

1. Adds `harness/skills/cyclomatic-complexity/` vendored under Apache-2.0 with proper attribution in `harness/skills/ATTRIBUTION.md`.
2. Registers the `code-complexity-and-refactor` trigger in `harness/triggers.json` and syncs with `cli/internal/harness/triggers.json`.
3. Adds `tests/evals/harness-evals.json` containing 10 curated test cases for evaluating agent harness design, auditing, and maintenance.
4. Compiles and verifies multi-agent skill deployment via `compile-harness.sh`.

## Out of scope

- Direct binary integration of AST parsers in Go CLI (delegates to host tools: radon, gocyclo, eslint).
- Modifying non-harness CLI subcommands.

## Risks / open questions

- *Risk:* Divergence between `harness/triggers.json` and embedded `cli/internal/harness/triggers.json`.
  *Mitigation:* Enforced by byte-identical assertion in `tests/triggers-registry.bats`.

## Acceptance criteria

- [x] `cyclomatic-complexity` skill record compiles cleanly with `skill-frontmatter.schema.json`.
- [x] `triggers.json` and embedded copy include `code-complexity-and-refactor` trigger.
- [x] `tests/evals/harness-evals.json` benchmark dataset is version-controlled.
- [x] All 118 Bats tests and Go unit tests pass with zero failures.

## References

- Issue: mlorentedev/dotfiles#1245
- Pattern: `00_meta/patterns/pattern-cross-agent-skill-pipeline.md`
- Pattern: `00_meta/patterns/pattern-llm-evals.md`

<!-- archived 2026-08-27 — PR: https://github.com/mlorentedev/dotfiles/pull/1246 -->
