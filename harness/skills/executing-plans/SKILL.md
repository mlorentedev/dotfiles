---
generated: true
generated_from: 00_meta/skills/executing-plans/SKILL.md
generated_sha: 3556db7bdaa98026
id: executing-plans-skill
type: skill
status: active
created: '2026-05-30'
owner: manu
name: executing-plans
description: Use when you have a written implementation plan to execute, typically
  after /writing-plans produces a plan file.
keywords: [execute plan, ejecutar plan, run plan, plan execution]
paths: ['**/plan*.md', '**/implementation-plan*.md']
---
# Executing Plans

Load plan, review critically, execute in batches, report for review.

## Process

### 1. Load and Review Plan

1. Read plan file completely
2. Identify questions or concerns
3. If concerns: raise with user before starting
4. If clear: proceed to execution

### 2. Execute Batch (3 tasks default)

For each task:

1. Mark as in_progress
2. Follow steps exactly (plan has bite-sized steps)
3. Run verifications as specified
4. Mark as completed

### 3. Report

After batch complete:

- Show what was implemented
- Show verification output
- Say: "Ready for feedback."

### 4. Continue

Based on feedback:

- Apply changes if needed
- Execute next batch
- Repeat until complete

### 5. Complete

After all tasks verified:

- Run full test suite
- Verify all requirements met
- Report completion with evidence

## When to Stop

**STOP executing immediately when:**

- Hit blocker mid-batch (missing dependency, test fails, instruction unclear)
- Plan has critical gaps
- Verification fails repeatedly

Ask for clarification rather than guessing.

## Rules

- Review plan critically first
- Follow plan steps exactly
- Don't skip verifications
- Between batches: just report and wait
- Stop when blocked, don't guess
- Never start on main/master without explicit consent
