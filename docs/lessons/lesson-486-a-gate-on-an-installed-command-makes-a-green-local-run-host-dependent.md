---
id: "lesson-486-a-gate-on-an-installed-command-makes-a-green-local-run-host-dependent"
type: lesson
status: active
title: "A gate on an installed command makes a green local run host-dependent"
created: "2026-10-18"
---

# A gate on an installed command makes a green local run host-dependent


**Context:** #2762 gated the harness manifest's three pi targets on `requires_command: pi`, so a box without pi gets nothing under `~/.pi` (#2102). Before the push, the 19 bats files that reference the manifest passed locally (505 ok).

**Problem:** CI's `test` job failed on `skills-pipeline.bats`, "AI-022: pi gets /spec as a native skill". The test deploys into a fake HOME and expects pi's skills there. The developer's Mac has pi on PATH, so the gate let the deploy through. The CI runner has no pi, so the gate skipped it. A second test, "deploy leaves pi-installed sibling symlinks alone", kept passing in CI, but only vacuously: with no deploy into `~/.pi`, nothing could touch the symlink.

**Solution:** The test that expects pi's files puts a fake `pi` on PATH, as the copilot tests already did for `requires_command: copilot` (BUG-771). The symlink test gets the same stub, so it exercises a deploy again. Both runs were repeated with every directory that holds a `pi` removed from PATH: the old test fails there exactly as in CI, and the fixed suite passes.

**Why:** A `requires_command` gate turns the host's PATH into a test input. A green local run on a box that has the command proves nothing about one that does not, and CI is usually the second kind. Once a change adds such a gate, run the suites that cover it a second time with the command removed from PATH. That catches a test that needs a stub and one that has become vacuous.
