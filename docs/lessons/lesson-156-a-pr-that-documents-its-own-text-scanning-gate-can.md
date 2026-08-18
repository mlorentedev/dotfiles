---
id: lesson-156-a-pr-that-documents-its-own-text-scanning-gate-can
type: lesson
status: active
created: "2026-08-07"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 156: A PR that documents its own text-scanning gate can trip that gate with its own prose

**Context**: #767 introduced `check-spec-gate.sh`'s archive-on-merge check, keyed on GitHub closing-keyword regex over the raw PR body. Its own PR body demonstrated the feature with a worked example (a fenced `$ SDD_PR_BODY='Closes #670' ...` transcript) and, separately, a sentence describing a *future* PR: "...is the next PR — the one that closes #670."

**Problem**: Both fired the very regex they were describing. The fenced example was never meant as a live directive; the prose sentence was about a different PR entirely. Neither is a real closing declaration, but a keyword-anywhere-in-the-body scanner cannot tell narration from directive. CI failed claiming #767 closed an issue it explicitly did not.

**Solution**: Reworded the two spots (a placeholder `#<N>` in the example, rephrased the prose to avoid the verb-adjacent-to-`#N` shape) rather than expanding the regex mid-PR — the code fix (strip fenced blocks, tighten keyword position) is real work deserving its own PR and tests, filed as `dotfiles#773`.

**Rule**: Any regex-over-raw-text enforcement gate is a gate a *documentation PR about that gate* is unusually likely to trip — the PR body is often the first place anyone writes a realistic example of the exact string the gate looks for. When authoring a PR that introduces or documents a text-scanning check, either keep worked examples out of fenced/quoted prose the scanner also reads, or grep your own PR body against the new regex before pushing.
