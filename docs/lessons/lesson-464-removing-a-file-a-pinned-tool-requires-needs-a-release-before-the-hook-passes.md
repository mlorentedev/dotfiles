---
id: "lesson-464-removing-a-file-a-pinned-tool-requires-needs-a-release-before-the-hook-passes"
type: lesson
status: active
title: "Removing a file a pinned tool requires needs a release before the hook passes"
created: "2026-10-17"
---

# Removing a file a pinned tool requires needs a release before the hook passes


**Context:** CLI-131 (#2554) stops committing `docs/lessons/_index.md`: two PRs that each add a lesson conflicted on the generated table, and under a merge queue a conflict ejects the PR. The change deletes the file and teaches `dotf lessons fmt` to need none.

**Problem:** The pre-commit hook runs `dotf lessons fmt --check` with the installed `dotf`, the pinned release (`DOTF_VERSION`), not the checkout. That release still fails with "has lessons but no _index.md". CI is not affected, because it builds `dotf` from the checkout with `go run`. So the commit that deletes the index, and every later commit that touches `docs/lessons/` on a host that has not updated, fails the local hook until a release carrying the change is installed. Lane prompts add a lesson per bug, so every lane would hit it.

**Solution:** Know the window before merging and bound it: the deleting commit skips the one hook (`SKIP=lessons-fmt`, disclosed in the PR), and the PR names the release as the step that ends the window. CI stays the gate in the meantime. The check that would catch a repo still committing an index (`fmt --check` naming it) ships in the same change, so the removal is loud on the repos that scaffolded one rather than silent.

**Why:** A hook pinned to a released binary and a CI job that builds from the checkout are two different versions of the same check. A change that makes the new tree invalid for the old binary is a change in two steps (merge, release) even when it is one PR. Look at which binary each gate runs before deleting what that binary requires.
