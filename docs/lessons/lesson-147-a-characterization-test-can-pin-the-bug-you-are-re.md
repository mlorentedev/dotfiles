---
id: lesson-147-a-characterization-test-can-pin-the-bug-you-are-re
type: lesson
status: active
created: "2026-07-14"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 147: A characterization test can pin the bug you are removing — grep every test extension, not just the source

**Context**: BUG-031 (#689) fixed the Windows Claude project-key encoding by deleting a local `Get-EncodedPath` (which mapped the drive `:` to `''`, the bug) and routing through a shared `dotf`-backed helper. Before pushing I grepped the repo for `Get-EncodedPath` — but only across `*.ps1`. Local Go tests and the Pester guard were green, so the PR looked done.

**Problem**: CI's `test` and `test-windows` jobs failed on two bats cases in `tests/knowledge-crystallize-ps1.bats`: one asserted `grep -q 'function Get-EncodedPath'` (the function I had just deleted), the other asserted `grep -q "Replace.*':'.*''"` (the exact colon-deleting pattern that WAS the bug). These were characterization tests written against the original behavior, so a correct fix flipped them red. My `*.ps1`-only grep never saw them because the stale assertions lived in a `.bats` file, and the local Go + Pester suites did not include that bats file.

**Solution**: Re-point both bats cases at the corrected reality — assert the script sources `utils.ps1` and uses `Get-ClaudeProjectKey` with no local encoder, and invert the colon test to assert the buggy `Replace ':' ''` is **absent** and the decoder expects the double-dash key. Confirmed by running `bats tests/knowledge-crystallize-ps1.bats` locally (16/16) before pushing the follow-up commit; CI then green.

**Rule**: When a fix removes or renames a symbol, or changes an observable string, grep the **whole test tree across every extension** (`*.bats`, `*.Tests.ps1`, `*_test.go`) for the old name/pattern before pushing — a green local run only covers the suites you actually ran, and a characterization test that encoded the old behavior will fail precisely *because* the fix is correct. Treat a test that asserts a bug's fingerprint (a specific buggy pattern) as a liability: when you kill the bug, invert the test to guard against its return in the same change.
