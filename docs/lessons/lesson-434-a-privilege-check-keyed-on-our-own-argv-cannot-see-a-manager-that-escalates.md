---
id: "lesson-434-a-privilege-check-keyed-on-our-own-argv-cannot-see-a-manager-that-escalates"
type: lesson
status: active
title: "A privilege check keyed on our own argv cannot see a manager that escalates inside"
created: "2026-10-14"
owner: manu
tags: [converge, tools, macos, homebrew, sudo]
---

# A privilege check keyed on our own argv cannot see a manager that escalates inside

**Context:** On macOS an unattended `dotf converge` stopped at the `tools` step with `catalog: veracrypt: brew install --cask veracrypt: exit status 1`, and `configs-deploy`, `records-bind`, `git-config` and every step after it never ran (BUG-134, #2469). Run by hand, the cask showed why: it depends on `macfuse`, whose installer runs `sudo installer -pkg`, and with no terminal sudo cannot read a password.

**Problem:** The installer already had a "needs sudo" outcome for apt: skip, print the command, keep going. It never fired for the cask because it was keyed on dotf's own command line. `needsSudoPassword` returns false unless argv starts with `sudo -n`, and an apt install is wrapped that way. `brew install --cask` is not: brew refuses to run as root and escalates inside its own process, so dotf's argv gave no sign. The plan said "install", the apply failed, and a failed reconciler stops the run (ADR-041 decision 4).

**Solution:** The catalog says which casks escalate: `"needs_sudo": true` on the source, accepted only with a `cask`. A marked cask is asked `sudo -n true` before the install and deferred when it refuses, in the plan and in the apply alike. Deferred packages carry their manager, so the one command to run is `sudo apt-get … && brew install --cask veracrypt` and never a cask token behind `sudo apt-get`. `dotf doctor` reports a marked cask on a Mac too. The mark on veracrypt in `packages.json` lands after the release that reads it: the pinned `dotf` rejects an unknown source key and would fail the whole catalog (#1814).

**Why:** Do not read `sudo -n true` refusing as "this install failed for want of a password" unless the install is known to need one. On an unattended Mac the probe always refuses, so applied to every cask it would swallow a 404 or a wrong token as a harmless wait. For apt the inference holds because the command itself was `sudo -n apt-get`; for brew nothing correlates the two, so the knowledge has to be declared, and the probe kept only to let a marked cask install when sudo works. When a classifier works for one manager, ask what ties its signal to the failure before extending it to the next.
