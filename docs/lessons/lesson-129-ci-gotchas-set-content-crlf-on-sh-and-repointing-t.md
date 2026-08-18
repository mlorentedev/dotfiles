---
id: lesson-129-ci-gotchas-set-content-crlf-on-sh-and-repointing-t
type: lesson
status: active
created: "2026-06-25"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 129: CI gotchas: Set-Content CRLF on .sh, and repointing tests creates duplicate names

**Context**: PR-C deleted files and edited tests via PowerShell Set-Content and bats edits.

**Problem**: (1) PowerShell `Set-Content` writes CRLF on Windows. Applied to scripts/test.sh (.gitattributes eol=lf) it CRLF'd the whole file -> shellcheck SC1017 on every line locally (git normalizes on commit, but the working copy + local checks break). (2) Repointing a removed test to an existing assertion ("dotfiles-sync.sh is executable") created a duplicate @test NAME within verify-setup.bats -> bats refuses to parse the file, failing the `test` AND `integration` jobs.

**Solution**: Stripped CR from test.sh (`sed -i 's/\r$//'`). Deleted the repointed tests outright.

**Rule**: Don't rewrite .sh files with PowerShell Set-Content (it injects CRLF); use an LF-preserving edit or strip \r after. When removing a feature, DELETE its tests -- don't repoint them to another assertion, or you risk a duplicate @test name (a per-file bats parse error).
