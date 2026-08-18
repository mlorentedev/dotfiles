---
id: lesson-080-re-running-a-failed-actions-run-replays-the-origin
type: lesson
status: active
created: "2026-06-07"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 080: Re-running a failed Actions run replays the *original commit's* workflow file

**Context:** The "Add to bitácora" workflow had failed repeatedly on `actions/add-to-project@v1` (a tag that does not exist). After pinning `@v1.0.2` on master, the temptation was to re-run the failed runs to confirm the fix.
**Problem:** `gh run rerun <id>` (and the UI "re-run") replays the workflow definition from the **commit the run was originally triggered on**, not from current `main`. Re-running the old failures would have re-executed the broken `@v1` pin and failed again — "proving" nothing and looking like the fix did not work.
**Solution:** Verify a workflow-file fix by triggering a **fresh event** on the patched ref (open/assign a throwaway issue, push a trivial commit), not by re-running historical failures. Delete the throwaway afterward.
**Tags:** `#github-actions` `#ci` `#rerun` `#verify-before-completion`
