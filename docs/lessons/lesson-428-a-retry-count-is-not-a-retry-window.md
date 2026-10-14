---
id: "lesson-428-a-retry-count-is-not-a-retry-window"
type: lesson
status: active
title: "A retry count is not a retry window"
created: "2026-10-14"
owner: manu
tags: [ci, curl, retry, github]
---

# A retry count is not a retry window

**Context:** Every CI download was `curl -fsSL --retry 3 --retry-all-errors … -o file`, and `tests/ci-download-retry.bats` checked that `--retry` was there. On 2026-10-14 `test-macos` on #2479 still failed on the age release download: GitHub answered HTTP 500 four times in 7.3 seconds, then curl gave up. The rerun passed.

**Problem:** `--retry 3` says how many times curl asks, not for how long. Without `--retry-delay`, curl waits 1, 2 and 4 seconds, so the whole budget is about 7 seconds, and a GitHub 5xx lasts longer than that. The guard checked for the flag, so it passed on a retry that could not absorb the failure it was there for.

**Solution:** `--retry 5 --retry-delay 10 --retry-all-errors` on every CI download, in `ci.yml` and `tests/Dockerfile.integration`. `--retry-delay` is a fixed wait. Measured against a local server that always answers 500: `--retry 3 --retry-delay 2` took 6 s over 4 requests, and the new setting took 50 s over 6. The guard now requires the whole string, so a shorter window fails it.

**Why:** A retry is sized by the outage it must outlast. State the window in seconds, measure it against a server that fails on purpose, and make the guard check the window, not just the presence of the flag.
