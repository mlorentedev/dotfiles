---
id: "lesson-468-a-machine-role-written-by-an-unreleased-dotf-breaks-the-released-one"
type: lesson
status: active
title: "A machine role written by an unreleased dotf breaks the released one"
created: "2026-10-17"
---

# A machine role written by an unreleased dotf breaks the released one


**Context:** OPS-053 (#2502) adds the `self-deploy` machine role. To prove the launchd half live on the macmini, a dev build ran `dotf env role add self-deploy` against the real `~/.config/dotfiles/machine.json`, then `dotf converge --only self-deploy`.

**Problem:** `LoadMachineRoles` refuses an unknown role on purpose (PLAT-001b 7b: a typo must not become a role nobody reads). The installed `dotf` 0.73.0 does not know `self-deploy`, so from that write on every `dotf converge` on the machine failed at workstation-target with `unknown machine role "self-deploy" (known: ssh-target)`, exit 1. That included the parallel sessions' converges and, at 03:00, the very agent the test had installed, which runs the installed `dotf update`. A peer session (wintarget-14) noticed it about 25 minutes later. The dev build's own tests and its live run were green the whole time, because it knew the role.

**Solution:** The role and the agent were rolled back with the dev build (`dotf env role rm self-deploy`, then the step's own opt-out removed the agent), and the live proof was kept: the renderer and the plist bytes do not change between the test and the release. A test of a change to `machine.json` runs against a scratch file, never the machine's real one, until a release that reads the new value is installed. A scratch `HOME` alone is not one: `env.MachinePath` prefers `$XDG_CONFIG_HOME`, so the step's own unit tests, which wrote the role under `t.TempDir()`, shared CI's `machine.json` and failed there (the opted-in tests declared the role for the rest), and on a host that exports `XDG_CONFIG_HOME` they would have written the real file. They set `XDG_CONFIG_HOME` to a temp dir too, as `workstationFixture` does. On the real file, the new role is declared after that release, not before.

**Why:** `machine.json` is shared state read by every `dotf` on the machine, and the one that reads it most is the pinned release, not the build under test. A strict reader makes every new value a breaking change for the older binaries on the same host. The same two-version window as lesson 464 (a hook pinned to a release), with the difference that this one fails at run time on a schedule, not at commit time in front of you.
