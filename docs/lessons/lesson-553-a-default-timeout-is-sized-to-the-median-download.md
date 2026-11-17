---
id: "lesson-553-a-default-timeout-is-sized-to-the-median-download"
type: lesson
status: active
title: "A default timeout is sized to the median download"
created: "2026-11-05"
---

# A default timeout is sized to the median download

## Context

A from-zero `dotf vm linux` run installs 41 mise tools in its first converge. mise bounds each download with two
settings. `http_timeout` (30s) restarts every time data arrives, so it catches a stalled connection.
`http_download_timeout` (30m) is the total budget for one download.

## Problem

Over a link that delivered about 85 kB/s, 40 tools installed and argocd (237 MB, the largest pinned asset) failed
at 1800s with 162 MB received. The whole first converge failed on bandwidth, not on what the run was checking (#3181).
A retry would not have helped: mise reported `attempt 1` and never tried again, and a second `mise install`
starts the asset over and hits the same wall. The issue's own option A assumed that a rerun resumes the download.
Nothing measured showed that.

## Solution

Size the total budget to the largest asset at the slowest link you accept, and write that sum next to the value.
The rendered mise config now sets `http_download_timeout = "2h"`, which holds argocd down to about 33 kB/s. This
costs nothing on a dead link, because the stall timeout still fires after 30s. Before you add a retry, check
whether the tool retries that failure class at all, and whether a rerun resumes or starts over. Here a retry
would have doubled the wait and still failed.
