---
id: lesson-317
type: lesson
status: active
created: "2026-09-29"
owner: manu
tags: [lesson, ci, performance, spec, measurement]
---

# 317 — A speed-up must name the critical path it shortens

## What happened

CI-004 set out to shorten the PR loop. One of its rows, AC9, was written from job durations alone.

AC9 asked for a layer cache that would bring the `integration` build step from 72 s to 20 s or less. The implementation started by reading the layer timings from a real run. Only about 21 s of the 71 s sit before the `COPY` of the repo, which is the part a cache can reuse. The rest runs again on every commit, because it is the test itself: `go build` of the tree (15.9 s), `setup-linux.sh` (26.5 s) and the image export (5.7 s). Neither option the spec offered (a BuildKit GHA cache or a published base image) could reach the bar.

The same run also showed that the bar did not matter. On eleven PR runs that ran the full suite, `integration` finished between 103 and 164 s, and `test-windows` between 401 and 1024 s. Every run ended with `test-windows`. A faster `integration` would have added two third-party actions and saved no PR any time.

AC10 had already been written with this in mind: it ships only if a measurement shows the critical path does not grow, and it was declined on that measurement (`go test` on Windows takes 81–114 s; the `test-windows` p50 is 486 s).

## The rule

Before writing a latency criterion, take two measurements:

1. **The breakdown of the thing being sped up.** Split it into the part the change can reach and the part it cannot. The bar must be reachable by the reachable part alone.
2. **The critical path of what the user waits for.** Find the job that ends the run. A speed-up to any other job saves runner minutes, not waiting time, so state which of the two the row is for.

Write the conditional exit into the criterion from the start, as AC10 did: "ships, or is declined with the measurement". A criterion without it leaves the implementer two choices, either to build something useless or to change the contract.

Refs: CI-004 (#1739), `specs/archive/CI-004-testing-surface-optimization/verification.md`.
