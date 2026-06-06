---
name: project-maturation
description: Use when a project needs a structured quality audit and improvement plan. Triggers include new projects needing hardening, repos with missing tests or CI, codebases before first release, or when technical debt has accumulated across multiple dimensions.
---

# Project Maturation

Structured audit and improvement cycle for codebases. Detects stack, scores maturity across dimensions, generates a prioritized plan, and executes phase by phase.

## Protocol

### Step 1 — Detect Stack

Scan the project root to identify:

| Signal | Stack indicator |
|--------|----------------|
| `go.mod` | Go |
| `pyproject.toml`, `setup.py`, `requirements.txt` | Python |
| `package.json` | Node.js / TypeScript |
| `Cargo.toml` | Rust |
| `pom.xml`, `build.gradle` | Java |
| `*.sh` + `*.bats` | Shell/POSIX |
| `.astro` files | Astro |
| `Makefile`, `Dockerfile` | Infrastructure |

Detect CI system: `.github/workflows/`, `.gitlab-ci.yml`, `Jenkinsfile`.
Detect build tool: Make, npm scripts, Poetry, Gradle, Cargo.

### Step 2 — Audit Dimensions

Score each dimension 0-3:

| Score | Meaning |
|-------|---------|
| 0 — Absent | Not present at all |
| 1 — Basic | Exists but incomplete or inconsistent |
| 2 — Solid | Covers happy paths, integrated into workflow |
| 3 — Exemplar | Comprehensive, automated, battle-tested |

#### Dimensions (adapt to detected stack)

| Dimension | What to check | Stack-specific examples |
|-----------|---------------|------------------------|
| **Tests** | Coverage, edge cases, integration | pytest/Go table-driven/bats/JUnit |
| **CI/CD** | Pipeline exists, jobs, triggers, caching | GitHub Actions/GitLab CI |
| **Types & Linting** | Type safety, static analysis | mypy/golangci-lint/eslint+tsc/shellcheck |
| **Error Handling** | Explicit errors, no silent failures | Go `if err != nil`/Python exceptions/Rust Result |
| **Docs** | README, API docs, inline | README quality, architecture docs |
| **Security** | Deps audit, secrets, auth | `npm audit`/`pip-audit`/Snyk/age encryption |
| **Architecture** | Separation of concerns, patterns | Layered, hexagonal, clean architecture |
| **Observability** | Logging, metrics, health checks | Structured logging, healthcheck endpoints |

### Step 3 — Score Report

Present a maturity matrix:

```
=== Project Maturity Audit ===
Stack: Python 3.12 + pytest + GitHub Actions
Project: youtube-toolkit

| Dimension      | Score | Notes                          |
|----------------|-------|--------------------------------|
| Tests          | 1     | 12 tests, no edge cases        |
| CI/CD          | 2     | Lint + test, no deploy         |
| Types/Linting  | 1     | Ruff configured, no mypy       |
| Error Handling | 2     | Try/except, some silent catches|
| Docs           | 1     | README only, no API docs       |
| Security       | 0     | No dependency audit            |
| Architecture   | 2     | Clean layering, some coupling  |
| Observability  | 0     | Print statements only          |

Overall: 9/24 (37%) — Basic maturity

Priority order: Security → Tests → Types → Observability → Docs
```

### Step 4 — Generate Plan

Create phased plan ordered by impact (what hurts most first):

**Priority heuristic:**
1. Security (score 0-1) — always first
2. Tests (score 0-1) — safety net before any refactoring
3. CI/CD (score 0-1) — automation before manual improvements
4. Types/Linting — catch bugs statically
5. Error Handling — resilience
6. Architecture — structural improvements
7. Docs — document what's stable
8. Observability — production readiness

Each phase follows the project's established workflow (TDD, atomic PRs, etc.).

**Plan format:**

```markdown
## Phase 1: Security (score 0 → 2)
- [ ] Add dependency audit to CI (pip-audit/npm audit)
- [ ] Scan for hardcoded secrets
- [ ] Add SECURITY.md with disclosure policy

## Phase 2: Tests (score 1 → 2)
- [ ] Add edge case tests for core functions
- [ ] Add integration tests for API layer
- [ ] Add coverage reporting to CI (target: 80%)
```

### Step 5 — Execute

Use `executing-plans` skill to implement phase by phase with checkpoints.

**Between phases:**
- Run full test suite — no regressions
- Update the maturity score
- Commit completed phase

### Step 6 — Crystallize

After completing all phases:
- Update vault `11-tasks.md` with completion status
- Log lessons learned to the repo's `docs/lessons.md` (project lessons live in the repo — see [[pattern-knowledge-placement]])
- If new patterns emerged, propose for `00_meta/patterns/`

## Rules

- **One phase at a time.** Don't mix security fixes with test improvements.
- **Verify before advancing.** Each phase must pass CI before starting the next.
- **Stack-aware.** Don't recommend mypy for a Go project or golangci-lint for Python.
- **Respect existing patterns.** If the project already uses a specific testing framework, don't switch it.
- **Atomic PRs.** Each phase may decompose into multiple PRs (~300 lines each).

## Pipeline

- Previous: Read project CLAUDE.md and vault context for existing standards
- Next: `/executing-plans` to implement each phase
