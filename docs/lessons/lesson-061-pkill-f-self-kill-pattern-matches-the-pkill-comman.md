---
id: lesson-061-pkill-f-self-kill-pattern-matches-the-pkill-comman
type: lesson
status: active
created: "2026-05-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 061: pkill -f self-kill: pattern matches the pkill command line itself

**Context:** During a cleanup session uninstalling several AI CLIs (opencode, agy, gemini-cli), I ran `pkill -TERM -f 'opencode|antigravity|/agy\b|gemini-cli'` to terminate any live processes. The shell exited with code 144 (128 + SIGTERM 16) — pkill killed the shell that invoked it.

**Problem:** `pkill -f PATTERN` matches against each process's **full command line** (argv as a single string). The shell running my pkill command had a command line containing the literal string `pkill -TERM -f 'opencode|antigravity|/agy\b|gemini-cli'` — and `opencode` substring of that command line matches the pattern `opencode|...`. So pkill cheerfully matched ITS OWN INVOKING SHELL and signaled it. The shell got SIGTERM mid-command and died. This is silent and confusing: the cleanup appears to have failed (exit 144) but actually nothing was killed except the wrapper itself.

This is a recurring class — anyone writing a script that uses `pkill -f` with a pattern broad enough to match human-readable words will hit this on certain shell invocations (especially when the pattern is constructed dynamically from user input or config). The bash-tool-in-an-AI-loop case is particularly prone because each command is launched in its own subshell whose argv contains the full command string.

**Solution:** Three safer alternatives in order of robustness:

```bash
# 1. Match the basename only (no -f flag) — kills by program name not command line.
#    Best when you know the exact binary name.
pkill -x opencode 2>/dev/null
pkill -x agy 2>/dev/null

# 2. With -f, explicitly exclude $$ (current shell PID).
MYPID=$$
pgrep -f 'opencode|/agy|gemini-cli' | grep -v "^$MYPID$" | xargs -r kill -TERM

# 3. Use a sentinel that the pattern won't match in your own command.
#    Wrap the pattern in [oo]pencode trick used by ps/grep — the brackets aren't
#    literal in regex but the LITERAL string '[o]pencode' doesn't match 'opencode'.
pkill -f '[o]pencode|[a]ntigravity'   # matches OUR processes only
```

**Why:** `pkill -f` is one of those tools where the obvious mental model ("match programs whose name contains X") is wrong — it actually matches "processes whose entire command line contains X", and your own pkill invocation IS such a process. The semantics are documented but counter-intuitive, and the failure mode (exit 144, no error message, your script terminated) gives no clue what happened.

**How to apply:** Default to `pkill -x BASENAME` when you can. Only use `pkill -f PATTERN` when you genuinely need command-line matching (e.g., killing one specific instance of a daemon with distinguishing args), AND in that case either exclude `$$` explicitly OR use the `[X]regex` self-exclusion trick. In any AI-agent loop or wrapper script context, never trust `pkill -f` with broad patterns without one of these guards — the wrapper's own argv WILL match common English words.

**Tags:** `#shell` `#process-management` `#pkill` `#self-kill` `#exit-codes` `#dotfiles` `#ai-loop-gotcha`
