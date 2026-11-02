---
id: "lesson-531-a-fixture-copied-from-upstream-keeps-passing-after-upstream-moves"
type: lesson
status: active
title: "A fixture copied from upstream keeps passing after upstream moves"
created: "2026-11-01"
---

# A fixture copied from upstream keeps passing after upstream moves

## Context

The `reviewable` step of `pr-agent.yml` read PR-Agent's lockfile list from `language_handler.py` at the audited
commit, so that nothing was a copy that could drift (#1417). Its bats cases fed it a short handler written by
hand in the shape `is_valid_file` had then.

## Problem

Upstream moved the list to a module constant, `AUTO_GENERATED_FILES_EXACT = frozenset({...})`. The step looked up
the old lowercase name, and `literal_eval` refuses a call anyway, so the list read empty at the audited commit. A
`go.sum`-only PR ran PR-Agent on an empty diff again and the guard went red. Every test stayed green, because the
fixture kept the old shape. The audit did not catch it either: the contract pinned `file_filter.py` and not the
two files the step actually reads. The failure narrowed the skip, which is the safe direction, so nothing said so.

## Solution

`dotf pr reviewable` (#3016) reads the real shape, and its tests read upstream's real files from
`cli/internal/prreview/testdata/pr-agent/`. `TestUpstreamFixtures_AreTheAuditedBlobs` holds those files to the git
blob SHAs in `harness/pr-agent-upstream-contract.json`, which now lists both, so an image bump cannot leave them
behind. A list that reads empty raises a `::warning::`.

## Why

A fixture written by hand tests the reader against the author's memory of the format. When the format belongs to
someone else, feed the test the real artifact and bind it to the pin. A fallback that fails safe still needs a
voice: "narrows, never widens" is why this was correct and also why it went unseen.
