---
id: lesson-121-a-thin-per-os-shim-is-still-a-twin-converge-to-dir
type: lesson
status: active
created: "2026-06-23"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 121: A thin per-OS shim is still a twin — converge to direct CLI invocation

**Context**: CLI-025 PR1 ported the SessionEnd hook (`session-handoff.{sh,ps1}`) to the Go `dotf mem session-end` noun. The spec's wording said the hook should become a "thin shim that `exec dotf mem …`".

**Problem**: A thin shim still ships a per-OS `.sh`/`.ps1` pair. It *miniaturizes* the cross-OS twin-drift the CLI convergence exists to eliminate rather than removing it — the disease is not the script's size, it's that it exists in plural per OS. Keeping a shim re-introduces, in the last mile, the exact maintenance burden the `dotf` binary was built to kill.

**Solution**: Wire the hook to invoke the binary directly — the Claude Code hook `command` is the single string `<abs dotf path> mem session-end`, identical on Windows and Unix because `dotf`/`dotf.exe` resolves the same subcommand. Delete both twins outright (no replacement). The one residual OS-variance — "is `dotf` on PATH / where the binary lives" — moves to the single layer that already owns it (env-contract + `dotf doctor`, ADR-025). Use the **absolute** binary path (not bare `dotf`) so the hook survives a broken profile PATH (#531).

**Rule**: When converging a shell-twin cluster to a `dotf` noun, delete the twins outright via direct binary invocation; never replace them with thin per-OS shims. Move the only residual OS-variance to the env-contract layer, never into a fallback inside the scripts you are deleting. If a spec says "thin shim", treat that wording as refinable — it predates the convergence clarity.
