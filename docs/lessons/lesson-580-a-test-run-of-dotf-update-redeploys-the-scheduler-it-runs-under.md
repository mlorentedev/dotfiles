---
id: "lesson-580-a-test-run-of-dotf-update-redeploys-the-scheduler-it-runs-under"
type: lesson
status: active
title: "A test run of dotf update redeploys the scheduler it runs under"
created: "2026-11-11"
---

# A test run of dotf update redeploys the scheduler it runs under

## Context

To verify the quiet-day `system-upgrade` tick (TOOL-036, #3312) on a real Mac, `dotf update` ran from a source build
against a scratch clone: `DOTFILES_REPO_DIR=<clone> go run ./cmd/dotf update --state-home <scratch>`. The clone's
upstream was the feature branch, and `--state-home` kept the run's `last.json` away from the real one.

## Problem

The first run was a quiet day and ran only the one step. Between runs a commit was pushed to the branch, so the
second run fast-forwarded the clone and ran a full converge on the real home. Every step deployed from the scratch
clone. The `self-deploy` step rewrote the launchd plist with the run's `XDG_STATE_HOME`, because a trigger carries
the homes of the converge that installed it (#2696). From then on the real 03:00 run would have written its record
into the scratch directory, where doctor never looks. The run exited 0 and reported `updated`.

## Solution

`dotf converge --only self-deploy`, run with the installed dotf and the normal environment, rewrote the plist without
the scratch home. A second run reported `changed=0`. The other steps had deployed content equal to main, so nothing
else needed undoing.

## Takeaway

`dotf update` is a deploy, not a probe: every flag and env var it runs with can end up in the scheduler it installs.
Verify the quiet-day path against a clone whose upstream cannot move: clone it with `--no-local` from a fixed commit,
or point the remote at a bare copy nobody pushes to. Use `dotf converge --only <step>` for anything else. After any
real run with a non-default home, read the trigger back (`launchctl print`, the plist, `systemctl --user cat`) before
calling the machine clean.
