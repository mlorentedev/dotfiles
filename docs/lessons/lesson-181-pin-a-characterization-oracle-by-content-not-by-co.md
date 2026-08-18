---
id: lesson-181-pin-a-characterization-oracle-by-content-not-by-co
type: lesson
status: active
created: "2026-08-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 181: Pin a characterization oracle by content, not by commit — a SHA answers the wrong question

**Context**: The crystallize golden corpus (CLI-021) records which revision of `knowledge-crystallize.{sh,ps1}` produced its expected bytes, and a test fails the suite when the tree drifts off that revision, so a recapture must be a deliberate act rather than a silent regeneration that turns a red golden green. The obvious implementation was `git log -1 --format=%H -- <path>` at capture, compared against the same at test time.

**Problem**: It passed locally and failed in CI on a corpus that was perfectly valid. `actions/checkout` is shallow by default (`fetch-depth: 1`), so there is exactly one commit in the history and `git log -1 -- <path>` resolves to that synthetic merge commit for **every** file — the check reported the oracle as moved when not a byte had changed. The deeper error was choosing the wrong identity: a commit SHA answers *"which revision"*, but the question the corpus actually asks is *"are these the bytes I captured from"*. Those come apart in both directions — a rebase rewrites the SHA with identical content (so this would also have gone red on the next rebase of its own branch), and in principle a SHA can be reachable while the working tree is dirty.

**Solution**: Assert on `sha256` of each twin. It needs no history, survives rebases and squashes, and is exactly the property being claimed. The git revision stays in the `ORACLE` file as a comment for human readers — informational, not load-bearing. Verified as a detector, not a tautology: appending one byte to the `.sh` turns the check red, restoring it turns it green, and the test no longer invokes `git` at all.

**Rule**: When a check asserts "this artifact still matches that source", key it on a hash of the source's **content**, never on a VCS identifier. VCS identity is metadata about history, and CI routinely runs with history truncated, detached, or rewritten — so a history-derived assertion has an environment dependency its author did not intend and will not see locally. The general form: before writing any equality check, ask which of the two things you actually mean — *same bytes* or *same commit* — because they diverge under rebase, squash and shallow clone, and only one of them is what a characterization corpus is claiming. Corollary from the same session: a guard whose failure mode is a false positive in CI trains people to bypass it, so its correctness matters as much as its existence.
