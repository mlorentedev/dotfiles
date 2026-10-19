---
generated: true
generated_from: 00_meta/skills/herdr/SKILL.md
generated_sha: 1afbc2a5799e4a6d
id: herdr-skill
type: skill
status: active
created: '2026-10-19'
owner: manu
name: herdr
description: Use when working inside the herdr terminal cockpit or when an agent pane is missing from its sidebar — diagnosing agent detection, naming an agent for a wrapper, shaping panes and keybindings, and deciding whether a herdr plugin can be pinned at all.
keywords: [herdr, agent-detection, HERDR_AGENT, pane, sidebar, popup, plugin, terminal-workspace, agent-cockpit]
paths: ['**/ai/herdr/**', '**/herdr*/**']
---

# herdr

herdr is the terminal workspace manager that hosts coding agents in real panes
and rolls their state up into a sidebar. Treat it as the environment the agent
runs **in**, not as a library: it owns panes, layout, keybindings and agent
identity, and it exposes all of that through a CLI that reads the same
environment it hands to the processes it launches.

## The one-line contract

An agent appears in the sidebar only when herdr can attribute a pane to it: the
**foreground process** of the pane's terminal must be the agent, or carry
`HERDR_AGENT=<id>`, and its screen must match that agent's manifest (or the agent
must report its own state). Anything that hides the foreground process hides the
agent.

## 1. Is the agent detected? Diagnose before changing anything

```sh
herdr pane process-info --pane <pane>   # who owns the terminal right now
herdr agent get <pane>                  # the agent herdr attributes to it
herdr agent explain <pane>              # why: manifest, matched rule, fallback
herdr pane list                         # every pane, with agent and status
```

`agent: null` / `agent_status: "unknown"` means no attribution. The first place
to look is `process-info`: if `argv0` is a wrapper (`dotf`, `sandbox`, `fence`,
`nono`, `docker exec`), the agent is running somewhere herdr cannot see.

## 2. A wrapper hides the foreground process — name the agent

Most secrets injectors and sandboxes give the child its **own pty**, so the pane
terminal's foreground process is the wrapper and the agent lives on a tty herdr
never inspects. herdr's documented fix is a per-command hint:

```sh
pi() { HERDR_AGENT=pi dotf secrets run --only NAN_API_KEY -- pi "$@"; }
HERDR_AGENT=claude fence -- claude           # herdr's own example
```

Rules that are not style preferences:

- **On the command, never in an `export`.** The hint applies to that foreground
  process; exported, it mislabels every later command in the shell.
- **Not inside the wrapping CLI.** herdr reads the environment the foreground
  process was **exec'd** with (Linux `/proc/PID/environ`, macOS
  `KERN_PROCARGS2`), which a variable set after `exec` does not rewrite. A
  self-reexec would work and is not worth coupling one tool to another's ids.
- **The value is herdr's canonical id** (`pi`, `opencode`, `claude`, `codex`,
  `antigravity-cli`, …) — the same key `rows_by_agent` uses. Never a display
  name or a shell alias.
- **The hint alone is not enough.** Registration needs the hint *and* a screen
  that matches the agent's manifest, or a state report from the agent's own
  integration. `HERDR_AGENT=pi sleep 300` stays invisible.
- **The wrapper is the caller, so a launch that bypasses it stays invisible.**
  A script that runs `dotf secrets run -- pi` directly gets no hint; the fix is
  the caller passing it, not the CLI guessing from `argv[0]`.

## 3. Sidebar rows

Rows are tokens, rendered per agent. The Agent-row vocabulary is small:
`state_icon`, `state_text`, `machine`, `workspace`, `tab`, `agent`,
`terminal_title`, `terminal_title_stripped`, and custom `$name` tokens reported
with `herdr pane report-metadata`. **There is no session token**: what looks like
a session name in a Claude row is `terminal_title_stripped`, because Claude
writes its session name into the OSC title. An agent that sets no title simply
loses that row. Space rows carry `branch` and `git_status` instead.

```toml
[ui.sidebar.agents.rows_by_agent]
pi = [["state_icon", "machine", "workspace", "tab"], ["terminal_title_stripped"], ["agent"]]

[ui]
agent_panel_sort = "priority"   # blocked/done float up (default: "spaces")
```

## 4. Panes and keybindings

`[[keys.command]]` has four types: `popup` (session-modal overlay, gets all
input until it exits), `pane` (temporary zoomed pane), `shell` (detached), and
`plugin_action`. A command receives `HERDR_ACTIVE_PANE_ID`,
`HERDR_ACTIVE_TAB_ID`, `HERDR_ACTIVE_PANE_CWD` and `HERDR_BIN_PATH`; a pane's own
shell exports `HERDR_PANE_ID`, `HERDR_TAB_ID` and `HERDR_WORKSPACE_ID`.

Consequences worth knowing:

- A **popup is not a split.** "On demand" is a popup; "always visible" is a real
  split, which needs `herdr pane split` or a plugin pane opened with
  `--placement split --direction right`.
- A right-docked explorer is a shell command that calls the herdr CLI, or a
  plugin pane. Toggle it by the pane's foreground process, not by a pane label:
  any rename clears the label.
- `herdr config check` validates a `config.toml` without a running server, and
  `HERDR_CONFIG_PATH` points it at a candidate file. It is the only cheap test
  for a binding — herdr silently ignores a key it does not know.

## 5. Plugins: pin them or do not install them

`herdr plugin install <owner>/<repo>[/subdir]` resolves the repository's **HEAD**
unless `--ref` names a commit, tag or branch, and the marketplace is explicitly
an unreviewed index: a plugin runs with the user's privileges. Before installing
one, read its manifest and scripts. If the repo's doctrine is reproducible
provisioning, a plugin has to be installed at a pinned `--ref` by something
committed, and a test has to notice when the pinned version drifts.

## 6. Provisioning and the release boundary

A tool herdr's config runs (`yazi`, an editor, a file manager) belongs to the
tool pin file of the repo that deploys the config, installed through mise so it
exists on every OS, never to a hand install.

A binding or command that names a **subcommand of your own CLI** must wait until
a released version of that CLI carries it. CI builds the tree and stays green; a
new machine installs the released pin and gets "unknown command" on the keypress.
This is lesson 373 in the dotfiles repo and it applies to every tool here.

## 7. Verification checklist

```sh
herdr config check                              # binding valid (HERDR_CONFIG_PATH for a candidate)
herdr agent get <pane>                          # agent, status, session identity
herdr agent explain <pane> --json               # why it matched, or why it fell back to idle
herdr pane process-info --pane <pane>           # wrapper vs agent in the foreground
herdr server reload-config                      # apply a deployed config without restarting
```

## References

- herdr docs: agents and wrappers <https://herdr.dev/docs/agents/>, configuration
  <https://herdr.dev/docs/configuration/>, plugins <https://herdr.dev/docs/plugins/>
- Worked example with the lessons: dotfiles `#2813`/`#2814`, and lessons 493-495
  in that repo's `docs/lessons/`.
