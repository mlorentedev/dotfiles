---
id: "lesson-483-a-grant-list-no-code-reads-is-a-promise-not-a-rule"
type: lesson
status: active
title: "A grant list no code reads is a promise, not a rule"
created: "2026-10-17"
---

# A grant list no code reads is a promise, not a rule


**Context:** kubelab#2431 asked to add promotion PRs to the `excluded:` list of kubelab's `.github/merge-grant.yml`, so an agent could not merge a PR that deploys to prod under the standing grant. The file's header said a PR that changes the grant is never merged under it.

**Problem:** `dotf pr land` read neither promise. It asked the contents API whether the file existed and ignored the body, and it refused only `release-please--` branches, from a constant in the code. Adding `branch:deploy/` to the YAML would have changed nothing, and a PR that edited the grant itself landed like any other. Both repositories' files looked like policy. A test pinned dotfiles' grant word for word, which made the file look even more enforced than it was.

**Solution:** `readGrant` decodes the file from the base branch and Decide applies `excluded:`. `release-please` keeps its hardcoded check. `branch:<prefix>` forfeits the merge on a head-branch prefix. `path:<file or dir/>` forfeits it on any PR that changes that path, whatever its branch is called. A kind it does not know refuses, and so does a file that does not decode. Any PR whose file list contains the grant file is refused, and so is a PR whose list gh truncated (100 paths), because a partial list cannot prove the grant is absent. A test asserts that no branch name lands a PR on a base without the grant.

**Why:** A declaration is policy only where the actor reads it, and here the actor read only the file's existence. Pinning the file's text in a test proves the text, not the behaviour. An unknown exclusion kind has to refuse: if the actor skipped it, the grant would say more than the actor enforces.
