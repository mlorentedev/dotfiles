---
id: "dotfiles-troubleshoot-claude-mem-surrogate-400"
type: troubleshooting
status: active
tags: [troubleshooting, dotfiles, claude-code, claude-mem, unicode, surrogate]
created: "2026-06-05"
owner: manu
---

# Troubleshooting: `400 ... no low surrogate in string` (claude-mem astral emoji)

Every request to the Anthropic API fails — the session is bricked, even after `/clear`:

```
API Error: 400 The request body is not valid JSON: no low surrogate in string: line 1 column 33521 (char 33520)
```

Self-healed at every session start by `claude-mem-heal.sh` (BUG-019). This doc explains the root cause and the recurrence playbook for when a NEW astral source appears.

## Root cause

`no low surrogate` means the serialized JSON request contains a **lone UTF-16 high surrogate** — an astral (non-BMP, 4-byte) character whose surrogate pair was split mid-pair.

The source is the **claude-mem SessionStart `context` hook**, which injects a `# [<project>] recent context` block whose legend uses astral emoji:

```
Legend: 🎯session 🔴bugfix 🟣feature 🔄refactor ✅change 🔵discovery ⚖️decision 🚨security_alert 🔐security_note
```

`🎯` (U+1F3AF) etc. serialize to surrogate pairs (`🎯`). Claude Code truncates the assembled request context at a UTF-16 **code-unit** boundary (classic JS `String.slice` bug) and splits a pair, leaving a lone high surrogate → the API rejects the whole body.

Two conditions must coincide, which is why it looks intermittent / repo-specific:

1. **An astral char in auto-loaded context** — claude-mem's legend (and a stale `<claude-mem-context>` block claude-mem used to write into `CLAUDE.md` files when `CLAUDE_MEM_FOLDER_CLAUDEMD_ENABLED=true`).
2. **A large enough total context to push that char onto the truncation boundary** — e.g. this repo's ~28 KB `AGENTS.md`. Smaller repos send the same emoji whole (before the cut) and never fail.

The position (`char ~33520`) is stable because it tracks the fixed truncation point, not the conversation.

> `CLAUDE_MEM_EXCLUDED_PROJECTS` does **not** reliably stop the injection (the daemon/context path ignores it). Do not rely on it.

## The fix (automated)

`claude-mem-heal.sh` → `neuter_context_hook()` replaces the `node ... hook claude-code context` invocation with `true` in every claude-mem `hooks.json`, so no context block is injected. Capture + `mem-search` are unaffected; only the auto-injected "recent activity" block is lost. It re-applies every session, surviving `/plugin update`.

Upstream fix request (use BMP markers so the block can be safely re-enabled): [thedotmack/claude-mem#2787](https://github.com/thedotmack/claude-mem/issues/2787). Claude Code's truncation bug: anthropics/claude-code#61301, #16294, #60168.

## Recurrence playbook (a NEW astral source)

If the 400 returns from a different source, **do not scan files blindly** — capture the real request and read the offending byte:

```sh
# 1. Capture the raw request body Claude Code sends (reproduces headless)
mkdir -p /tmp/cmdbg
cd <the-failing-repo>
OTEL_LOG_RAW_API_BODIES=1 CLAUDE_CODE_DEBUG_LOGS_DIR=/tmp/cmdbg timeout 90 claude -p "ping" >/dev/null 2>&1

# 2. Find the astral char(s) in the captured body
python3 - <<'PY'
import glob
t = open(sorted(glob.glob('/tmp/cmdbg/*.txt'))[-1], encoding='utf-8', errors='replace').read()
for i, c in enumerate(t):
    if ord(c) > 0xFFFF:
        print(f'@{i} {hex(ord(c))} {c!r}  …{t[max(0,i-50):i+8]!r}')
PY
```

The surrounding text identifies the source (a hook's `additionalContext`, a `CLAUDE.md`/`AGENTS.md`, a tool/skill description, the `<env>` block). Then either remove the astral char at the source or neuter the injecting hook. Verify by re-capturing: zero astral in the body.

> Lesson (cross-project): when several file-scan hypotheses fail, **capture the actual request instead of guessing**. See vault `10_projects/knowledge/90-lessons.md`.
