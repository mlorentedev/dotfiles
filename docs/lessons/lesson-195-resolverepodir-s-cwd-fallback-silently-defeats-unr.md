---
id: lesson-195-resolverepodir-s-cwd-fallback-silently-defeats-unr
type: lesson
status: active
created: "2026-08-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 195: `resolveRepoDir`'s cwd fallback silently defeats "unresolvable repo" test cases

**Context**: HARNESS-070, writing Go unit tests for two new doctor checks (`checkHarnessMirrorOrphans`, `checkInstructionDrift`) that both call `resolveRepoDir(sys)` to find the repo checkout. Both checks' contracts include "repo not found → SKIP, not FAIL," so each got a test setting `DOTFILES_REPO_DIR` to a non-existent path to exercise that branch.

**Problem**: `resolveRepoDir` falls back to `os.Getwd()` + walking up for a `.git` when the env var doesn't resolve — and during `go test` execution, the process's cwd is somewhere inside the real repo checkout (`cli/internal/doctor`), which genuinely has `.git` a few levels up. So "point `DOTFILES_REPO_DIR` at a path that doesn't exist" does not produce an unresolvable repo in this test harness — it produces the *real* dotfiles checkout, silently, via the fallback the test author forgot was there. The first version of both tests asserted on a SKIP message that never fired for the reason intended; one of the two happened to still pass (by accident — the real repo also lacked the fixture directories being compared, so the check bottomed out at a *different* SKIP with the same failure/pass shape), the other failed loudly with a mismatched substring, which is what caught it.

**Solution**: for `checkHarnessMirrorOrphans`, dropped the "unresolvable repo" subtest entirely — every other subtest already sets a valid `DOTFILES_REPO_DIR` deterministically, and the unresolvable branch is a one-line early return not worth chasing across environments. For `checkInstructionDrift`, kept a dedicated test but made it self-skipping: it calls `resolveRepoDir(sys)` itself first and `t.Skip`s if that resolves to a real path, so the test is honest about being environment-dependent instead of silently testing the wrong thing.

**Rule**: any check consuming `resolveRepoDir` (or anything else with a cwd/environment-walking fallback) cannot have its "nothing resolved" branch tested by merely pointing the primary env var somewhere invalid — the fallback will paper over it inside a real checkout, which is exactly where `go test` always runs. Either avoid exercising that branch directly (compose it from cases that are deterministic some other way), or probe the seam directly in the test and skip when the ambient environment doesn't cooperate, rather than asserting a specific SKIP message and hoping.

**Tags**: `testing`, `go`, `harness`, `test-isolation`
