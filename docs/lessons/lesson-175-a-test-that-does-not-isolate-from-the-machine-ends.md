---
id: lesson-175-a-test-that-does-not-isolate-from-the-machine-ends
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 175: A test that does not isolate from the machine ends up measuring the machine

**Context**: `tests/board-pickup.bats` failed on a clean `main` with no local changes: "assigns once, no redundant fallback" saw two log entries where it expected one. The obvious reading — and the one first published on the issue — was that the double-assignment bug it guards against was live.

**Problem**: The helper was correct. GUARD-001 wires `core.hooksPath` globally, so the fixture's own `git checkout` fired the machine's real `post-checkout` dispatcher, which launches the helper under test in the background. The assertion counted the machine's hook plus the explicit call. Worse, the leak is racy — the dispatcher backgrounds the helper, so whether the second line lands before the assertion is a timing question, which is how a test like this passes locally and fails elsewhere. A sibling suite already defended against exactly this and said so in a comment; this file never got the same treatment.

**Solution**: Neutralise `core.hooksPath` in the fixture's own config rather than per call, so all thirteen git invocations are covered along with any added later. Add a guard for the isolation itself — a bare checkout must leave the log empty — with a `sleep`, since an immediate assertion could pass on timing rather than on isolation.

**Rule**: Any suite that shells out to `git` inherits the developer's global git config, and a machine-wide `core.hooksPath` turns "run a git command" into "run the product". Isolate at the fixture level, not the call site, and add an assertion that the isolation holds — otherwise the isolation is a convention that decays the moment someone adds a fourteenth call. And when a test contradicts the code, resolve which one is wrong before reporting it as evidence: a red test is a claim about the test *and* the code together.
