---
id: lesson-046-audit-all-call-sites-of-a-vulnerable-upstream-api-
type: lesson
status: active
created: "2026-05-20"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 046: Audit ALL call sites of a vulnerable upstream API when guarding one

**Context:** BUG-004 (PR #57, 2026-05-19) wrapped `claude plugin install` with snapshot/restore against upstream truncation bug `anthropics/claude-code#59870`. One week later (2026-05-20) the user reported `.claude.json` truncated AGAIN after a setup-windows.ps1 run. Diagnosis revealed BUG-004 missed ~9 sibling call sites that go through the same vulnerable deserialize-modify-serialize path: `claude mcp get`, `claude mcp add` (each ~9 times per setup), and `claude plugin list`. BUG-011 (PR #69) had to wrap all of them belatedly.</context>
<parameter name="problem">Fixing one symptom of a CLI-level vulnerability without enumerating siblings creates a guaranteed recurrence. The cost of audit-all-call-sites is small (~30 min grep + wrap); the cost of skipping it is a recurring incident class one week later.
**Problem:** 
**Solution:** When patching ANY guard around a vulnerable upstream API call, MUST in the same PR: (1) grep both setup scripts (sh + ps1) for every invocation of the vulnerable binary/subcommand family; (2) wrap each invocation with the same guard; (3) add bats parity asserts that fail CI if a future call site lands without the guard. The audit step is non-negotiable; "wrap one and move on" is a smell.
**Tags:** `#incident` `#guard-pattern` `#audit-discipline` `#claude-cli` `#cross-os-parity`
