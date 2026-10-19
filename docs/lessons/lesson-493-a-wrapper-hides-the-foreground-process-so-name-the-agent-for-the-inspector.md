---
id: "lesson-493-a-wrapper-hides-the-foreground-process-so-name-the-agent-for-the-inspector"
type: lesson
status: active
title: "A wrapper hides the foreground process, so name the agent for the inspector"
created: "2026-10-18"
---

# A wrapper hides the foreground process, so name the agent for the inspector

## Context

herdr identifies the agent in a pane by the **foreground process of the pane's
terminal**, and pi and opencode were the two agents that never appeared in its
Agents sidebar (`agent: null`, `agent_status: "unknown"`) while claude and agy
did. The difference was not the agent: it was the wrapper. `pi()` and
`opencode()` hand the launch to `dotf secrets run`, which allocates its own pty
for the child (`runChildPTY`), so on the pane's terminal the foreground process
is `dotf` and the agent runs on a tty the inspector never sees:

```
$ herdr pane process-info --pane w10:p1
foreground_processes: [{argv0:"dotf", cmdline:"dotf secrets run --only … -- pi"}]
$ ps -eo tty,comm -p 27887
pi    ttys012        # a different tty, invisible to herdr
```

## Decision

`HERDR_AGENT=<canonical agent id>` on the wrapper command, as herdr documents
for host-visible wrappers ("the hint applies only to that foreground process"):

```sh
pi() { HERDR_AGENT=pi dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY -- pi "$@"; }
```

## Why not in the CLI

`dotf` could set the variable for its child, and it would still be invisible:
herdr reads the environment the foreground process **was exec'd with** (Linux
`/proc/PID/environ`, macOS `sysctl KERN_PROCARGS2`), which an `os.Setenv` after
exec does not rewrite — the kernel hands out the initial stack. Setting it on
`dotf` itself would need a self re-exec, and inferring the id from
`childArgv[0]` would couple the CLI to another tool's vocabulary. The caller
names the agent; the wrapper is the caller.

## Consequences

- The hint is per-command, never an `export`: an exported value mislabels every
  later command in that shell, which is the failure the herdr docs warn about.
- Registration needs the hint **and** a screen that matches the agent's manifest
  (or a live report from the agent's own integration). A hint on a process whose
  screen matches nothing does not register: `HERDR_AGENT=pi sleep 180` stayed
  invisible, while the real `pi` under the same hint registered with
  `agent_session` and `screen_detection_skipped: true`.
- The gap this leaves is documented on the issue: a launch that bypasses the
  wrapper (`dotf secrets run -- pi` from a script, including the reviewer panes
  in `cli/internal/spec/review_launch.go`) is still invisible, and is fixed by
  the caller passing the hint, not by `dotf` guessing.
- Both values are also herdr's `rows_by_agent` keys, so the sidebar layout that
  was dead code in `ai/herdr/config.toml` (#2646) starts rendering with no
  further change.
