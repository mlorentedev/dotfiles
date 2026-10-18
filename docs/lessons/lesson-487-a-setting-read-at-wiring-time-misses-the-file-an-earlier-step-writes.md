---
id: "lesson-487-a-setting-read-at-wiring-time-misses-the-file-an-earlier-step-writes"
type: lesson
status: active
title: "A setting read at wiring time misses the file an earlier step writes"
created: "2026-10-18"
---

# A setting read at wiring time misses the file an earlier step writes


**Context:** `dotf converge` builds its tools installer in `convergeOptions`, before any step runs. The installer read the Python pin from `versions.conf` right there, through the checkout or the deploy dir (#2769).

**Problem:** On a clean Windows 11 guest, the checkout did not exist yet and `records-mirror` had not created `~\.dotfiles`. The read found no file and warned. Without a pin, `uv tool install` picked the newest interpreter, CPython 3.15.0. pyyaml 6.0.3 has no cp315 wheel for win-arm64, so `pre-commit` tried to build it and the `tools` step failed. The issue first blamed the deploy-dir set, but `versions.conf` was in it all along: the file existed by the time `tools` ran, just not when the value was read.

**Solution:** `Installer.Python` became a function, called at the first uv-tool install and cached. By then the checkout step and `records-mirror` have run. A test builds the installer, writes `versions.conf` afterwards, and expects the pin; reading the pin eagerly fails it.

**Why:** In a run whose early steps create the files later steps read, a value read while the steps are being wired reflects the machine before the run. Only a fresh machine shows the difference, because everywhere else the file already exists. Read such values where they are used, and test with a file that appears after wiring.
