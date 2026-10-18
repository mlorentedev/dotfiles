---
id: "lesson-485-a-config-check-that-validates-syntax-cannot-see-an-ignored-include"
type: lesson
status: active
title: "A config check that validates syntax cannot see an ignored Include"
created: "2026-10-18"
---

# A config check that validates syntax cannot see an ignored Include


**Context:** `dotf converge --only workstation-target` hardens sshd with a drop-in (key only, `AllowUsers <user>@100.64.0.0/10`). Before writing it, `checkInclude` requires the main `sshd_config` to `Include` the drop-in's directory; after writing it, the step runs `sshd -t`. The PLAT-003 Windows test guest (#2737) was the first run of the Windows half against a real Win32-OpenSSH.

**Problem:** Measured on 2026-10-18 with OpenSSH_for_Windows_9.5p2:
- The form `checkInclude` accepts on Windows, `Include C:\ProgramData\ssh\sshd_config.d/*.conf`, is one sshd does not read. It does not treat `C:\` as absolute: `sshd -ddd -t` logs `new include __PROGRAMDATA__\\ssh/C:\\ProgramData\\ssh\\sshd_config.d/*.conf`, then `no match`. Three other absolute spellings behaved the same; only the relative `sshd_config.d/*.conf` was honoured.
- An Include that matches nothing is valid, so `sshd -t` passed. `sshd -T` still printed `passwordauthentication yes` and no `allowusers`.
- The step's error, `… does not include C:\ProgramData\ssh\sshd_config.d/`, told the owner to write exactly the ignored form. Filed as #2763.

**Solution:**
- Check what a config does, not whether it parses: dump the effective configuration (`sshd -T`, with `-C user=…,addr=…` for each address that matters) and compare it with the values the drop-in sets.
- Take the path form from the parser's own debug output on the target platform, never from the other platforms' convention.

**Why:** A syntax check answers "will it start", and an Include that reaches no file is perfectly valid syntax. Every layered config (sshd `Include`, sudoers `@includedir`, systemd drop-ins) can lose a layer this way while reporting success, so the only proof that a drop-in applies is the effective value it was written to change.
