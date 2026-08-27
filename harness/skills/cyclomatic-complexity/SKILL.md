---
generated: true
generated_from: 00_meta/skills/cyclomatic-complexity/SKILL.md
generated_sha: 8cacaae276dc5f65
id: cyclomatic-complexity-skill
type: skill
status: active
created: '2026-08-26'
owner: manu
name: cyclomatic-complexity
description: Use when refactoring deeply nested logic, reducing cyclomatic or cognitive
  complexity, breaking down god functions, or cleaning up heavily branched code. Also
  use when reviewing code quality or preparing PRs with complex control flow.
source: https://github.com/saurabhkumar8112/cyclomatic-complexity-skill
license: Apache-2.0
keywords: [cyclomatic complexity, cognitive complexity, refactor nesting, god function,
  reduce complexity, guard clauses, extract function, code smells, spaghetti code]
paths: ['**/*.py', '**/*.go', '**/*.ts', '**/*.tsx', '**/*.js', '**/*.jsx', '**/*.rs', '**/*.c', '**/*.cpp']
requires: [test]
---
# Cyclomatic Complexity & Refactoring

Refactor code to reduce cyclomatic complexity, eliminate deep nesting, and keep code human-maintainable and readable.

## Core Principle

Cyclomatic Complexity ($\text{CC}$) measures independent execution paths:
$$\text{CC} = \text{Decision Points} + 1$$

Decision points: `if`, `else if`/`elif`, `case`, loops (`for`, `while`), `catch`/`except`, ternary operators (`? :`), and boolean operators (`&&`, `||`, Python `and`, `or`) inside conditions.

### Threshold Brackets
- **1–5**: Low complexity. Clean, leave alone.
- **6–10**: Moderate complexity. Watch; refactor if touching anyway.
- **11–15**: High complexity. Refactor now.
- **16+**: Very high complexity. Must split into smaller units.

*Project-specific linter thresholds (`.eslintrc`, `golangci.yml`, `ruff.toml`) override these defaults.*

## Measurement Tools

Prefer deterministic AST tools when available in the environment:
- **Python**: `radon cc -s -a <path>` or `ruff check --select C901`
- **Go**: `gocyclo -over 10 <path>` or `golangci-lint run --enable cyclop,gocyclo`
- **TypeScript / JavaScript / React**: `npx --no-install eslint --rule 'complexity: ["error", 10]' <path>`
- **Polyglot / Other**: `lizard <path>`

*When no tool is available, count decision points manually per function and display the calculation.*

## Refactoring Hierarchy (Order of Preference)

Apply refactoring tactics in this strict sequence:

1. **Guard Clauses:** Invert conditions and return early to eliminate nested indentation blocks.
2. **Extract Function:** Isolate sub-tasks into single-responsibility functions named for *what* they do, not *how*.
3. **Lookup Table / Map:** Replace branching `switch` or `if/else if` chains with dictionary/map lookups.
4. **Named Predicates:** Replace compound boolean expressions (`if (a && (b || c) && !d)`) with self-documenting helper functions (`if (isEligible(...))`).
5. **Polymorphism / Strategy:** Dispatch based on type or strategy objects when a `switch-on-type` occurs in 2+ places.
6. **Flatten Loops:** Extract loop bodies or use early `continue` to avoid indentation inside loops.

## Hard Rules

- **Preserve Behavior:** Run existing tests before and after refactoring. If tests are missing, declare the gap, add baseline regression tests first, or refactor conservatively.
- **Never Game the Metric:** Do not pack multiple branches into dense one-liners (e.g., compound ternary expressions, nested list comprehensions). Complexity must move into well-named functions, not disappear into syntactic cleverness.
- **Protect Public APIs:** Preserve exported function signatures and interfaces.
- **Single Responsibility:** If an extracted function requires "and" in its name, split it further.

## Workflow

1. **Identify Hotspots:** Measure touched functions and rank by $\text{CC}$ descending.
2. **Report Baseline:** State initial complexity numbers before modifying code.
3. **Refactor Incrementally:** Target the highest-complexity function first using the hierarchy above.
4. **Re-measure & Verify:** Run test suite and re-calculate $\text{CC}$.
5. **Output Report:** Conclude with the before/after complexity table.

## Output Format

End refactoring operations with a concise summary:

```markdown
## Complexity Report

| Function | CC Before | CC After | Refactoring Applied |
|---|---|---|---|
| `<functionName>` | 14 | 4 | Extracted `validateHeader`, `resolveDiscount` |
| `<functionName2>` | 12 | 3 | Guard clauses + lookup table |

**Verification:** <Command output / test run results proving zero regressions>
```
