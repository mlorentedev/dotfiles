---
description: Use when you have a spec or requirements for a multi-step task, before touching code. Use after brainstorming or requirements gathering to create an actionable implementation plan.
---

# Writing Plans

Create implementation plans assuming the engineer has zero codebase context. Document everything needed: files to touch, code, testing, verification steps.

## Task Granularity

Each step is one action (2-5 minutes):

- "Write the failing test" -- step
- "Run it to verify it fails" -- step
- "Implement minimal code to pass" -- step
- "Run tests to verify pass" -- step
- "Commit" -- step

Each task follows the **RED-GREEN-REFACTOR** cycle: write a failing test first, implement minimal code to pass, then refactor. See `test-driven-development` for the full discipline, anti-patterns, and rationalization counters.

## Plan Header Template

```markdown
# [Feature Name] Implementation Plan

**Goal:** [One sentence]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies]

---
```

## Task Template

```markdown
### Task N: [Component Name]

**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py`
- Test: `tests/exact/path/to/test.py`

**Step 1: Write failing test**

` ` `python
def test_specific_behavior():
    result = function(input)
    assert result == expected
` ` `

**Step 2: Run test to verify failure**

Run: `pytest tests/path/test.py::test_name -v`
Expected: FAIL with "function not defined"

**Step 3: Write minimal implementation**

` ` `python
def function(input):
    return expected
` ` `

**Step 4: Run test to verify pass**

Run: `pytest tests/path/test.py::test_name -v`
Expected: PASS

**Step 5: Commit**

` ` `bash
git add tests/path/test.py src/path/file.py
git commit -m "feat: add specific feature"
` ` `
```

## Rules

- Exact file paths always
- Complete code in plan (not "add validation")
- Exact commands with expected output
- DRY, YAGNI, TDD, frequent commits

## Pipeline

- Previous: Requirements, spec, or design document
- Next: `/executing-plans` to implement task-by-task
- Quality gate: `/verification-before-completion` before claiming done
