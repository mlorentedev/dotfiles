---
id: lesson-067-verify-a-plan-s-file-map-against-the-code-before-e
type: lesson
status: active
created: "2026-05-30"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 067: Verify a plan's file-map against the code before editing load-bearing scripts

**Context:** SDD-008 skill pipeline (PR #179). The implementation plan specified which loops to remove from setup-linux.sh / setup-windows.ps1 to migrate skill deploy to render-at-deploy ("agy ~L428, claude ~L505").
**Problem:** The plan's file-map was incomplete/wrong: the loop that actually produced the symlinks AC1 forbids was the vault-skill symlink loop (setup-linux.sh:1087 / setup-windows.ps1:594), which the plan did not mention. There were 5 skill-deploy loops per OS, not 2. Editing only the loops the plan named would have left AC1 (zero symlinks) silently failing, and opencode deployed from a separate pipeline (skills-to-opencode.sh -> ai/opencode/commands) the plan also missed.
**Solution:** Before editing, grep the actual code for every site that touches the thing being migrated (here: every loop writing to ~/.claude/skills, ~/.config/opencode/commands, ~/.gemini/*, and every vault-symlink loop) and reconcile that against the plan. Treat a plan's line numbers / file list as a hypothesis to verify, not ground truth. A wrong file-map makes an acceptance criterion fail silently because the obvious edits look complete.
**Tags:** `#sdd` `#verification` `#deploy` `#shell`
