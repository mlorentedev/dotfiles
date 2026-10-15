---
id: "lesson-433-launchctl-getenv-answers-for-the-callers-domain-not-the-login-session"
type: lesson
status: active
title: "launchctl getenv answers for the caller's domain, not the login session"
created: "2026-10-14"
owner: manu
tags: [go, macos, launchd, silent-failure]
---

# launchctl getenv answers for the caller's domain, not the login session

**Context:** #2507. Inside herdr, the session-start hook and `dotf doctor` reported all twelve persisted variables as missing, and `dotf converge` offered to set them again. From a login terminal the same checks were clean, and `dotf update` had reported `env-persist` OK minutes earlier.

**Problem:** `launchctl getenv` and `setenv` act on the bootstrap domain of the calling process. A process under tmux or herdr is not in the login session's domain: `launchctl managername` answers `Background` there and `Aqua` in a login terminal and in an agent bootstrapped into `gui/<uid>`, such as the login agent that re-applies the scope. So from herdr every `getenv` read an empty domain and reported drift, and a `setenv` from there would have written into `Background`, where no app launched from the Dock reads it, and then reported the write as done. Each probe returns exit 0 and empty output, which is exactly what "unset" looks like.

**Solution:** the launchd store asks `managername` first. In the login session it keeps using `getenv`/`setenv`. Elsewhere it reads the login session's environment block from `launchctl print gui/<uid>`, which any domain can see, and refuses to write with `ErrOutsideLoginSession`. Converge reports a drift it cannot reach as a skip that says to run from a login terminal, instead of writing to the wrong domain. Output that has no `gui/<uid> = {` header is an error, never an empty scope.

**Why:** a per-process scope that looks global reads as absent from the wrong scope, and absence is a plausible answer. Before trusting "unset" from such a tool, establish which scope the caller is in; and test from the context the tool actually runs in (a multiplexer here), not only from the terminal where it was written.
