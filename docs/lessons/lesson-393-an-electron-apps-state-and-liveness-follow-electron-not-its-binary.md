---
id: "lesson-393-an-electron-apps-state-and-liveness-follow-electron-not-its-binary"
type: lesson
status: active
title: "An Electron app's state and liveness follow Electron, not its binary"
created: "2026-10-10"
---

# An Electron app's state and liveness follow Electron, not its binary

## Context
`dotf orca tune` rewrites Orca's settings file and must refuse while Orca runs, because the
running app flushes its in-memory state on exit and clobbers the edit (lesson 234). It looked for
the file in `~/.config/orca` and for the process with `pgrep -f "orca-ide|orca-linux.*AppImage"`.
Both were the Linux AppImage's shape, so on macOS `tune` missed the file, and a fixed path alone
would have let it write while Orca ran (#2013 F-024).

## The Trap
- The binary's name and path change with every package: AppImage, `.deb`, `Orca.app`, `orca.exe`.
  A process pattern written for one of them is silently false on the others.
- The state directory looks Linux-specific, but it is Electron's `userData`: `appData` joined with
  the app's internal name. That name is lower-case even when the bundle is not: this Mac has
  `obsidian/` for `Obsidian.app`.

## The Solution
- Resolve the directory the way Electron does: `~/Library/Application Support` on macOS, `APPDATA`
  on Windows, and `XDG_CONFIG_HOME` or `~/.config` elsewhere, each joined with `orca`.
- Decide liveness from the lock Chromium's process singleton keeps in that directory: a
  `SingletonLock` symlink to `<host>-<pid>`. It exists only for apps that call
  `requestSingleInstanceLock()`, and Orca does. Measured here: the Bitwarden and Obsidian locks
  named live pids. The host half is ignored because macOS hostnames drift, and anything
  unreadable reads as running. Windows keeps `tasklist`, since there the singleton is a mutex.
- Read the upstream source before trusting an integration's layout. Orca has since moved its state
  to `profiles/<id>/` and SQLite, which is #2294.

## Takeaways
- For an Electron app, derive paths and liveness from Electron's conventions, which every package
  shares, not from the package you happened to install.
- Fix a path and the guard that protects writes to it in the same change. The wrong path was
  harmless only because it failed first.
