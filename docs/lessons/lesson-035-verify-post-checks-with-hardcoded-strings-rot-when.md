---
id: lesson-035-verify-post-checks-with-hardcoded-strings-rot-when
type: lesson
status: active
created: "2026-05-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 035: Verify post-checks with hardcoded strings rot when the verified file is refactored

**Context:** setup-windows.ps1 had Select-String post-checks (lines 142 and 466) grepping for the literal "CORE PRINCIPLE" in deployed CLAUDE.md/GEMINI.md to confirm the AI-013 pointer-style refactor (2026-05-16) landed correctly. AI-013 actually replaced that content with pointers starting "First, read `AGENTS.md`" -- the string "CORE PRINCIPLE" no longer existed in any deployed file. Setup kept emitting two spurious [ERROR] lines on every run despite the actual Copy-Item succeeding. Discovered empirically on 2026-05-18 during WIN-003 validation re-run.</context>
<problem>Hardcoded verify strings create an invisible coupling between two files (the deploy source and the verifier). When the deploy source is refactored, the verifier silently lies: deploy succeeds (Copy-Item ran), post-check says "failed" (string not found), and the script appears partially broken. Same class as BUG-001 (copilot-instructions verify, fixed in PR #40 with the exact same pattern fix). Both bugs lived in main for weeks -- they were only caught by empirical re-run of setup on a clean machine where someone read the output carefully.</problem>
<solution>Use a durable marker tied to the *file format convention*, not arbitrary body content. For pointer-style files the convention is the first-line marker `'First, read \`AGENTS.md\`'`. Match it with `-SimpleMatch` (PowerShell Select-String) or `grep -F` (POSIX) to avoid regex interpretation of backticks. The marker survives content refactors because it IS the convention, not an arbitrary string in the content. setup-linux.sh already had this; setup-windows.ps1 lagged in 2 places (BUG-002, fixed PR #47). Lock the parity in tests/setup-windows.bats with cross-OS asserts so future drift fails CI not production.</solution>
<tags>["bash", "powershell", "verification", "setup-scripts", "refactor-drift", "parity", "silent-failure"]</tags>
</invoke>
**Problem:** 
**Solution:**
