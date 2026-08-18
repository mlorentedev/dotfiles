---
id: lesson-183-a-real-dependency-test-can-rest-on-an-undeclared-e
type: lesson
status: active
created: "2026-08-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 183: A real-dependency test can rest on an undeclared environment precondition, and then it tests one thing locally and another in CI

**Context**: BUG-066 added `tests/spec-gate-pr-real.bats`, the real-`gh` sibling that `tests/stub-real-pairing.bats` requires of any suite stubbing a binary. Its core assertion is that `gh` accepts the `--json` field names the adapter asks for — the one contract a stub can never check, since a stub answers whatever it is asked.

**Problem**: It passed locally and failed in CI on its first run. `gh pr view --json <fields>` validates field names before it resolves a repository, but only once past the auth check — and an **unauthenticated** gh exits at that auth check first. The developer machine is logged in, so the assertions were reached; the bats job has no token, so they never were. Same file, same command, two different tests. The suite rested on an environment precondition it never declared — the same family as the defect it was written to catch: verified under conditions nobody stated.

**Solution**: Declare the precondition and enforce it where it matters. `gh auth status` is checked in `setup`, and a missing precondition is **fatal under CI** while remaining a skip on a dev machine without gh auth — a real-dependency suite that quietly skips is a green proving nothing (BUG-055, #807). CI supplies the token as `DOTF_TEST_GH_TOKEN`, not `GH_TOKEN`, and only this suite promotes it, so no other suite's auth state changes and none can begin making real API calls as a side effect. Verified in all three modes before merging — authenticated (runs), `CI=1` without auth (fails loudly, naming the reason), unauthenticated dev box (skips) — and the CI log shows the three cases running rather than skipping.

**Rule**: Every test carries preconditions; the dangerous ones are those you never wrote down, because your machine satisfies them silently and CI does not. When a test drives a real external tool, name what it needs — auth, network, a binary, a writable path — and assert it in `setup`. Then treat the two environments as different decisions: a skip is acceptable on a developer machine and is a lie in CI, so gate the skip on `$CI` and let the pipeline fail loudly. Corollary on the plumbing: when CI must supply a credential for one suite, pass it under a name only that suite reads, never the tool's canonical variable — otherwise every other suite silently changes behaviour, and a test that used to stop at an auth error starts making real calls.
