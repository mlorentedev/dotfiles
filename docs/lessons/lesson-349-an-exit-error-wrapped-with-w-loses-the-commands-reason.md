---
id: "lesson-349-an-exit-error-wrapped-with-w-loses-the-commands-reason"
type: lesson
status: active
title: "An exit error wrapped with %w loses the command's reason"
created: "2026-10-07"
---

# An exit error wrapped with %w loses the command's reason

## Context
`dotf pr land` stopped a landing queue with `gh pr update-branch: exit status 1` (#2087).
Run by hand a minute later, the same command succeeded. The failure was transient, and its
reason was gone: nothing in the output said what gh had refused.

## The Trap
`exec.Cmd.Output` does capture stderr: it stores it in `*exec.ExitError.Stderr`. But
`ExitError.Error()` renders only `exit status N`, so `fmt.Errorf("...: %w", err)` prints a
message with the reason removed, while the reason sits in a field nobody reads. The code looks
correct, since it wraps the error, and the test that stubs the runner never sees a real
`ExitError`.

The repository had already fixed this twice, each time in its own way (`secrets.AgeDecrypt`
captures stderr in a buffer, `spec.stderrOf` read the field), while the two gh runners that
`pr land` and `pr triage-queue` use still had the bug.

## The Solution
Use `execerr.WithStderr(err)` wherever a command's `Output()` error leaves the function. It
appends the captured stderr to the message and still unwraps to the `*exec.ExitError`, so
exit-code checks keep working. Its test runs a real failing command, because only a real
`ExitError` carries the field that gets lost.
