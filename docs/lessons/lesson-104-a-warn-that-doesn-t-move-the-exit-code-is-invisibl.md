---
id: lesson-104-a-warn-that-doesn-t-move-the-exit-code-is-invisibl
type: lesson
status: active
created: "2026-06-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 104: A WARN that doesn't move the exit code is invisible to CI — give the CI surface its own probe, don't shell out to the tool

**Context**: OPS-009 added a PAT-expiry preflight (`dotf doctor`'s "PAT expiry" section) after a classic PAT expired silently and broke release-please's first run with `Bad credentials`. The instinct for the second surface — a scheduled Action that warns *before* CI goes red — was to reuse the binary: run `dotf doctor` in the workflow and act on its exit code.

**Problem**: `Report.ExitCode()` is non-zero only on `StatusFail`; `WARN`/`SKIP`/`INFO` are advisory and never move it (the healthcheck/doctor exit contract). The whole point of the Action is to fire on "expiring **soon**" — which the check classifies as **WARN**. So shelling out to `dotf doctor` and reading `$?` would see exit 0 for exactly the state the Action exists to catch; it could only ever detect an already-dead token (FAIL), i.e. the outage it was meant to prevent. Coupling the CI alert to the binary's exit status would have silently defeated the feature.

**Solution**: The Action does its **own** focused probe (curl `GET /user`, read the `github-authentication-token-expiration` header, compute days-left in ~15 lines of shell) rather than invoking the Go binary. The duplication is deliberate and documented (proposal R5/R7): the two surfaces run in different contexts (local shell env vs Actions secrets) and answer different questions (full diagnostic exit code vs a single "is this one token expiring?" boolean). The local `dotf doctor` surface still shows the WARN to a human who runs it; the Action owns the machine-actionable alert.

**Rule**: Before reusing a diagnostic tool's exit code as a machine signal, check *which* severities actually move that exit code. If the state you need to act on is advisory (WARN/INFO) rather than failing, the exit code can't see it — give the consuming surface its own narrow probe instead of shelling out. A tool whose exit code means "something is hard-broken" cannot also mean "something will break soon"; don't conflate the two axes.
