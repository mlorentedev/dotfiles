---
id: "lesson-471-a-reviewer-cannot-check-a-digest-it-cannot-recompute"
type: lesson
status: active
title: "A reviewer cannot check a digest it cannot recompute"
created: "2026-10-17"
---

# A reviewer cannot check a digest it cannot recompute

**Context:** Round 1 of the independent review of CLI-131 (2026-10-17) failed on a Blocker: the three contract digests in `review-request.json` "match none of the three files", measured with raw `sha256sum`, a newline-stripped hash and `git hash-object`. Every other acceptance criterion passed.

**Problem:** The launcher records the normalised digest (`ContractDigests`: lifecycle status folded, checkboxes unticked, harness-owned feature fields dropped, CRLF folded). The adversarial-review skill says so in a sentence, but nothing a reviewer can run prints that form, so the only checks open to it are the wrong ones. A reviewer that follows the prose faithfully reaches a false Blocker and the round is spent. The "tree moving mid-review" corroboration in the same finding came from the run's own worktree being armed in parallel, not from the spec.

**Solution:** Dispositioned as a false positive in `verification.md` with the reason, round 1 kept as `review-round1.md`, and round 2 dispatched. The root cause is ticketed (#2684): a read-only `dotf spec digest <id>` that prints the map the launcher records, named in the skill and the launch prompt.

**Why:** A check that depends on an algorithm the checker cannot run is a claim, not a check. When a gate pins a value computed by a normalisation, ship the command that recomputes it next to the value, or reviewers will fail the artifact for the one thing they were never able to verify.
