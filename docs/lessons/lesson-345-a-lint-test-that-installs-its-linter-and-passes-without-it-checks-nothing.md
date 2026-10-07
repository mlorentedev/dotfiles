---
id: "lesson-345-a-lint-test-that-installs-its-linter-and-passes-without-it-checks-nothing"
type: lesson
status: active
title: "A lint test that installs its linter and passes without it checks nothing"
created: "2026-10-07"
---

# A lint test that installs its linter and passes without it checks nothing

## Context
While verifying a CI cache (#2046, 2026-10-07), `test-windows`'s Windows bats subset took 425 s
instead of its usual ~77 s. The extra time was in two tests, `setup-windows.ps1 passes
PSScriptAnalyzer` (226 s) and `obs-cli.ps1 passes PSScriptAnalyzer` (155 s). On the two baseline
runs that day the first one took 22 s and 18 s.

## The Trap
Each test ran `Install-Module PSScriptAnalyzer -Force` against the PowerShell Gallery before
linting, so its duration was the Gallery's latency that hour. The worse half was the error path:

```powershell
try {
    Install-Module PSScriptAnalyzer -Force -Scope CurrentUser -ErrorAction SilentlyContinue
    $results = Invoke-ScriptAnalyzer ...
    if ($results) { exit 1 }
} catch {
    Write-Warning "PSScriptAnalyzer not available: $_"
    exit 0
}
```

If the module could not be installed or loaded, the test **passed**. A Gallery outage read as
"lint clean". The tests were also redundant: `lint-powershell` already ran
`Invoke-ScriptAnalyzer`, fail-closed and with the same settings file, over every `*.ps1` in the
repo. Two owners of one check meant the weaker one only added cost and false assurance.

## The Solution
- **One owner per check.** The three bats tests were deleted (#2065). `lint-powershell` is the
  only place PowerShell is linted, and a guard test fails if any `tests/*.bats` or
  `tests/*.Tests.ps1` runs PSScriptAnalyzer again.
- **The linter is a pin, not the Gallery's latest.** `PSSCRIPTANALYZER_VERSION` in
  `versions.conf`; installed with `-RequiredVersion` only when the image lacks it, imported with
  `-RequiredVersion`, and the step says which path it took.
- **A tool that cannot be fetched fails the job, naming the source.** Three attempts with
  backoff, then `::error::PSScriptAnalyzer <v> could not be installed from the PowerShell
  Gallery ... so no file was analyzed`. Pester got the same treatment in #2046 (#1605).

When a test provisions its own tool, read its `catch` before trusting its green: "skip" or
"pass" on a missing tool is the same failure dressed as a result that `.claude/CLAUDE.md` asks a
preflight for.
