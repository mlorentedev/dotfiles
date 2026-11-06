---
id: "lesson-556-measure-a-reviewer-by-what-gets-addressed-not-by-what-it-says"
type: lesson
status: active
title: "Measure a reviewer by what gets addressed, not by what it says"
created: "2026-11-05"
---

# Measure a reviewer by what gets addressed, not by what it says

## Context

The PR-Agent pool draws one reviewer per run by weight (ADR-046). The first re-weight (#2216) needed quality
per member, from a month of reviews across dotfiles, kubelab and hive.

## Problem

Every signal a review carries about itself pointed the wrong way. Haiku posted the most findings (3.6 a
review), the most `Class: REAL` labels and the fewest silent reviews, so it looked like the best member.
PR-Agent's own `RESOLVED` state looked like an addressed rate, and put glm first at 56%. Both were wrong:

- `RESOLVED` means the next run did not raise the finding again. The next run is usually another member,
  drawn at random, so a finding nobody acted on resolves as soon as a different model reviews the PR.
- Findings per review rewards volume. The member that says the most is not the one whose comments change code.

Two other traps cost an hour:

- The comment footer (`Agent run details`) names the model of the LAST run on the PR, because the review
  comment is persistent. Shares and spills per run come only from the run logs.
- The runs API returns an empty `pull_requests` for nearly every run here. Join runs to PRs through
  `head_branch`, not that field.

## Solution

Count a finding as addressed when PR-Agent marked it `RESOLVED` AND its lines changed after the head it was
last seen on: `git diff -U0 <last_seen_head_sha> refs/pull/<n>/head -- <path>`, with a hunk overlapping
`[line_start, line_end]`. The state block on every review comment carries all four fields. Attribute each
finding to the member whose run published at its `first_seen` time, from the run logs.

Measured that way, Haiku's findings were the least often addressed (12% of line-anchored findings, about 0.29
a review) and deepseek's and mimo's the most (about 0.45 a review): the reverse of what the self-reported
signals said.

## Takeaway

A reviewer's output describes itself; only the next commit describes its effect. Rank members by what changed
in the code after they spoke, and fetch `refs/pull/*/head` into a bare clone so the diff is local and costs no
API calls.
