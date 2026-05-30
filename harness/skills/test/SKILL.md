---
name: test
description: Use when creating tests for functions, classes, or modules. Covers edge cases, failure modes, boundary conditions, and proper mocking. Supports pytest (Python), testing package (Go), vitest/jest (JS/TS), and built-in test (Rust).
---

# Test Generation

Generate complete test file ready to run.

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
