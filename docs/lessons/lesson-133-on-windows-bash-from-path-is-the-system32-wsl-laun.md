---
id: lesson-133-on-windows-bash-from-path-is-the-system32-wsl-laun
type: lesson
status: active
created: "2026-06-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 133: On Windows, `bash` from PATH is the System32 WSL launcher, not Git Bash — resolve the real interpreter before shelling out

**Context**: `dotf mem session-start` ports `session-brief.sh` (HARNESS-026) and shells out to `vault-health.sh` via Go. The first cut used a bare `exec.Command("bash", script)` (the faithful port of the shell's `bash "$vault_health"`). It worked on Linux/macOS but on Windows the vault-health step failed with `execvpe(/bin/bash) failed: No such file or directory` (#629).

**Problem**: Windows ships `C:\Windows\System32\bash.exe` — the **WSL launcher**, not a real bash. It precedes Git Bash on `PATH` for most installs, so Go's `exec.LookPath("bash")` (and a bare `"bash"` argv, which the OS resolves the same way) picks it. Two distinct failure modes hide behind it: (1) with **no WSL distro installed** it is a broken stub that aborts with `execvpe(/bin/bash) failed`; (2) even with a working distro, WSL runs in the Linux namespace and **cannot read a Windows-path script argument** — it sees `/mnt/c/...`, not `C:\...`, so `bash C:\...\vault-health.sh` fails to find the script. A bare `"bash"` is therefore wrong on Windows in *both* WSL states.

**Solution**: Added `resolveBash()` (`cli/internal/mem/session_start.go`) and route every bash shell-out through it instead of a literal `"bash"`. Resolution order: `$DOTF_BASH` override → the first `bash.exe` on `PATH` that is **not** `%SystemRoot%\System32\bash.exe` (compared case-insensitively, the WSL launcher is skipped) → a bare `"bash"` fallback (Linux/macOS have no System32 ambiguity, so behaviour there is unchanged). dotfiles installs Git Bash; that is the interpreter that can run a Windows-path script. The same resolver is reused by the deploy doctor check (`cli/internal/doctor/checks_deploy.go`).

**Rule**: On Windows, never shell out to a bare `"bash"` (nor trust `exec.LookPath("bash")`) — it resolves to the System32 WSL launcher, which is a broken stub without a distro and cannot read Windows-path arguments even with one. Resolve a real interpreter first: prefer an explicit override env var, then skip the `%SystemRoot%\System32\bash.exe` candidate when scanning `PATH`, then fall back to `bash` only where there is no such ambiguity (POSIX). Cross-platform Go that ports a shell script must treat the interpreter itself as a resolution problem, not a constant.

---
