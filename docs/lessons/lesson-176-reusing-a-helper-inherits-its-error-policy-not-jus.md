---
id: lesson-176-reusing-a-helper-inherits-its-error-policy-not-jus
type: lesson
status: active
created: "2026-08-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 176: Reusing a helper inherits its error policy, not just its code

**Context**: Building the advisory adjacency check for `spec-gate` (HARNESS-063) — on a PR that closes an issue, list *other* open issues naming a file in the diff. `check-spec-gate.sh` already had `_strip_markdown_code()`, used by `_closing_issue_numbers` to ignore references inside code blocks. Reaching for it was the obvious move: same script, same job, "find issue references in text".

**Problem**: The two matchers have opposite error economies. `_closing_issue_numbers` strips code spans because GitHub does not close from a reference inside code, so a false *positive* there would demand an archive that GitHub never triggers and block a legitimate merge. Adjacency inverts that: a false *negative* is the entire defect the check exists to catch. The worked example proves it — #849's body cites `scripts/knowledge-crystallize.sh` only inside an inline code span, which the stripper deletes. Reusing the helper would have shipped a check that looked right, passed its tests, and stayed silent on the one issue it was built for.

**Solution**: Match over unstripped title + body, with a comment naming the asymmetry at the reuse site. Then pin it with a test whose fixture mentions the path *only* inside a code span — and verify the pin fires by applying the mutation: with `_strip_markdown_code` inserted, that case goes red while the primary case stays green, because #849's title happens to carry the bare basename unformatted.

**Rule**: Before reusing a helper, ask which error it was tuned to avoid, not just what it computes. A function is a decision about which mistake is cheaper, and that decision does not travel with the call site. When the answer inverts, write a second matcher and say why in a comment — then prove the guard fires by mutating the code it forbids, because a comment forbidding a refactor is not a check, and the test that would catch it may pass for an unrelated reason.
