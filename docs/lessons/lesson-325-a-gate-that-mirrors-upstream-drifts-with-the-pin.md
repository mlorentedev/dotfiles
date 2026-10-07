---
id: "lesson-325-a-gate-that-mirrors-upstream-drifts-with-the-pin"
type: lesson
status: active
title: "A gate that mirrors an upstream tool drifts with the pin, and trusts whatever the tool quotes"
created: "2026-09-30"
owner: manu
tags: [lesson, pr-agent, ci, security]
---

# A gate that mirrors an upstream tool drifts with the pin, and trusts whatever the tool quotes

## What happened

`scripts/pr-agent-push-gate.sh` decides whether a push gets a PR-Agent review by re-implementing how PR-Agent finds its previous review and counts new commits. Two archive reviews of TOOL-023 each found a defect in that copy.

1. **The copy described a version the workflow no longer ran.** The gate was written against v0.45.0, which counts commits by author date. The workflow pins v0.46.0, which counts by committer date. A rebase keeps author dates, so after one the gate counted 0 new commits and returned `run=false`. That skipped PR-Agent and the "no review published" guard behind it (#1893).
2. **The copy trusted text the tool had quoted.** The gate read `last_run.head_sha` from a `pr-agent-review-state` block in any bot review. PR-Agent appends that block only to a full review, as its last element. An incremental review has none of its own, so a block inside one is text the review quoted, and a PR author can supply it. A forged `head_sha` naming their newest commit made the count 0.

## Rule

- When a script re-implements an upstream tool's behaviour, name the upstream version in the script, and add a test that fails when the workflow's pin and the version the script cites disagree. A bump then breaks the build instead of the behaviour.
- Read a machine-written block only in the position the tool writes it (here: a full review, last in the body). Treat the same text anywhere else as content the tool echoed, which may come from the PR author.
- A bot author check (CWE-345) proves who posted the comment, not who wrote every line in it.

## References

- `tests/pr-agent-push-gate.bats`: "every PR-Agent version the gate cites is the one the workflow pins", "a state block inside an incremental review is quoted text, not the baseline"
- `specs/archive/TOOL-023-pr-agent-incremental-push-review/review-round-1.md`, `review.md`
- #1893, #1895
