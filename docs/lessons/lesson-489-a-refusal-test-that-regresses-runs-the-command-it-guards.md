---
id: "lesson-489-a-refusal-test-that-regresses-runs-the-command-it-guards"
type: lesson
status: active
title: "A refusal test that regresses runs the command it guards"
created: "2026-10-18"
---

# A refusal test that regresses runs the command it guards


**Context:** #2772 added checks that refuse a bad `--dotf` for `dotf vm windows` before any guest boots: not a file, not a Windows Arm64 PE, not an executable image. Each test runs the cobra command with the bad argument and expects exit 2.

**Problem:** One case was written before its check existed (test first), so the command accepted the file and went on. On the owner's Mac, which holds a sealed `clean-base` from `dotf vm windows build`, it found that base, booted a real Windows guest, copied the fixture in, and failed 84 seconds later in `install.ps1`. The test was red for the right reason, but only by luck: nothing in it stopped a regression from reaching the hypervisor. In CI the same case fails fast, because a runner has no `clean-base`, so the hazard shows only on the one machine that has the real state.

**Solution:** The refusal tests set `XDG_STATE_HOME` to an empty temporary directory (`noGuest(t)`). If a refusal regresses, the command fails on `no clean-base` in under a second and the test reports it. It never reaches QEMU.

**Why:** A test that asserts "refused before X" runs X when the refusal breaks. If X touches state the developer's machine holds (a VM base, a deploy target, a credential), the test needs a seam that keeps X from finding that state. CI cannot catch this, because CI never has the state, so the local machine is the only place it can happen.
