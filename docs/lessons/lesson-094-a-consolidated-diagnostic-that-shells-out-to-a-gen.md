---
id: lesson-094-a-consolidated-diagnostic-that-shells-out-to-a-gen
type: lesson
status: active
created: "2026-06-14"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 094: A consolidated diagnostic that shells out to a generator is on-demand-cheap but per-event-expensive

**Context**: `dotf doctor` (CLI-012) consolidates the 12-section healthcheck. One section gates on `compile-harness.sh --check`, which re-renders every skill record offline. The retired `claude-session-start.sh` used to run a light, env-contract-only `doctor.sh` on every Claude session start.

**Problem**: Repointing the per-session hook to the full `dotf doctor` would fork a ~2.8s sweep on **every** session start (the `compile-harness --check` re-render dominates the time), and a PATH-command call would also break the hermetic isolation the session-start test relies on — that test copies only the hook into a temp dir so sibling scripts are *absent* and skipped, an assumption a PATH binary violates. The faithful "just repoint it" would have shipped a silent latency + context-noise regression.

**Solution**: Retire the per-session drift block rather than repoint it; surface env-contract drift post-setup (`setup-linux.sh` runs `dotf doctor`) and on demand instead. A focused `dotf doctor --quick` (env-contract only, no harness gate) is tracked for the hook with the SessionStart hook port. Time the tool against the hot path before wiring it in.

**Rule**: A diagnostic that's fine to run by hand can be far too heavy for a per-event hook once it shells out to a generator or probes N tools. Before wiring a "do everything" command into a hot path (session start, pre-commit, prompt-submit), measure it and split a `--quick` subset — "it's the same checks" ignores that frequency, not check count, sets the cost budget.
