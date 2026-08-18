---
id: lesson-165-the-defensive-half-of-a-fix-is-the-least-reviewed-
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 165: The defensive half of a fix is the least-reviewed code in the PR, and its failures are silent by construction

**Context**: Two fixes in one session, both to guards, both with a hole in the code added to *protect* the guard rather than in the guard itself. `#805` resolved a worktree's hook dir via `git rev-parse --git-common-dir` and validated the answer with `[ -d "$common_dir" ]`. `#814` excluded quoted agent markers by stripping inline code spans with `` `+[^`]*`+ ``.

**Problem**: Each defence had exactly one input that walked straight through it. In `#805` an empty probe result was joined into `"$toplevel/"` — and the worktree root **always** is a directory, so the one answer that most needed rejecting was the one the directory test waved through, resolving hooks under `<toplevel>//hooks`. In `#814` the pattern accepted *unequal* backtick runs, so `` `[AGENT-DRAFT]`` `` — which CommonMark does not treat as a code span at all — was stripped, hiding a live marker. Both holes reintroduced precisely the silent failure their PR existed to remove. Neither was found by the author; both came from review.

**Solution**: `#805` requires the resolved path to be non-empty *as well as* a directory, and leaves an empty probe empty through the join rather than turning it into something that looks valid. `#814` replaced the regexp with a scanner that requires a closing run of exactly the opening length — Go's RE2 has no backreferences, so the balanced-delimiter rule **cannot be written as a regexp at all**, and reaching for one was itself the tell. Both now err toward refusing rather than passing.

**Rule**: When a fix adds validation to protect its own new logic, review that validation harder than the logic — it is the part written last, tested least, and the only part whose failure is silent by definition. Two questions catch most of it: *what is the degenerate input* (empty string, zero, unterminated delimiter), and *does the check accidentally hold for it?* A path test passes for `"$root/"`, a length check passes for `0`, a "looks like a span" regex passes for a malformed span. And when the correct rule cannot be expressed in the tool you reached for, that is evidence the shortcut is wrong, not that the rule needs relaxing.
