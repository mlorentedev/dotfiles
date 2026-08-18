---
id: lesson-159-a-guard-that-is-quiet-when-idle-and-quiet-when-bro
type: lesson
status: active
created: "2026-08-07"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 159: A guard that is quiet when idle and quiet when broken is not a guard

**Context**: `windows/hive-upgrade.ps1` runs every 15 minutes to upgrade `hive-vault`. Its step 0 deliberately returns early when there is nothing to do, so a healthy machine is not restarted 96 times a day just to discover it is already current. That early return was one condition covering three cases: no install found, PyPI unreachable, or already up to date. All three exited 0 with no output.

**Problem**: The uv-tool install disappeared from the maintainer's Windows box. From then on the first case fired on every tick: the script exited 0, printed nothing, and `Get-ScheduledTaskInfo` reported `LastTaskResult: 0`. It ran that way for roughly two months. In the same period the orphaned trampoline it should have flagged left the Hive MCP server unable to start, so Claude Code registered zero `vault_*` tools -- and the only automated thing watching that install was reporting success. Three other mechanisms failed the same way for the same reason: `setup-windows.ps1`'s activation gate, the MCP `prerequisite_command`, and the versioned-layout build all infer "is hive installed?" from `uv tool list`, so all four went silently inert together. Every one of them had been written to degrade quietly on purpose ("non-fatal", "warning", "exit 0"). Fault tolerance had become fault invisibility.

**Solution**: Split the guard so each outcome carries its own signal: already current stays silent and exits 0 (the intended common case, unchanged); PyPI unreachable prints and exits 0 (transient, and failing here would cry wolf on any laptop that closed its lid); no install found prints and exits **non-zero**. The non-zero exit is the load-bearing part -- Task Scheduler does not capture stdout, so `LastTaskResult` is the only signal visible without running the script by hand, and it is precisely the one that read green throughout. A test pins the branches apart (`! grep -qF '-not $installed -or'`) so a later tidy-up cannot re-merge them.

**Rule**: When a periodic job returns early to avoid doing unnecessary work, "nothing to do" and "I cannot do anything" must not share an exit code, because the health signal an operator actually reads is usually the exit code, not the log. Ask of any quiet guard: if the thing it protects were completely broken, would this look different? If not, the quiet is hiding the failure rather than avoiding noise -- and the more mechanisms that share the same "is it installed?" predicate, the more of them go dark at once when that predicate turns false.
