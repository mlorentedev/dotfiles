---
id: lesson-192-a-multi-call-binary-bug-report-can-name-the-wrong-
type: lesson
status: active
created: "2026-08-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 192: A "multi-call binary" bug report can name the wrong mechanism — verify the dispatch, not just the symptom

**Context**: BUG-054, fixing `tests/install-dotf.bats`'s busy-binary fixture so the ETXTBSY swap path it claims to exercise is actually reached. The filed issue diagnosed the root cause precisely — `sleep` copied to a file named `dotf` exits immediately instead of sleeping on a multi-call coreutils build — and proposed a fix: `exec -a sleep "$0" 30` to hand the copy the argv[0] the dispatcher expects while the file on disk keeps the name `install_dotf` needs to swap.

**Problem**: the proposed fix does not work on this machine, and the reason is that "multi-call binary" names a shape, not a single mechanism. GNU coreutils' traditional multi-call dispatch reads argv[0], which is exactly what `exec -a` controls — but this system's `sleep` is `uutils coreutils` (the Rust reimplementation), confirmed by `sleep --version`. Direct testing (`/proc/$PID/cmdline` right after `exec -a sleep ... &`) showed the override reaching the process correctly, yet the copy still refused with `unknown program 'dotf'`. Invoking the SAME bytes at their original resolved path dispatched correctly regardless of argv[0]; invoking the copy at a different path failed regardless of argv[0]. That is only consistent with dispatch keyed on the executable's own resolved path (`current_exe()`/`/proc/self/exe`), not on argv[0] at all — the opposite axis from what the issue assumed and what `exec -a` can influence.

**Solution**: stop trying to satisfy whichever dispatch mechanism a given coreutils build uses, and stand in a binary that has no multi-call dispatch to satisfy in the first place. A copy of `bash`, driven by `"$DEST/dotf" -c 's=$SECONDS; while (( SECONDS - s < 30 )); do :; done'`, busy-spins using only shell builtins — no forked or exec'd child, so the copied ELF's own text image stays open for the whole duration regardless of what its filename is or which coreutils flavor is on the host. Verified before wiring into the test: copying bash to a renamed file and running it directly reproduced ETXTBSY on a second `cp` onto it while busy, confirming the mechanism independent of any dispatch behavior.

**Rule**: a bug report's stated root cause is a hypothesis until it reproduces on the machine actually running the fix, even when the diagnosis reads as obviously correct and even when it comes with a plausible-looking patch. "Multi-call binary" covers at least two different dispatch strategies (argv[0]-keyed, as GNU coreutils and busybox use; resolved-path-keyed, as observed here in uutils coreutils) that look identical from a bug report's black-box symptom but demand opposite fixes. When a fixture needs to hold a *real* executable text image open/busy without depending on a specific tool's argv0-vs-path dispatch behavior, prefer a binary with no multi-call ambiguity at all (a shell, driven by a builtin-only blocking loop) over trying to satisfy whichever mechanism the host happens to implement.

**Tags**: `testing`, `bash`, `coreutils`, `debugging`
