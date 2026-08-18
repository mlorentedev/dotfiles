---
id: lesson-184-two-clients-one-resource-the-green-path-and-the-re
type: lesson
status: active
created: "2026-08-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 184: Two clients, one resource: the green path and the red path shared a credential, so the credential was never the answer

**Context**: #884. `bitacora-reconcile.yml` had failed every run since it shipped — three for three — with `unknown owner type` on every item in every repo. The obvious readings were a dead token or a rate limit, and both were wrong.

**Problem**: `bitacora-rollout.sh` wrote to the board with `gh project item-add --owner`, which makes the gh CLI resolve the owner's *type* before it does anything, probing the `user` and `organization` nodes. `BITACORA_PAT` cannot perform that lookup. What made this hard to see is that the board was demonstrably writable the whole time: the event-driven `add-to-project.yml` was green, using the **same secret**, because its `addProjectV2ItemById` mutation is handed a project node ID and never resolves an owner. The reconciler had inherited its invocation form from the provisioning path, which runs locally under a different credential — so the form had never once been exercised against the credential it actually runs under.

**Solution**: repoint the back-fill at the mutation the other path already proves works. The node IDs were free — `gh issue list --json id,url` returns them from a listing that had to happen anyway. Fixing it in code beat widening the PAT's scopes: nothing else needs owner-type resolution, so granting a scope for one CLI call would enlarge a leaked token's blast radius for no gain.

**Rule**: When one code path fails against a shared resource and another succeeds, the credential is not the variable — stop testing it and diff the two call paths instead. The corollary is about where these bugs come from: a call form copied from a context with different auth is untested code even though it is running in production, and it will keep working locally for whoever copied it. Ask of any borrowed invocation *which credential has actually executed this shape*, and treat "a different one" as untested. Note also what a stub cannot do here — `tests/bitacora-rollout.bats` pins which call is made, and nothing more; a stub accepts any form, so the API's verdict on that form only ever arrives from a real run.
