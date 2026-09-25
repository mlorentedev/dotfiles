---
id: lesson-298
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, go, processes, signals, timeout, review]
---

# 298 — A wrapped runner is a grandchild: bound its process group

## What happened

A reviewer runs as `dotf secrets run -- pi …`, and `secrets run` starts pi as its own child. A deadline that kills only its direct child therefore leaves pi running and spending, with nobody waiting for it. Putting the runner in a process group of its own fixes that. It also means Ctrl-C and `tmux kill-session` no longer reach the runner, so a killed review session would have orphaned the reviewer.

## The rule

To bound a command that may start others, start it in its own process group. At the deadline, signal the whole group: SIGTERM, then SIGKILL after a grace period. The process that owns the group also forwards SIGINT, SIGTERM and SIGHUP to it. Test both paths with a grandchild that must be gone afterwards. A live `tmux kill-session` check is worth one run too.

Refs: HARNESS-152 (#1710, #1721), `cli/internal/spec/deadline*.go`.
