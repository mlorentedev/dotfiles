---
id: lesson-072-a-skip-guarded-test-is-green-in-ci-but-a-real-asse
type: lesson
status: active
created: "2026-05-31"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 072: A skip-guarded test is green in CI but a real assertion locally — it can hide a genuine cross-OS parity gap

**Context:** While running the full bats suite locally to verify PR #192, tests/antigravity.bats test 64 ("setup-windows.ps1 syncs Shared Skills to ~/.gemini/skills") FAILED locally, yet the very same commit was GREEN in CI.
**Problem:** The test carries `skip "agy not installed (CI / fresh machine)"`. On CI (fresh runner) agy is absent → the test skips → counts as `ok`. On a dev box with agy installed, the skip condition is false → the real assertion runs → red. The assertion is a static grep proving setup-windows.ps1 contains the `sharedSkillsDir` / "Synced Shared Skills to" sync that setup-linux.sh has; those strings are currently ABSENT from setup-windows.ps1. So a genuine Windows-parity gap (or a stale test left behind when SDD-008's compile-harness --deploy subsumed the old skills-sync block) is invisible in CI and surfaces only locally.
**Solution:** Treat skip-guarded tests as CI-BLIND: green-in-CI does NOT mean the contract holds when the skip condition is "tool not installed on CI". For a cross-OS-parity-of-SOURCE assertion (static grep of a script's text), there is no reason to skip on a missing runtime tool — gate it on file presence, not on `agy` being installed, so CI actually runs it. Reserve skips for tests that exercise the tool's RUNTIME behavior. When such a local-only red appears, record it as an open thread / Windows-empirical ticket rather than assuming the suite is clean.
**Tags:** `#testing` `#ci` `#skip-guard` `#cross-os-parity` `#false-green` `#windows`
