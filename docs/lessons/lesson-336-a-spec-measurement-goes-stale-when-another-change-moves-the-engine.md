---
id: "lesson-336-a-spec-measurement-goes-stale-when-another-change-moves-the-engine"
type: lesson
status: active
title: "A spec's measurement goes stale when another change moves the engine under it"
created: "2026-10-03"
---

# A spec's measurement goes stale when another change moves the engine under it

## Context
CLI-063's proposal (2026-09-05) measured `dotf deploy`'s merge strategy as a top-level
replace: `mergeInto` wrote each source key whole. From that it planned the expensive part of
the spec: new per-key vocabulary in `ai/deploy.json` (`keys: {env: deep, ...}`), a manifest
version bump to 4, and with it a release that had to ship before the change could merge.
The strict decoder refuses an unknown field, so every installed `dotf` would otherwise stop
deploying.

## The Trap
Between the proposal and the implementation, AI-042's review round 3 made the merge granular:
`deepMerge` recurses into objects and `unionLists` unions lists. That was the exact policy
the proposal said was missing. Nothing linked the two specs. The proposal's table still read
"no — top-level replace loses the box's keys", the manifest's own `$comment` still described
a whole-value replace, and the plan was internally consistent. The increment was about to be
split in two around a release the owner did not want to cut, to add vocabulary the engine
no longer needed.

## The Solution
Re-measure the engine before implementing a spec's increment, not only when it is proposed.
A claim like "the engine cannot do X" in a proposal is a measurement with a date. Before
building around it, run it again against `main`: read the function, or deploy the real
template onto a copy of the real file and diff. Here that took one read of `deepMerge` and one
deploy onto a copy of `~/.claude/settings.json`. The result was one manifest entry with
existing fields, no version bump and no release dependency. The proposal's criteria were
amended with a date rather than worked around silently.
The doc that described the old behaviour (the manifest `$comment`) was corrected in the same
change, because it is what the next reader trusts.
