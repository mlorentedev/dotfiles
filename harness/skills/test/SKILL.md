---
generated: true
generated_from: 00_meta/skills/test/SKILL.md
generated_sha: 47192803c8d525d8
id: test-skill
type: skill
status: active
created: '2026-05-30'
owner: manu
name: test
description: Use when creating tests for functions, classes, or modules. Covers edge
  cases, failure modes, boundary conditions, and proper mocking. Supports pytest (Python),
  testing package (Go), vitest/jest (JS/TS), and built-in test (Rust).
keywords: [write test, unit test, integration test, pytest, go test, vitest, crear
    test]
paths: ['**/*test*', tests/**]
---
# Test Generation

Generate complete test file ready to run.

> **Boundary vs `test-driven-development`:** this skill *generates* tests for code that already exists (coverage after the fact). If you are *implementing* a feature or fixing a bug, use `test-driven-development` instead — it writes the failing test FIRST and drives the code. Same multi-language mechanics (below); different moment in the loop.

## Coverage Requirements

- **Edge cases:** Empty inputs, nil/null, boundary values, zero, negative
- **Failure modes:** Invalid inputs, network errors, timeouts, exceptions
- **Boundary conditions:** Off-by-one, max/min values, empty collections

Do NOT only test happy path.

## Patterns

| Language | Framework | Style |
|----------|-----------|-------|
| Python | pytest | `@pytest.mark.parametrize`, fixtures, `unittest.mock` |
| Go | testing | Table-driven with `t.Run`, `testify` for mocks |
| JS/TS | vitest | `describe`/`it`, `vi.mock` |
| Rust | built-in | `#[test]`, `#[should_panic]` |

## Structure (Python)

```python
import pytest
from unittest.mock import Mock, patch

@pytest.fixture
def mock_db():
    return Mock()

class TestUserService:
    @pytest.mark.parametrize("email,valid", [
        ("user@example.com", True),
        ("invalid", False),
        ("", False),
    ])
    def test_validate_email(self, email, valid):
        assert validate_email(email) == valid

    def test_create_user_duplicate_raises(self, mock_db):
        mock_db.exists.return_value = True
        with pytest.raises(DuplicateError):
            create_user(mock_db, "existing@example.com")
```

## Output

Complete test file with imports, fixtures, mocks, and all test cases.

---

## Mutation Testing (Anti-Vacuous Test Verification)

> Detailed doctrine in [[00_meta/patterns/pattern-mutation-testing|pattern-mutation-testing]].

Code coverage proves lines executed; **mutation testing proves the tests actually notice when behavior is broken**. Use mutation testing on agent-authored logic and security guards before trusting the test suite.

### Quick Invocations (Diff-Scoped)

* **Python (`mutmut`):**
  ```bash
  mutmut run --paths-to-mutate <changed_module.py>
  mutmut results
  mutmut show <mutant_id>
  ```
* **Go (`go-mutesting`):**
  ```bash
  go-mutesting ./pkg/<target>/...
  ```
* **TypeScript (`Stryker`):**
  ```bash
  npx stryker run --mutate '<changed_file.ts>'
  ```
* **Rust (`cargo-mutants`):**
  ```bash
  cargo mutants --file <changed_file.rs>
  ```

### Reading Surviving Mutants
A surviving mutant means the synthetic bug produced no test failures:
1. Identify the unasserted boundary or side-effect.
2. Add a targeted test case asserting that specific behavior.
3. Re-run the mutation command until the mutant is killed.

