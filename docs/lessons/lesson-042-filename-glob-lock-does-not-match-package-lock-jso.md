---
id: lesson-042-filename-glob-lock-does-not-match-package-lock-jso
type: lesson
status: active
created: "2026-05-19"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 042: Filename glob *.lock does NOT match package-lock.json (basename matters)

**Context:** Building scripts/check-spec-gate.sh for SDD-003: the LOC-exclusion list needed to skip lockfiles. Initial regex used a single glob pattern `*.lock` in a bash `case` statement. A bats test for `package-lock.json` failed because the file ends in `.json`, not `.lock` — the npm convention puts the lock-marker in the middle of the name, not the suffix.</context>
<problem>The naive pattern `*.lock` only matches files whose filename ends in literal `.lock`. It matches `Cargo.lock`, `poetry.lock`, `Pipfile.lock`, `yarn.lock`, `Gemfile.lock` — all the suffix-style conventions. It does NOT match npm's `package-lock.json`, pnpm's `pnpm-lock.yaml`, or Go's `go.sum`. Lockfile filtering with `*.lock` alone produces silent false negatives — exactly the kind of bug that ships green and breaks in production.</problem>
<parameter name="solution">Use basename-aware matching with explicit literals for non-suffix conventions. Pattern from scripts/check-spec-gate.sh:

```bash
_excluded() {
    local path="$1"
    local base="${path##*/}"
    case "$path" in
        tests/*|specs/archive/*) return 0 ;;
        *generated*) return 0 ;;
    esac
    case "$base" in
        *.lock|*.lockb) return 0 ;;
        package-lock.json|pnpm-lock.yaml|go.sum) return 0 ;;
        .gitignore|CHANGELOG.md) return 0 ;;
    esac
    return 1
}
```

Two `case` blocks: first checks path prefixes (tests/, specs/archive/) and substring matches (*generated*); second checks basename for filename conventions. ${path##*/} is bash parameter expansion for basename — no external `basename` call needed.

General rule: when filtering filenames by convention, list the literal exceptions BEFORE relying on a suffix glob. Suffix globs are a lower bound, not a complete set.</parameter>
<parameter name="tags">["bash", "globs", "lockfiles", "ci", "spec-gate"]</parameter>
</invoke>
**Problem:** 
**Solution:**
