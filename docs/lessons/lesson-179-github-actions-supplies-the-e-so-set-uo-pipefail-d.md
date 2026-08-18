---
id: lesson-179-github-actions-supplies-the-e-so-set-uo-pipefail-d
type: lesson
status: active
created: "2026-08-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 179: GitHub Actions supplies the `-e`, so `set -uo pipefail` disables nothing

**Context**: `bitacora-reconcile.yml` (OPS-023) is the daily healer for board items the event-driven add drops. Its `run:` block was written with deliberate error handling: capture the rollout output, classify it, soft-pass green on a rate limit because the healer hitting the limit it heals is expected, and file a deduplicated issue on anything else because a backstop failing silently is the exact fault the ticket exists to remove. The first line reads `set -uo pipefail   # deliberately not -e: the classification below is the error handling`.

**Problem**: Actions does not run a `run:` block as `bash {0}` — it runs it as `bash -e {0}`, visible in every job log's `shell:` line. The `-e` arrives on the *invocation*, and `set -uo pipefail` does not clear an already-active `-e`; only `set +e` does. So `out=$(./scripts/bitacora-rollout.sh --backfill-only ...)` aborted the step on that line the instant the rollout exited non-zero, and every branch below it was unreachable dead code. Two scheduled runs went red and silent (2026-08-08, 2026-08-09): no output, no issue filed. The damage was second-order and worse than the outage — the abort happened *before* `printf '%s\n' "$out"`, so the rollout's own diagnostics were destroyed unprinted, and the underlying cause became unknowable from the log. The self-reporting was designed and correct and never got to run; the absence of the issue it promised to file is what made the failure look like a transient.

**Solution**: Extract the classification into `scripts/bitacora-reconcile.sh` and leave the workflow a one-line invocation, so the injected `-e` governs only that call, where exit 0/1 already means green/red. Inside the script use `rc=0; out=$(...) || rc=$?` rather than a bare capture, so the line is correct under any flag state a caller imposes, and print `$out` unconditionally *before* classifying. Then pin it with a bats suite that executes the classifier under `bash -e` — the same three cases invoked without `-e` pass against the pre-fix logic and fail with it, which is the whole proof.

**Rule**: When a harness invokes your shell, the flags it chose are part of the contract and your `set` line cannot revoke them — check the `shell:` line in the log rather than assuming. Prefer making a snippet flag-independent (`|| rc=$?`) over asserting a flag state, and prefer a script the runner calls in one line over logic embedded in a `run:` block, because embedded logic cannot be executed by a test and only the executable form catches this. Above all: a comment asserting a property (*"deliberately not -e"*) is not that property. This one had a test-shaped hole exactly where a test would have caught it — and note the ordering, because it recurs: an error handler that dies before printing its evidence removes the very diagnosis its failure demands.

**Postscript**: the first cut of the fix reproduced the bug inside its own reporting path — the dedupe lookup was a bare `existing=$(gh issue list ...)`, so a failing `gh` aborted the script under `-e` before the `::error::` and before anything was filed. Review caught it; the suite did not, because every stubbed `gh` *succeeded*. Two things generalise. Knowing a failure mode is not the same as having eliminated it: the same shape can survive three lines below the line you just fixed, because attention anchors on the instance rather than the pattern — so after fixing one, grep the file for every other bare `x=$(...)` before calling it done. And a stub that only models the happy path cannot test error handling at all; the fix-path tests here were the ones with no failing-dependency case, which is exactly where the bug hid.
