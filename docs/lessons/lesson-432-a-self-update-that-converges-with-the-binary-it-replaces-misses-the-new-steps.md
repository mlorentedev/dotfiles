---
id: "lesson-432-a-self-update-that-converges-with-the-binary-it-replaces-misses-the-new-steps"
type: lesson
status: active
title: "A self-update that converges with the binary it replaces misses the new steps"
created: "2026-10-14"
owner: manu
tags: [go, update, converge, self-update, silent-failure]
---

# A self-update that converges with the binary it replaces misses the new steps

**Context:** on macOS, `dotf update` run by 0.69.0 fast-forwarded the checkout to release 0.70.0 (`DOTF_VERSION=0.70.0`) and reported `self-update complete`. `dotf version` still said 0.69.0, a second `dotf update` printed `already current (origin/main) — no deploy needed`, and `dotf doctor` FAILed `dotf version drift: installed=0.69.0 pinned=0.70.0`. Only `./install.sh` cleared it, and its converge then ran a step 0.69.0 does not have (`pi-settings`) and changed `tools`. #2506.

**Problem:** the fast-forward path converged with the running binary, and no converge step owns the dotf binary pin. A converge reads the checkout but runs the registry compiled into the binary, so an old binary converges the new checkout only as far as it knows. The `current` path never compared the running version with the pin, so after the first run the command answered "nothing to do" about a machine its own doctor failed.

**Solution:** after the checkout is level with its upstream (fast-forwarded or already current), `update.Run` compares the running version with the checkout's `DOTF_VERSION`. On a difference it runs the checkout's installer (`install.sh`, or `install.ps1` under pwsh), which installs the pin and converges with the new binary; `update` does not converge again. A failed install exits non-zero naming both versions. A source build (`dev`) is never replaced and the run says the check was skipped. The installer stages the binary beside the target and renames it into place (`install.ps1` parks a live `dotf.exe` first), so replacing the binary of the process that started it is safe. `DOTF_VERSION` is passed to it explicitly, because it prefers an exported value to the checkout's `versions.conf`.

**Why:** "up to date" for a tool that deploys from its own repository has two halves, the checkout and the binary that reads it, and a command that checks one reports success about both. The signal is in the second run: a skip message ("already current") that coexists with a failing health check means the skip does not cover what the check covers. Test a self-update with a pin above the running version, for each status that leaves the checkout level, not only with a moved checkout.
