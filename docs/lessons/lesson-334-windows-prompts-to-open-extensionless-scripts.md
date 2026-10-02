---
title: "Windows prompts to open extensionless bash scripts"
date: "2026-10-02"
---

# Windows prompts to open extensionless bash scripts

## Context
Cross-platform tools often ship bash scripts without a `.sh` extension (e.g., `bats`) mapped to `~/.local/bin/bats`. 

## The Trap
When invoking these tools directly from PowerShell or CMD on Windows (e.g. running `bats tests/`), the Windows Shell does not know it is a bash script. Lacking an extension match in `PATHEXT` or file associations, Windows will spawn a blocking GUI dialog asking the user: "How do you want to open this file?" This halts automated CLI sequences and AI tools.

## The Solution
Never rely on extensionless bash scripts being natively executable on Windows. When porting or using such commands locally, wrap them in a `.cmd` shim in the local bin path.
Example `bats.cmd`:
```cmd
@echo off
"C:\Program Files\Git\bin\bash.exe" "%~dp0bats" %*
```
This forces the explicit routing of the extensionless file to bash, preventing Windows from relying on native file associations.
