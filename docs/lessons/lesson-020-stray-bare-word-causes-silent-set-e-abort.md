---
id: lesson-020-stray-bare-word-causes-silent-set-e-abort
type: lesson
status: active
created: "2026-03-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 020: Stray bare word causes silent set -e abort

**Context**: `github-secrets-manager.sh` had an accidental bare word `tmp` on its own line, immediately before a valid `tmp=$(create_temp_file "ssh_key")` assignment.

**Problem**: Under `set -e`, bash interprets a bare word as a command to execute. `tmp` is not a valid command, so it exits non-zero, and `set -e` aborts the script immediately. The SSH_PRIVATE_KEY_BASE64 branch would always fail silently with no error message — the "command not found" error went to stderr but was easily missed.

**Solution**: Delete the stray bare-word line. The variable assignment on the next line was already correct.

**Rule**: Under `set -e`, any bare word on its own line is treated as a command. A stale variable name from a copy-paste or incomplete edit becomes an invisible script-killer. Always review diffs for orphaned identifiers — they compile silently but crash at runtime.
