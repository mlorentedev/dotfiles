---
id: "lesson-354-a-skip-reason-covers-only-the-question-it-answers"
type: lesson
status: active
title: "A skip reason covers only the question it answers"
created: "2026-10-07"
---

# A skip reason covers only the question it answers

## Context
`dotf doctor` compares every `ai/deploy.json` entry with its installed copy. It skipped every
`render: true` entry, because comparing one needs `secrets render` and the daemon, and doctor
stays read-only. On the new Mac, pi was installed and `~/.pi/agent/models.json` had never been
deployed. The manifest check counted pi as "not compared", and the two pi checks said SKIP. pi
had no models, and doctor reported the machine as healthy (#2100).

## The Trap
The reason was valid, but it was too broad. "Cannot render here" stops doctor from asking whether
the content matches. It does not stop doctor from asking whether the file exists, which needs no
render. Because the guard was a single `c.Render ||` in the skip condition, one valid reason
silenced two questions. Nothing failed, so nothing showed that the second question had stopped
being asked.

Gating a fix like this exposes a second form of the trap. Adding `requires: pi` to every
`~/.pi` entry turned `verify-setup` red. That test checked the pi settings merge inside the
integration image, which has no Node.js and therefore no pi. The check was green only because the
box lacked the tool the file is for (#2102).

## The Solution
When writing a skip, name the question it answers and keep asking every other question. Here,
doctor still skips the content comparison for a rendered entry. If the entry applies on this OS,
its `requires` tool is present and the destination is absent, doctor now WARNs
`<name>: <dst> not deployed (run: dotf deploy <name>)`. To mutation-check the fix, disable the
existence branch. The new subtest goes red.

Before gating an entry on a tool, read the CI log for each leg that deploys it, and look for the
`deployed` and `skipped` lines. They show which tests depend on the entry deploying where the tool
is absent.
