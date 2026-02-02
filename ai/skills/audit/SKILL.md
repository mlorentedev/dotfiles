---
name: audit
description: Security audit and code review for vulnerabilities, performance issues, and bad practices. Use when reviewing code for security (SQL injection, XSS, hardcoded secrets, CSRF), performance (N+1 queries, memory leaks, blocking async), or code quality (complexity, error handling, type safety).
---

# Security Audit

Analyze code for vulnerabilities, performance issues, and bad practices.

## Checklist

| Category | Issues to Find |
|----------|----------------|
| Injection | SQL concatenation, XSS, command injection, path traversal |
| Secrets | Hardcoded credentials, API keys in code, .env committed |
| Auth | Missing validation, broken access control, CSRF |
| Performance | N+1 queries, unbounded loops, blocking in async |
| Resilience | Unhandled errors, missing timeouts, race conditions |
| Quality | Magic numbers, deep nesting, missing types, dead code |

## Output Format

```markdown
## Issues

### HIGH
- [file:line] Issue description
- [file:line] Issue description

### MEDIUM
- [file:line] Issue description

### LOW
- [file:line] Issue description

## Fixes

### [Issue name]
[Fixed code - no explanation]
```

Provide fixes for top 3 HIGH/MEDIUM issues. Code only, no explanations.
