---
id: "lesson-488-winget-links-a-portable-package-only-where-it-may-create-symlinks"
type: lesson
status: active
title: "winget links a portable package only where it may create symlinks"
created: "2026-10-18"
---

# winget links a portable package only where it may create symlinks


**Context:** #2770 made the catalog install uv on Windows through winget (`astral-sh.uv`), a portable package. The tools step then appended `%LOCALAPPDATA%\Microsoft\WinGet\Links` to its own PATH, where winget is documented to link portable packages, so the uv tools waiting on uv could install in the same run.

**Problem:** On the clean Windows 11 Arm64 test guest, winget installed uv and the step still failed: `uv (for poetry), uv (for pre-commit)` were not on PATH. The Links dir was empty. winget had put uv in `WinGet\Packages\astral-sh.uv_Microsoft.Winget.Source_8wekyb3d8bbwe\` and added that directory to the user's PATH in the registry. The link needs the right to create symlinks, which an unelevated user without Developer Mode lacks; the install ran as such a user. tea and eza landed the same way.

**Solution:** The step reads back the PATH installers registered: the machine and user `Path` values, with `REG_EXPAND_SZ` expanded, as `Sync-SessionPath` does in PowerShell. It reads them again before retrying the entries that waited. One reader serves every installer (winget's portable packages, MSIs, Git for Windows), so no step guesses an install directory. `env.RegisteredPath` is the reader; off Windows it returns nothing.

**Why:** On Windows, an installer reports where it placed a tool in the registered PATH, for the next process. A directory derived from documentation describes only one configuration of the installer. Read what the installer registered, and confirm it on a machine with the default rights: a developer's box with Developer Mode on is the configuration the documented path describes.
