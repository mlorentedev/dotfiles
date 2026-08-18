---
id: lesson-040-wide-try-catch-misclassifies-the-error-and-mislead
type: lesson
status: active
created: "2026-05-19"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 040: Wide try/catch misclassifies the error and misleads the next reader

**Context:** SDD-002 (PR #51) wrapped `ConvertFrom-Json -AsHashtable` in `Merge-ClaudeSettings` with `try { ... } catch { Write-Warn "Claude settings template is not valid JSON after placeholder substitution: $_"; return }`. Under Windows PowerShell 5.1 (the default `PowerShell` interpreter on Windows), `-AsHashtable` does not exist — it was added in PowerShell 7.0. The actual exception thrown is `ParameterBindingException` ("A parameter cannot be found that matches parameter name 'AsHashtable'"), NOT a JSON parse error.

**Problem:** The catch was wide and the user-facing message anchored on "not valid JSON". For a debugging human reading the warning, the natural next step is to inspect the template file for JSON syntax errors — which are nonexistent. The actual root cause (PS version mismatch) is buried in the trailing `$_` interpolation that most observers skip. This was the entire surface area of BUG-005: not the missing parameter, but the misleading log line that prevented someone from finding the missing parameter for hours.

**Solution:** Two layers: (1) at the FIX site, replaced the catch by an explicit version check at script entry that re-execs under pwsh (BUG-005 / PR #58). (2) At the PATTERN site, catches should be either narrow (`catch [System.Management.Automation.ParameterBindingException]` for the specific case) OR the user-facing message should NOT assert a cause it cannot verify (write "Claude settings merge failed" + the raw `$_` exception type, not "is not valid JSON"). Better still: use `Test-Json` to check structure before parsing, so a JSON failure is a separate code path from any other parse-pipeline failure.

**Rule:** A wide `catch { ... }` is fine for "swallow and continue"; it is NOT fine when paired with a user-facing message that asserts a cause. If the message names a cause, the catch must be narrow enough that the cause is the only possibility. Otherwise the error message becomes a lie that costs more debugging time than no message at all. Auxiliary lesson: when a script must run under a newer runtime (PS 7+, Python 3.12+, Node 22+), the cheapest portability fix is to detect-and-reexec at the front door (single point of policy) rather than to backfill compat into every helper (N points, easy to miss one).

**Tags:** `#error-handling` `#powershell` `#portability` `#log-quality` `#wide-catch-trap` `#auto-reexec`
