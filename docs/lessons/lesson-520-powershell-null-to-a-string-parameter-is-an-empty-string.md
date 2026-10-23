---
id: "lesson-520-powershell-null-to-a-string-parameter-is-an-empty-string"
type: lesson
status: active
title: "PowerShell's $null passed to a string parameter is an empty string"
created: "2026-10-22"
---

# PowerShell's $null passed to a string parameter is an empty string

## Context

The PowerShell `pi` and `opencode` wrappers in `powershell/profile.ps1` put secrets into the session while the
app runs, then clean up. The cleanup used to remove every name it had set, which deleted a key the user had
set themselves (#2909). The fix records each value first and restores it afterwards. A name that was absent
before has to be absent again.

## Problem

The first version of the restore called `[Environment]::SetEnvironmentVariable($n, $prior[$n])`, with `$null`
for a name that had been absent. That is the documented way to delete a variable from .NET. On macOS the new
behavioural test still failed: the variable was there afterwards, set to an empty string.

PowerShell converts `$null` to `""` when it binds an argument to a .NET parameter typed `string`, so .NET
never receives the null. .NET documents an empty value as a delete on Windows. On macOS the variable was
measured to stay, defined and empty. A check that reads `if ($env:X)` cannot tell the two apart, which is why the bug stayed
hidden.

The refactor also took the profile out of a static guard without any error. `tests/lib/only-scope-scan.py`
reads `--only` lists with a regex. Once the helper forwarded `--only ($Names -join ',')`, the regex no longer
matched. The scan still printed OK, now with 19 tokens instead of 24.

## Solution

An absent name is removed with `Remove-Item Env:$n`, and a value is set with `Set-Item Env:$n`. Neither goes
through a .NET string parameter. The behavioural test, `tests/profile-secret-wrappers.Tests.ps1`, runs the real
functions from the profile, which it reads with the PowerShell parser, and checks
`[Environment]::GetEnvironmentVariable` against `$null`. It is a Pester suite because CI runs every `*.Tests.ps1`
on Windows, where the wrappers run; a new bats file there needs a `ci.yml` edit to be picked up. The scanner
now reads the `-Names` list at each call site and fails on any `--only` argument it cannot parse, instead of
skipping it.

## Rule

To delete a variable from PowerShell, use `Remove-Item Env:X`. Never pass `$null` to a .NET method expecting a
string: PowerShell delivers `""`, and only Windows reads that as a delete. When a refactor changes the shape of
code a static guard scans, compare the count the guard reports before and after. A regex guard that stops
matching still reports OK.
