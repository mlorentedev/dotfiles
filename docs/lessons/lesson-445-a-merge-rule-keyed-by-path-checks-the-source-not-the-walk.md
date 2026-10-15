---
id: "lesson-445-a-merge-rule-keyed-by-path-checks-the-source-not-the-walk"
type: lesson
status: active
title: "A merge rule keyed by path is checked on the source, not where the walk reaches it"
created: "2026-10-15"
owner: manu
tags: [deploy, merge, toml, validation]
---

# A merge rule keyed by path is checked on the source, not where the walk reaches it

**Context:** The deploy merge unioned lists by whole-element equality, so editing one field of a managed `[[keys.command]]` table in `ai/herdr/config.toml` left the old table beside the new one: two bindings on one key (#2455). Manifest version 5 adds `listKeys`, a path such as `keys.command` mapped to the field that identifies each table, and the merge replaces the element with the same identity instead.

**Problem:** The first version validated the managed list inside `deepMerge`, at the moment the recursive walk reached `keys.command`. Every test with a destination that already held `[keys]` passed. With an empty or absent destination the walk never got there: `keys` is missing, so the whole managed table is copied in one assignment and the list under it is never visited. A managed binding without its `key` field, or two sharing one, was written silently on a fresh machine, the case that matters most. A second gap: the walk builds paths from real keys, so a JSON key `"a.b"` reached the identity merge under the path `a.b` while the up-front lookup split it as nesting and found nothing. The merge then ran on unchecked elements and asserted them to tables, a panic.

**Solution:** `mergeInto` looks every `listKeys` path up in the source and runs `checkIdentities` before reading the destination, so the destination's shape cannot decide whether the rule is enforced. `deepMerge` runs the same check where it applies the rule, and `identityIndex` uses a checked type assertion. `TestDeploy_MergeRefusesAManagedElementWithoutItsIdentity` deploys into an empty home, which is what caught the first gap, and `TestMergeInto_ListKeysOnADottedKeyIsRefused` pins the second. Eleven mutations, each removing one guard, all fail the suite.

**Why:** A recursive merge visits only the places where both sides have structure. Where the destination lacks a parent, the walk short-circuits, and any rule attached to a deeper path goes unenforced. Validate what you are about to write against the source alone, before the destination can choose the code path; and when a rule is addressed by a path, make sure every reader computes that path the same way, or the gap between them is where a check silently stops applying.
