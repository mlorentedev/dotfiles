---
id: "lesson-347-a-change-is-verified-by-every-test-that-references-the-file"
type: lesson
status: active
title: "A change is verified by every test that references the file, not by the file's own suite"
created: "2026-10-07"
---

# A change is verified by every test that references the file, not by the file's own suite

## Context
Two PRs in one week passed the local suites a builder would pick and failed CI on a bats file nobody ran. Each round cost about 15 minutes of CI plus a push (measured 2026-10-07, #2043):
- Moving mise/zoxide activation in `powershell/profile.ps1` from `Invoke-Expression` to `[ScriptBlock]::Create` broke `tests/setup-windows.bats:1152`, a parity test across shells that reads the profile.
- Earlier, moving `.zshrc` to `dotf deploy` broke `tests/profile-heal-ps1.bats:121`.

## The Trap
The suites run locally were the "obvious" ones, `setup-linux.bats` and `setup-windows.bats`, picked by the file's name or by the script that deploys it. A file in this repo is read by more tests than the one named after it: parity tests, heal tests and doctor tests each assert on a file they do not own. Which suite owns a file is a guess. Which suites mention it is a fact, and the guess missed it twice.

## The Solution
Before pushing a change to a file, list the bats files that mention it and run all of them:

```bash
git grep -l <path-or-basename> -- tests
~/.local/bin/bats $(git grep -l <path-or-basename> -- tests)
```

Search both the full path and the basename (`powershell/profile.ps1` and `profile.ps1`), since a test often builds the path from parts. A file with no hits is still run through the wider suite before the PR leaves draft; the grep sets the minimum, not the whole check.
