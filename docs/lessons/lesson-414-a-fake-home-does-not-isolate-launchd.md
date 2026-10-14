---
id: "lesson-414-a-fake-home-does-not-isolate-launchd"
type: lesson
status: active
title: "A fake HOME does not isolate launchd: a test's brew services job outlives the test"
created: "2026-10-13"
---

# A fake HOME does not isolate launchd: a test's brew services job outlives the test

## Context
On 2026-10-13 an end-to-end run of the Colima engine fix (#2013 P5b) used a throwaway `HOME`
under the session scratchpad. `brew services start colima` wrote its plist there and loaded it.
Hours later `dotf doctor --fix` stopped the hand-started VM to move it into the service, and
`brew services start colima` failed with `Bootstrap failed: 5: Input/output error`.

## The Trap
`HOME` decides where Homebrew writes `~/Library/LaunchAgents/sh.brew.colima.plist`. It does not
decide where launchd loads it. The job went into the real user domain, `gui/501`, under the
label `sh.brew.colima`. It kept running after the test because its plist has
`KeepAlive: SuccessfulExit`, so the `colima stop` that ends `colima start -f` cleanly made launchd
start it again. The real plist could not load while a job with the same label was loaded from
the scratchpad path. `launchctl print gui/501/sh.brew.colima` showed the scratchpad `path =`, and
nothing else pointed to the test.

## The Solution
Unload the leaked job by label (`launchctl bootout gui/$UID/sh.brew.colima`), then let doctor
start the service from the real plist. In a test, any command that registers with launchd,
systemd `--user` or a login item reaches the real session whatever `HOME` says. Stub it, or
boot it out in the test's cleanup. A fake `HOME` isolates files, not services.
