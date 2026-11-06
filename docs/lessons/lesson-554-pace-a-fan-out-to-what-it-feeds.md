---
id: "lesson-554-pace-a-fan-out-to-what-it-feeds"
type: lesson
status: active
title: "Pace a fan-out to what it feeds"
created: "2026-11-06"
---

# Pace a fan-out to what it feeds

## Context

`scripts/bitacora-rollout.sh` pushes the canonical bitácora workflows to every consumer. Where the default
branch is protected it pushes to a rollout PR instead, and every push to a PR starts a PR-Agent review on NaN.

## Problem

The rollout pushed to eight PRs within a few minutes. NaN serves five requests at a time, and a review there
takes 9 to 13 minutes, so three reviews published nothing and their `review` gates went red. It happened on
both rollouts of 2026-11-05, to different repositories each time. Each push was correct by itself. The fault was
the rate: the script is a loop with no notion of the queue its pushes fill, and nothing in it failed.

## Solution

The PR pushes go out in batches below the slot count (`ROLLOUT_BATCH`, 4), with a pause sized to one review
(`ROLLOUT_PAUSE`, 900 s) between batches. Only a repo that actually pushes counts against a batch, so a re-run
over converged repositories does not wait. When a loop's every step starts work somewhere else, size the loop to
that capacity, and pin the pacing with a test that sees where the pause falls.
