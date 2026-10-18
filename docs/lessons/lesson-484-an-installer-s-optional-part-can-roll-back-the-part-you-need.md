---
id: "lesson-484-an-installer-s-optional-part-can-roll-back-the-part-you-need"
type: lesson
status: active
title: "An installer's optional part can roll back the part you need"
created: "2026-10-18"
---

# An installer's optional part can roll back the part you need


**Context:** `dotf vm windows build` installs Windows 11 Arm64 unattended under QEMU, and the first logon installs the QEMU guest agent from virtio-win 0.1.302, then shuts down. The agent is the only way into the guest: the build waits for that shutdown, and every later run talks to the agent.

**Problem:** Measured on 2026-10-18, the first live build:
- `msiexec /i qemu-ga-x86_64.msi` exited 1603. Its verbose log names the `RegisterCom` custom action: `rundll32 qga-vss.dll,DLLCOMRegister`, which registers the agent's VSS provider in COM+, fails on Arm64 with error 1722. The MSI then rolled the whole install back, agent included.
- `qemu-ga.exe -s install` registers the same provider first and gives up the same way.
- VSS only backs filesystem freeze, which nothing here uses. A dependency of one command took down all of them.
- With no agent, the build waited out its timeout on a black screen. The same build had already shown a second trap: the Enter key pressed to answer the ISO's "press any key" prompt kept arriving after Setup opened, landed on its focused Cancel button, and asked to quit the install.

**Solution:**
- Extract the MSI with an administrative install (`msiexec /a … TARGETDIR=…`), which runs no custom action.
- Create the service with the command line the MSI would give it: `New-Service … -BinaryPathName '"…\qemu-ga.exe" -d --retry-path'`.
- Measured on the same guest: the service runs, and the host's `guest-sync` gets its answer.
- Answer the boot prompt with an arrow key, which wakes it and does nothing on Setup's lone button.

**How it was diagnosed:** QMP was the way in without an agent. `screendump` showed the screen; Shift woke a blanked display; `send-key`, one qcode per character, typed into Win+R. Ctrl+Shift+Enter, then Left and Enter on the UAC prompt, opened an elevated PowerShell. That shell printed the first logon's transcript, then the MSI's verbose log. Keep that channel for any headless guest whose agent is the thing that failed.

**Why:** An installer is all-or-nothing by design, so a component you never call can still decide whether you get the one you do. When an install fails, read which action failed before blaming the platform: here the x64 agent ran under Arm64 emulation without trouble, and only its optional COM+ registration did not.
