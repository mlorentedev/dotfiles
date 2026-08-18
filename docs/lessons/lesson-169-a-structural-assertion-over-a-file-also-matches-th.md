---
id: lesson-169-a-structural-assertion-over-a-file-also-matches-th
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 169: A structural assertion over a file also matches the comments that explain it

**Context**: Pinning the design of two workflows with bats cases — that `add-to-project.yml` classifies rate-limit failures separately, carries no blanket `continue-on-error`, and no longer uses `actions/add-to-project`.

**Problem**: All three assertions failed on first run, and the code was correct. Each workflow's header *explains at length* why those constructs are or are not used, so `grep -c 'continue-on-error'` matched the sentence arguing against it, and `grep -c 'actions/add-to-project'` matched the paragraph explaining why it was removed. Well-documented code is the code most likely to defeat a text assertion about itself — the better the rationale, the more the forbidden strings appear.

**Solution**: Strip comment lines before matching, and assert on *syntactic* forms that only occur in code (`continue-on-error:` with the colon, `uses: actions/add-to-project`) rather than bare names. The red run is what surfaced it; a green-first assertion would have shipped as a test that could never fail.

**Rule**: Same family as the 2026-08-06 plugin audit above — that one judged usage from a capability listing, this one judges structure from a file that documents itself. Both reduce to: **descriptive text about X is not evidence about X**, even when it lives in the artifact itself. When asserting on a file's structure, exclude the commentary first, or key on syntax that prose cannot accidentally produce.
