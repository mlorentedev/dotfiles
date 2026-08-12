---
generated: true
generated_from: 00_meta/skills/audit/SKILL.md
generated_sha: 4b77476729b7009a
id: audit-skill
type: skill
status: active
created: "2026-05-30"
owner: manu
name: audit
description: Use when reviewing code for security vulnerabilities (SQL injection, XSS, hardcoded secrets, CSRF), performance issues (N+1 queries, memory leaks, blocking async), or code quality concerns (complexity, error handling, type safety).
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
