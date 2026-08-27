---
id: lesson-233-piping-a-single-element-array-into-convertto-json-un
type: lesson
status: active
created: "2026-08-27"
owner: manu
tags: [lesson, dotfiles, powershell]
---

# Lesson 233: Piping a single-element array into ConvertTo-Json unwraps it

**Context**: AI-032/#1247 — syncing `enabledModels` into a deployed `~/.pi/agent/settings.json` needed a way to compare two PowerShell arrays for exact (order-sensitive) equality, mirroring the Linux side's `jq -c '.enabledModels'` canonical-string comparison.

**Problem**: `Compare-Object` treats arrays as sets, so a differently-ordered array with the same elements reads as "no difference" — wrong for this comparison, where order matters. The natural fix is a canonical JSON string via `ConvertTo-Json -Compress`, but PowerShell's pipeline enumerates a collection element-by-element before the cmdlet's process block ever sees it: pipe an array with exactly one element into `ConvertTo-Json` and it renders the bare scalar, `"x"`, not `["x"]`. An array with two or more elements looks fine (the pipeline delivers them as repeated invocations that `ConvertTo-Json` re-collects into an array before serializing), which is exactly the trap — the bug is invisible until the collection this code handles happens to shrink to one item, and by then the code has shipped and been trusted.

**Solution**: Bind the array as a named parameter instead of piping it — `ConvertTo-Json -InputObject $array -Compress`. Parameter binding passes the whole array as one value; the pipeline's per-element enumeration never happens, so the output is `[...]` at any count, including zero and one.

**Rule**: Before piping a PowerShell collection into `ConvertTo-Json` (or any cmdlet whose behavior depends on "is this one object or a collection"), ask what happens at zero and one elements, not just at the size you're testing with today. If the count can vary, use `-InputObject $var`, never `$var | ConvertTo-Json`.
