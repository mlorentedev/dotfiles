---
id: lesson-301
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, harness, lessons, adr, definition-of-done, ci]
---

# 301 — Knowledge goes where a mechanism asks for it

## What happened

Over three weeks, dotfiles went from 0.42 lessons per commit to 0.18, and wrote no ADR. It felt like less was being learned. Measured, the opposite held.

One session merged 13 PRs and filed 10 tickets, each with a measured root cause. Its PR bodies carried evidence tables, and one spec went through four review rounds with every finding dispositioned. It added no lesson and no ADR, and declined one lesson candidate as "captured elsewhere".

Every place the knowledge did land had a mechanism asking for it:

- the spec gate asks for a spec;
- review-attestation asks for a review;
- a ticket asks for a root cause.

Lessons and ADRs had only Definition of Done §2 and the handoff's harvest step. Both are text, and both are read at the end of the work, when the next task is already waiting.

## The rule

When a kind of record stops being written, look first for what asks for it, not for who forgot.

- An instruction read at the end of the work loses to the next task.
- A check at the moment of the change does not: here, the PR that produced the knowledge.

The check needs a cheap way out that still makes the author decide. For this one, that is `none: <reason>`, one sentence per line.

Refs: HARNESS-024 (#387), ADR-039.
