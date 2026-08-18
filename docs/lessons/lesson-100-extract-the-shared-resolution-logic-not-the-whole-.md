---
id: lesson-100-extract-the-shared-resolution-logic-not-the-whole-
type: lesson
status: active
created: "2026-06-16"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 100: Extract the shared resolution logic, not the whole caller — keep agent-specific detail in the hook

**Context**: MEMORY-002 pulled the vault→memory symlink target resolution out of `claude-session-start.sh` into a standalone `ensure-memory-symlink.sh`, so other agents (or `dotf init`) could reuse the linking mechanics without reimplementing it.

**Problem**: The naive extraction boundary is "move the whole function" — but the original function mixed two concerns: computing Claude's agent-specific encoded project key (`encode_project_path`) and the agnostic vault-source-to-target linking mechanics (resolve, link, idempotent no-op). Extracting the whole thing either drags Claude-specific encoding into a script every other agent must also call, or forces each agent to reimplement the encoding step redundantly.

**Solution**: Split at the seam: `encode_project_path` (agent-specific) stays in the Claude hook; the shared script receives the already-computed target and only does the generic resolve+link+safety-check. A future agent (or `dotf init`) supplies its own encoding scheme and calls the same shared script.

**Rule**: When extracting shared plumbing out of an agent-specific caller, extract only the pure "given X, do Y" mechanics — leave every agent-specific naming/encoding decision in the caller. The caller computes what makes it unique; the shared script does what would be identical no matter which caller invoked it.
