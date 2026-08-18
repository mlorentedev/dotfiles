---
id: lesson-056-setup-time-mutations-to-repo-symlinked-files-creat
type: lesson
status: active
created: "2026-05-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 056: Setup-time mutations to repo-symlinked files create permanent drift false-positives

**Context:** BUG-024 (PR #93). After REFACTOR-001 audit chain closed earlier today, the next ./setup-linux.sh fresh run reported `[12/12] Repo ↔ Deploy-Dir Drift FAIL`. diff-check.sh (PR #10) showed 3 lines drifting in both .bashrc and .zshrc: opencode PATH, project-init alias, and ~/.dotfiles/scripts on PATH.</context>
<parameter name="problem">setup-linux.sh appended those 3 lines to ~/.bashrc and ~/.zshrc via `ensure_line_in_file` *after* symlinking them to the repo (L431-433 opencode PATH, L903-905 project-init alias, L922-924 scripts PATH). Because the rc files are symlinks into the deploy-dir, the appends always wrote through to the deploy-dir copies — making those copies diverge from the (clean) repo source on every fresh setup. The drift detector then flagged it on the very next CI run. The repo source rc files also lacked a trailing newline, which caused the first appended line to concatenate onto `fi`.</problem>
<parameter name="solution">Make the repo the SINGLE writer for any file the drift detector watches. Bake the 3 lines directly into the repo source .bashrc/.zshrc (with trailing newline), delete the 3 `ensure_line_in_file` blocks from setup-linux.sh. `ensure_line_in_file` remains a valid pattern for *external* rc files (files the dotfiles repo does NOT own), but NEVER for files the setup has already symlinked from the repo. Test invariant rewritten in tests/opencode.bats #5 to assert repo-as-SSOT and forbid the old `ensure_line_in_file` pattern for these specific lines. Sibling of `lesson_dotfiles_two_tier_deploy.md` — same family of two-tier-deploy bugs where a writer bypasses the SSOT.</solution>
<parameter name="tags">["drift-detection", "setup-idempotence", "two-tier-deploy", "BUG-024"]
**Problem:** 
**Solution:**
