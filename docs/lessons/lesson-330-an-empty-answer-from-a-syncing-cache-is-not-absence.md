---
id: lesson-330
type: lesson
status: active
created: "2026-10-01"
owner: manu
tags: [lesson, secrets, bitwarden, bw-serve, concurrency]
---

# 330 — An empty answer from a cache that is syncing is not absence

## What happened

Two plain `dotf doctor` runs reported one genuinely missing Bitwarden item. A `dotf doctor --fix` straight after them reported 22 missing items, one for every item the registry names. The same run's DR section said the export "contains no items", and its PAT section said both tokens were "not found". Ten minutes later the daemon listed all 170 items.

`bw serve` answers `GET /list/object/items` with `success:true` and an **empty** list while a forced sync replaces its cache, whichever client started that sync. Measured with one background sync and 40 reads 100 ms apart: between 1 and 5 reads per sync came back empty. Running the readers against the live daemon across three syncs, `origin/main` saw an empty vault in 9 of 120 reads, and the fix saw it in 0 of 120 (BUG-113, #1949).

This is the second time this class has turned up. On 2026-09-29 the `?search=` index was empty while it rebuilt. Moving reads to the unfiltered list fixed that symptom but not the class, because the unfiltered list has the same window.

## Rule

An empty listing from a cache that syncs behind your back is not a fact about the store. Never conclude absence from it. Check every read against the same assumption, not only the endpoint that failed last time. The fix is a single chokepoint that every reader goes through:

- It re-reads an empty answer on a bounded backoff, sized to outlast a whole sync on any machine rather than the window one machine happened to show.
- An answer that stays empty becomes its own error. It must never become an empty success, and never "not found", because a caller reads "not found" as permission to create.
