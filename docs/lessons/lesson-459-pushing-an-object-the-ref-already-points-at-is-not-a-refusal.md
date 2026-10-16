---
id: "lesson-459-pushing-an-object-the-ref-already-points-at-is-not-a-refusal"
type: lesson
status: active
title: "Pushing an object the ref already points at is not a refusal"
created: "2026-10-16"
owner: manu
tags: [lane, claim, git, concurrency]
---

# Pushing an object the ref already points at is not a refusal

**Context:** CLI-128 (#2551) makes the lane claim a push of an empty-tree commit to `refs/dotf/claims/<issue>` with `--force-with-lease=<ref>:` (an empty expected value, so the ref must not exist). The ref form could not be probed against the real `origin` (the push was denied), so the behaviour was pinned by tests over a real local bare remote.

**Problem:** The first version of the test failed: a second `Acquire` by the same holder succeeded. A claim commit is deterministic (same tree, same message, same author, same second), so the second claim was the very object the ref already pointed at. Git treats that push as "everything up-to-date" and exits 0 before the lease is ever compared. Two sessions run by the same user in the same second would both have won.

**Solution:** Every claim commit carries a random nonce, so a second claim is a different object and the empty-expected lease refuses it. `TestClaimAcquire` asserts the same-holder case on purpose, and `TestClaimConcurrent` runs eight claimants at one remote 20 times.

**Why:** A compare-and-swap whose "new value" can equal the current value is not a CAS: idempotence on the transport side reads as success on yours. When a primitive's guarantee comes from the server's refusal, build the payload so that two callers can never send the same one, and test the same-caller case, not only the other-caller case.
