---
id: lesson-043-numeric-bats-threshold-drift-is-invisible-comment-
type: lesson
status: active
created: "2026-05-19"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 043: Numeric bats threshold drift is invisible — comment the bump inline

**Context:** AI-019 (model-tier policy) added a Model Tier subsection to ai/claude/CLAUDE.md, pushing the file from 70 to 78 lines. The existing bats assertion `wc -l < CLAUDE.md -le 70` started failing. Two options: compact existing content to fit under 70, or bump the threshold.</context>
<problem>If you silently bump a numeric threshold in a test (70 → 80 lines, 50 → 100 tests, etc.) without leaving a trace, the next contributor sees the new number and has no way to know whether (a) the threshold is calibrated to real constraints, or (b) it was raised to accommodate scope creep that should have been resisted. Threshold drift is invisible — every bump compounds; six months later, the assertion has become meaningless rubber. The classic "boiled frog" failure mode.</problem>
<parameter name="solution">When raising any numeric threshold in a bats test (or any CI assertion), add an inline comment in the SAME line/block stating: which spec/PR caused the bump, by how much, and the justification. Example from tests/opencode.bats line 148:

```bash
@test "ai/claude/CLAUDE.md is a pointer to AGENTS.md (≤ 80 lines)" {
    # Threshold bumped 70→80 in AI-019 (model-tier overlay added ~8 lines).
    # Future per-agent extensions should justify each bump in the spec.
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/claude/CLAUDE.md"
    [[ $(wc -l < "$DOTFILES_DIR/ai/claude/CLAUDE.md") -le 80 ]]
}
```

Now any future contributor reading the test sees: previous threshold, new threshold, what caused the change, and the implicit rule for next-bump justification. The comment also makes the audit trail visible to `git blame` so PR review can challenge unjustified bumps. Apply this to ALL thresholds — function-length linters, coverage minimums, perf budgets, file-size caps.</parameter>
<parameter name="tags">["testing", "bats", "thresholds", "code-review"]</parameter>
</invoke>
**Problem:** 
**Solution:**
