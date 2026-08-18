---
id: lesson-142-bash-on-path-via-scoop-is-not-gnu-bash-it-silently
type: lesson
status: active
created: "2026-07-07"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 142: `bash` on PATH via scoop is not GNU Bash — it silently mis-executes bashisms

**Context**: Adding two new skills required running `scripts/compile-harness.sh --refresh` from a Windows machine to render them into the committed `harness/skills/` record. `bash` resolved on PATH to `C:\Users\mlorente\scoop\shims\bash.exe`.

**Problem**: Invoking the script through that shim failed immediately with `syntax error: bad substitution`, then a cascading `[ERROR] manifest not found: C://harness/manifest.json` — a path that could not have come from the script's own real repo-root resolution logic. `bash --version` returned `bad option '--version'`, and `$HOME` resolved to the malformed `C:Usersmlorente` (no separators). All three symptoms point the same way: scoop's `bash` shim is not GNU Bash — it is a minimal shell (BusyBox `ash` or equivalent) that accepts the name `bash` but rejects real bash syntax (parameter expansion, `--version`) and does not populate the environment the way real Bash does. The script never had a chance to run; it failed inside its own `#!/usr/bin/env bash` shebang-equivalent invocation.

**Solution**: Use Git for Windows' real Bash instead — `C:\Program Files\Git\bin\bash.exe` (already present on any machine with Git installed) — and invoke the script explicitly through its full path rather than relying on whatever `bash` resolves to on PATH. The script then ran correctly end-to-end (skill records rendered, `[refresh] OK`).

**Rule**: On Windows, never trust a bare `bash` on PATH to be GNU Bash — multiple tools (scoop, WSL stubs, Git for Windows, MSYS2) can all provide something answering to that name with materially different capabilities. Before running any bash script that isn't trivially POSIX, verify with a cheap probe (`bash -c 'echo ${x//a/b}'` or just `bash --version`) or invoke Git for Windows' bash by its full path (`C:\Program Files\Git\bin\bash.exe`) directly — it is the one on this machine confirmed to be real Bash.
