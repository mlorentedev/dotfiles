---
id: lesson-105-an-npm-global-cli-under-nvm-is-invisible-to-gui-ad
type: lesson
status: active
created: "2026-06-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 105: An npm-global CLI under nvm is invisible to GUI/ADE processes and to any shell on a different node version — install agent CLIs into ~/.local

**Context**: Using Orca (the parallel-agent ADE), `pi` launched fine from the default terminal but `command not found` from inside Orca — only `claude` would start. `setup-linux.sh` installs `pi` with a bare `npm install -g`.

**Problem**: A bare `npm i -g` under nvm lands the binary in nvm's **per-node-version** tree (`~/.nvm/versions/node/<v>/bin/pi`). The terminal resolves node via `nvm use default` (LTS, where pi was installed); Orca — a desktop AppImage that never sources the interactive shell — propagates a PATH carrying a *different* node version (v26 vs the v24 LTS), so `pi` is absent there. Same machine, same user, different node on PATH → the CLI exists but is unreachable. `claude` worked because it lives in `~/.local/bin` as a standalone binary, independent of any node version. The doctor compounded it: when pi was configured (`~/.pi/` present) but off PATH, `checkOpenCode` reported the misleading SKIP "pi not installed" instead of failing on the real cause.

**Solution**: Install pi into the manager-independent `~/.local` prefix — `npm install -g --ignore-scripts --prefix "$HOME/.local"` puts the launcher at `~/.local/bin/pi`, the same dir `claude`/`dotf` use, which is on PATH for login shells AND GUI/ADE processes. Its `#!/usr/bin/env node` shebang then runs under whatever node each environment provides. Guard the install on `~/.local/bin/pi` (the stable location), not bare `command -v pi`, so a stale nvm-version copy can't mask a missing launcher. Add the incident→guard branch to `dotf doctor`: configured-but-off-PATH → FAIL with the root cause, not a SKIP.

**Rule**: Any CLI a GUI app / ADE (or a cron job, or a different-node shell) must spawn has to live on a version-manager-independent PATH dir like `~/.local/bin` — never only in nvm/asdf/volta's per-version global, which is invisible outside the shell that activated that version. When a tool is "found in my terminal but not in <other launcher>", first compare the *active node/runtime version* in each environment before assuming it's uninstalled. And a health check must distinguish "absent" from "present-but-unreachable" — the second is the actionable failure, not a skip.
