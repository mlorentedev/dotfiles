---
id: "lesson-416-a-foreign-home-on-windows-splits-every-tool"
type: lesson
status: active
title: "A foreign user-scope HOME on Windows splits git, ssh and dotf across two homes without an error"
created: "2026-10-13"
---

# A foreign user-scope HOME on Windows splits git, ssh and dotf across two homes without an error

## Context
On 2026-10-13 a Windows session on the corporate box began with `dotf doctor` reporting 23
FAILs. Most were files "missing" from a home nobody recognised, `C:\SPB_Data`. A Cadence
Allegro install the day before had set the user-scope variable `HOME=C:\SPB_Data`, as the SPB
installer does to place its `pcbenv` directory (#2432).

## The Trap
Windows has no `HOME` of its own, so whatever sets one wins for every tool that prefers it:

- Git for Windows read `C:\SPB_Data\.gitconfig`, which did not exist, so `user.email` was unset and commits failed.
- dotf's `env.Home()` and doctor's `System.home()` prefer `HOME`. Twelve other call sites use Go's
  `os.UserHomeDir()`, which on Windows reads only `USERPROFILE`, so a single doctor run
  checked two homes at once.
- `dotf secrets` materialised SSH keys, the kubeconfig and recovery codes into the foreign
  home. That directory sat under `C:\`, whose inherited ACL grants `Authenticated Users: Modify`.

Nothing failed loudly. Each tool answered about the home it had resolved. The only clue was a
FAIL that named a path the owner had never used.

A second trap came with the fix. In PowerShell 7, `[Environment]::SetEnvironmentVariable('HOME',
$null, 'User')` left an empty `REG_SZ` in `HKCU\Environment` rather than deleting the value. Only
`Remove-ItemProperty -Path HKCU:\Environment -Name HOME` removed it.

## The Solution
Delete the user-scope `HOME`, restart the terminal so that processes inherit the clean
environment, and verify with `reg query HKCU\Environment /v HOME`. Cadence falls back to
`%USERPROFILE%\pcbenv`. Inventory the foreign home by hash against the real one before deleting
anything. Remove junctions with a link-only `rmdir`, so the deletion does not follow them into
the vault. Move any state that exists only there, such as the gate ledgers, back to the real
`StateDir`.

When a Windows doctor run reports paths outside `%USERPROFILE%`, check `HOME` first: that
symptom means a third-party installer has repointed the home. The durable guard is a doctor
check that FAILs when `HOME` disagrees with `USERPROFILE`, plus a single home resolver (#2432).
