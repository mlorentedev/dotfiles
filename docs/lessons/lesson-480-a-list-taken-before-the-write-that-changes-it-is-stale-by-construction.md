---
id: "lesson-480-a-list-taken-before-the-write-that-changes-it-is-stale-by-construction"
type: lesson
status: active
title: "A list taken before the write that changes it is stale by construction"
created: "2026-10-18"
---

# A list taken before the write that changes it is stale by construction


**Context:** #2677. A converge from a tree that does not pin `zsh-autosuggestions` failed with `after mise prune, still installed: http:zsh-autosuggestions@0.7.1`. The issue named two candidates, both in mise: the prune not matching the `http:` spec, or `mise prune` and `mise ls --prunable` applying different rules.

**Problem:** Neither was it. Run in an isolated mise (`MISE_DATA_DIR` and the other three state dirs in a scratch directory, with a copy of the install), `mise prune --yes --tools http:zsh-autosuggestions` removed the file tool and the listing came back empty. The transcripts held the answer. One failing run never called `mise prune`; the other pruned `gh` only. The orphan list came from the plan, taken while the deployed config still asked for the tool, so mise did not list it as prunable. The apply then wrote the config that dropped it, and only the check after the prune listed it.

**Solution:** After a config write, list the orphans again before pruning, as the step already re-asked `mise outdated` after the same write. Keep the earlier list when the second listing fails, since those are still orphans of the same pins, and let the check after the prune report the failure. The fake mise models the cause: an install the config at `cfg` names is not prunable until a write drops it.

**Why:** The orphan set is a function of the config, and the apply changes the config between the listing and its use. Any plan computed from state that the apply then rewrites goes stale at that write. A step that writes and then acts must re-read what the write can change. The first clue was in the transcript, not the error: a prune line naming one tool, and a failure naming another.
