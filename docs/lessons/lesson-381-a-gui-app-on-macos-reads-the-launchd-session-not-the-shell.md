---
id: "lesson-381-a-gui-app-on-macos-reads-the-launchd-session-not-the-shell"
type: lesson
status: active
title: "A GUI app on macOS reads the launchd session, not the shell"
created: "2026-10-10"
---

# A GUI app on macOS reads the launchd session, not the shell

## Context
#2013 S3: apps launched from the Dock on the Mac mini saw none of the contract's path variables
(`VAULT_PATH`, `DOTFILES_DIR` …), because `paths.sh` reaches shells only. The fix writes them into
the user's launchd session with `launchctl setenv`, through the `dotf env persist` store that Windows
already has for HKCU.

## The Trap
Three properties of that scope are easy to get wrong, and the first two answer wrongly rather than
failing:

- **`launchctl getenv` cannot tell unset from empty.** It exits 0 and prints nothing in both cases.
  A store that wrote an empty value would read it back as missing, and every later run would see
  drift and write it again.
- **A GUI app started with `open` from a shell inherits that shell's environment.** The first check
  launched an editor this way and saw `VAULT_PATH`, but its `PATH` held the shell's mise entries. The
  value came from `paths.sh`, so the check proved nothing about the launchd session.
- **The scope does not survive logout.** `setenv` is per session. What makes it persistent is a
  LaunchAgent with `RunAtLoad` that re-runs the command at every login.

## The Solution
- The launchd store reads an empty answer as absent and refuses to store an empty value.
- The `env-persist` converge step writes a LaunchAgent that runs the installed
  `~/.local/bin/dotf env persist`, never the running binary. It bootstraps the agent only when
  `launchctl print` fails or the plist changed, and boots it out first in that case, because
  `bootstrap` refuses a loaded label. A second run reports zero changes.
- Doctor's persisted-environment check runs on macOS through the same store. There it names
  `dotf converge --only env-persist`, since `dotf env persist` alone lasts only until logout.

## Takeaways
- **Before relying on a store's "not found", ask how it reports absence.** If absence and an empty
  value look the same, make one of them impossible to write.
- **Verify a GUI app's environment from a launch that carries none of your own.** Any check started
  from your shell inherits what it set.
