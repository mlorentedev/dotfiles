---
id: lesson-182-a-check-that-never-ran-and-an-escape-that-was-neve
type: lesson
status: active
created: "2026-08-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 182: A check that never ran and an escape that was never taken are the same defect: verified in isolation, never exercised in situ

**Context**: BUG-066. The SDD spec-gate's documented escape — a `skip-archive` label plus a non-empty `## Archive skip rationale` section in the PR body — was applied correctly on #877 and never went green; that PR merged with the check red. Separately, HARNESS-063's adjacency report had been fixture-tested and green since the day it shipped.

**Problem**: Two unrelated faults with one shape. The escape was unreachable because the gate read its labels and body from `github.event.pull_request.*`: applying it takes *two* changes, so `labeled` and `edited` fire within the same second, and under `cancel-in-progress: true` the two runs raced into cancelling each other — leaving an older, already-stale failure as the visible check. A re-run could not recover it either, because **a re-run replays the original event payload rather than re-reading the PR**, so the gate kept judging the PR as it was before the operator fixed it. The only working recovery was pushing an unrelated commit, which is precisely what an escape hatch exists to avoid. Meanwhile the adjacency report's workflow collected its feed into `$RUNNER_TEMP` and never passed `--adjacency-issues`: the feature had never once executed in CI, and its green fixtures said nothing about that, because they exercise the script and the missing wiring was not in the script.

**Solution**: For the escape — read labels/body/author live from the API through a thin CI-only adapter (a script, not inline `run:` shell, so the derivation stays reachable by tests per the BUG-063 remedy), fail closed if that read fails, and make `cancel-in-progress` conditional on `github.event.action` so a metadata event never kills a run in flight. Note the discriminator: `github.event_name` is `pull_request` for all six trigger types and cannot tell them apart. For the report — pass the flag. And then the part that actually closed the loop: the fixing PR needed a spec-gate escape of its own, so it *took* one, and the label-plus-rationale path was **observed** green (two metadata runs, both completing, no commit pushed) instead of argued green.

**Rule**: A green test proves the code; only an execution in the real pipeline proves the wiring. Before calling any conditional path done, ask when it last actually ran — not whether it is covered. Two shapes recur and both read as finished: a feature whose tests pass while its invocation is missing, and an escape hatch that is documented, code-pathed, and never taken. The cheapest remedy is to arrange for a change to exercise its own path once — if a fix repairs an escape, take that escape on the PR that ships it. Corollary worth its own line: in GitHub Actions a re-run is a **replay, not a retry**, so any decision derived from the event payload is frozen at the moment the event fired; read anything an operator can change while the PR is open from the API instead.
