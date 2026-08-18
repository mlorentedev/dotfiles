---
id: lesson-085-tests-aimed-at-a-runner-that-doesn-t-exist-yet-are
type: lesson
status: active
created: "2026-06-10"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 085: Tests aimed at a runner that doesn't exist yet are dead weight — and "if available" guards rot silently

**Context:** WIN-004 (PR #325) added the first `windows-latest` CI job, finally executing `setup-windows.ps1` end-to-end plus the Pester and PowerShell-bats suites. `tests/sdd-009-deploy-time-secrets.Tests.ps1` had declared "WIN-004 will pick this up in CI" in its header and sat unexecuted for two weeks; the bats PSScriptAnalyzer/syntax tests only ran where pwsh happened to exist.
**Problem:** Four consecutive live runs each surfaced a real latent bug that no installed box could reproduce: (1) `$tool.Version` under StrictMode killed installs on clean machines; (2) `DOTFILES_DIR`/`DOTFILES_REPO_DIR` defaults pointed at paths that don't exist on runners; (3) Pester `-Skip` conditions evaluate at discovery, before `BeforeAll` runs; (4) MSYS paths inside quoted pwsh `-Command` strings bypass Git Bash auto-conversion and resolve against the drive root (`D:\d\a\...`). Worse, the knowledge-crystallize analyzer test had escaped its variables (`'\$PS1_SCRIPT'`) and was analyzing an empty path — its catch-block `exit 0` made it pass everywhere, forever, without analyzing anything.
**Solution:** Land the runner with the first test that targets it, not after a backlog of "will be picked up later" suites — every deferred suite is unverified code wearing a green badge. Audit `catch { exit 0 }` / "if available" guards: a test that can't fail is documentation, not verification (the healthcheck variant that exits 1 in its catch is what exposed the path bug). For Git Bash → native pwsh boundaries, convert paths explicitly (`tests/winpath.bash`, `cygpath -w`) — auto-conversion only applies to plain arguments, never inside quoted command strings.
**Tags:** `#ci` `#windows` `#pester` `#bats` `#msys` `#silent-failure` `#dead-tests` `#verify-before-completion`
