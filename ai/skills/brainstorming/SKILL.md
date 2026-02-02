---
name: brainstorming
description: Use before any creative work - creating features, building components, adding functionality. Explores user intent, requirements, and design before implementation through collaborative dialogue.
---

# Brainstorming

Turn ideas into designs through collaborative dialogue before implementation.

## Process

### 1. Understand Context

- Check project state (files, docs, recent commits)
- Ask questions ONE at a time
- Prefer multiple choice when possible
- Focus on: purpose, constraints, success criteria

### 2. Explore Approaches

- Propose 2-3 approaches with trade-offs
- Lead with recommended option and reasoning
- Let user choose direction

### 3. Present Design

Once direction is clear:

- Present in sections (200-300 words each)
- Ask after each section: "Does this look right?"
- Cover: architecture, components, data flow, error handling, testing
- Iterate based on feedback

### 4. Document

Save validated design to `docs/plans/YYYY-MM-DD-<topic>-design.md`

### 5. Implementation Setup (if continuing)

- Ask: "Ready to set up for implementation?"
- Create isolated workspace with `using-git-worktrees` skill
- Create implementation plan with `writing-plans` skill

## Principles

| Principle | Application |
|-----------|-------------|
| One question at a time | Don't overwhelm with multiple questions |
| Multiple choice preferred | Easier to answer than open-ended |
| YAGNI ruthlessly | Remove unnecessary features from designs |
| Explore alternatives | Always propose 2-3 approaches |
| Incremental validation | Present design in sections, validate each |
