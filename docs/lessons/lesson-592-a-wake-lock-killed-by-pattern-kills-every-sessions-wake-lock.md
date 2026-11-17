---
id: "lesson-592-a-wake-lock-killed-by-pattern-kills-every-sessions-wake-lock"
type: lesson
status: active
title: "A wake lock killed by pattern kills every session's wake lock"
created: "2026-11-12"
---

# A wake lock killed by pattern kills every session's wake lock

## Context

An overnight unattended run on the Mac kept the host awake with `caffeinate -i -t 43200`, started as a
background job of the agent's shell tool, as the autopilot skill's host precondition asks. Several other
agent sessions on the same host had armed their own wake locks the same way for their own runs.

## Problem

At disarm the run stopped its wake lock with `pkill -f` on its command line. The pattern also matched
the `caffeinate` processes of the other sessions, which had been started with the same or an
overlapping command line, and sent all of them SIGTERM. One of those sessions was mid-run on a VM and
noticed only because it checked; it re-armed and asked that its process be left alone. Nothing failed
loudly: the host simply became free to sleep under runs that believed they were covered.

The skill's own snippet did not prevent this. It read `caffeinate -i -w $$ &`, which ties the lock to
`$$`, but in an agent's shell tool `$$` is the shell of one tool call: it lives only as long as that call,
and a bare `&` job dies with it. An agent that follows the snippet literally gets no wake lock, so it
improvises one, and the improvised one has no owner record to stop it by.

## Solution

Bound the wake lock by the run's expiry (`-t` set to the seconds until the envelope's `expires_at`), so
it ends by itself; record its PID when it starts; and, to disarm early, kill that PID and nothing else.
Never stop a process by pattern on a host shared by several sessions: a command line is not an owner.
The autopilot skill now says so (vault `00_meta/skills/autopilot/SKILL.md`, rendered into
`harness/skills/autopilot/SKILL.md`).

The general rule: anything a run starts that outlives a tool call needs an owner handle (a PID file, a
named unit, a lease), written when it starts, because the only other way to find it later is a pattern,
and on a shared host a pattern also finds everyone else's.

## Related

- Lesson 591, from the same run.
