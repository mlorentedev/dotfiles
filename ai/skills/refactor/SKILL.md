---
name: refactor
description: Refactor code for improved structure, readability, and maintainability. Use when code needs cleanup following SOLID/DRY/KISS principles, adding type hints, reducing complexity, improving naming, or modernizing to current language idioms. Preserves behavior.
---

# Refactor

Improve code structure without changing behavior.

## Constraints

| Rule | Limit |
|------|-------|
| Function length | < 40 lines |
| Cyclomatic complexity | < 10 |
| Nesting depth | < 4 levels |

## Actions

1. Break large functions into small, pure functions
2. Add type annotations (no `Any`, no `interface{}`)
3. Rename variables descriptively (no `x`, `data`, `temp`)
4. Extract magic numbers to constants
5. Use modern language idioms (Python 3.12+, Go 1.26+, ESNext)

## Output

Full refactored code only. No explanations.

For significant changes, add a comment header:

```python
# Refactored:
# - Extracted validation logic
# - Added type hints
# - Renamed variables
```
