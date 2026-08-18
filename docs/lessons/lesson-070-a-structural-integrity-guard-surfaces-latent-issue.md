---
id: lesson-070-a-structural-integrity-guard-surfaces-latent-issue
type: lesson
status: active
created: "2026-05-31"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 070: A structural integrity guard surfaces latent issues you didn't know you had

**Context:** SDD-012 built a backlog-integrity guard (`check-backlog-integrity.sh`) to stop the dotfiles `11-tasks.md` from drifting (the same ticket listed twice with diverging status).
**Problem:** Run on the real file, the guard flagged not only the expected view-duplication but **9 number collisions** — two *different* tickets sharing one number (`BUG-020-pwsh` vs `BUG-020-splitpath`, `IDEAS-007-promote` vs `IDEAS-007-cross-provider`, ...). Nobody had noticed; enforcing the invariant exposed them. It also forced a guard-design refinement: match by FULL id (slug-aware), not by number, so legitimate-but-untidy reuse becomes an advisory NOTE instead of a false "duplicate" that would demand history-desyncing renumbers.
**Solution:** When you add a structural guard (incident→guard), expect it to surface adjacent latent issues beyond the bug class it targets — budget for that, and decide up front which are hard-fails vs advisories. Match the guard to the real identity of the thing (full id), distinguishing "same thing listed twice" (drift → hard-fail) from "two things share a label" (collision → advisory). An over-strict guard demands harmful fixes.
**Tags:** `#sdd` `#incident-to-guard` `#backlog` `#guard-design`
