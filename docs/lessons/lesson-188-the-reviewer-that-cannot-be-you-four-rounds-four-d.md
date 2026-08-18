---
id: lesson-188-the-reviewer-that-cannot-be-you-four-rounds-four-d
type: lesson
status: active
created: "2026-08-11"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 188: The reviewer that cannot be you: four rounds, four defects, three of them in the fix for the last one

**Context**: DOCS-013 (#916, PRs #922/#924) added a guard that fails CI when an instruction file names a repo path that no longer resolves. It was written after `.claude/CLAUDE.md` was tracked into the repo carrying seven dead file references, two of which sessions had already acted on. The change was reviewed by CodeRabbit and by the implementing session's own tests, then by three independent adversarial-review rounds run as fresh subagents.

**Problem**: every round found real, reproducible defects in code the previous round had examined and passed. Round 1 found the one that matters most: the new *Verification Commands* section documented `go install …@v$(. versions.conf; echo "$GOLANGCI_LINT_VERSION")`, which under zsh resolves the version to **empty** — a slashless argument to the `.` builtin is searched on `$PATH` only, and unlike bash, zsh does not fall back to the cwd. A wrong instruction, shipped in the change whose entire purpose was to stop instructions being wrong, by the bash/zsh divergence class documented twenty lines above it in the same file. Neither the author nor CodeRabbit saw it.

Round 2 then found three more, none inherited: the `..`-containment fix from round 1 had been ordered *before* the "is this token ours to judge" gate, so fixing a false negative introduced a false positive; the regression test guarding the zsh bug used a regex requiring a delimiter before `. file`, making it blind to the flush-left form — the canonical shape of the very bug it guarded; and two instruction files were governed by nothing. Round 3 found a third ungoverned file. Round 2 also caught a `verification.md` claim that a specific mutation turned a test red: it did not, because the claim was written from intent rather than from a run.

**Solution**: fix each finding, but treat the third occurrence of a class as evidence about the mechanism rather than the instance. Round 3's proposed fix was one line — add the missing file to a list. The chain's own record (three rounds, three different files missing from a hand-maintained list) argued against it, so the list was replaced with discovery from the git index plus a probe case that stages a temporary instruction file and asserts the guard fails on it. The governed set went 8 → 9 without anyone naming the ninth.

**Rule**: an implementer cannot review their own change, and the reason is structural rather than a matter of diligence — you verify what you thought about, and the defects live in what you did not. Brief an independent reviewer with the feature id and the repo only; do not hand it your rationale, and instruct it to read the diff before the justification, or it checks whether the code matches your story instead of whether the story is right. Three things make the difference between a review and a rubber stamp: require mutation (a test that passes before *and* after a fix discriminates nothing), tell it the prior findings are the floor of what is wrong and never the ceiling, and re-run every claim the verification document makes rather than reading it. And when a class of defect recurs a third time, stop fixing instances — the recurrence is telling you the mechanism is hand-maintained and will rot again.

**Tags**: `review`, `shell`, `zsh`, `testing`, `sdd`
