---
id: lesson-048-vault-patch-timeout-patch-not-applied
type: lesson
status: active
created: "2026-05-20"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 048: vault_patch timeout != patch not applied

**Context:** During WIN-001 (PR #71) and the post-merge vault tick, two `mcp__hive__vault_patch` calls to `10_projects/dotfiles/11-tasks.md` returned `vault_patch timed out after 60s. Server may be under load or a lock is contended; retry shortly.` In BOTH cases the patch had actually committed; the file was already in its new state when the retry ran, and the retry then failed with `patch N: find text not found.` (because the original anchor text no longer existed). Pattern observed twice in the same session.</context>
<parameter name="problem">The naïve retry path on `vault_patch` timeout (call the same patch again with the same find/replace) produces a false `find text not found` failure that masks the fact that the first call succeeded. Wasted ~30s per occurrence verifying after the fact. If the file was not idempotent under the new state, a retry could also corrupt content.
**Problem:** 
**Solution:** On any `vault_patch` timeout, do NOT immediately retry. Instead: (a) `mcp__hive__vault_query` the file to inspect current state, (b) determine if the patch text is already applied (by checking for the post-state content), (c) only retry if the original anchor still exists. This is the same defensive pattern as setup-script idempotence: read-then-write, never blind-write-twice. Treat the Hive timeout response as "outcome unknown" not "outcome failed".
**Tags:** `#hive` `#vault` `#mcp` `#idempotence`
