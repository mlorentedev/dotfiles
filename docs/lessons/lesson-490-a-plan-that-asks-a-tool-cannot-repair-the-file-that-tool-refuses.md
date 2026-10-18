---
id: "lesson-490-a-plan-that-asks-a-tool-cannot-repair-the-file-that-tool-refuses"
type: lesson
status: active
title: "A plan that asks a tool cannot repair the file that tool refuses"
created: "2026-10-18"
---

# A plan that asks a tool cannot repair the file that tool refuses


**Context:** `dotf tools sync` owns `~/.config/mise/conf.d/dotfiles.toml`. It rewrites the file whenever versions.conf renders something different. Apply runs the plan first, and the plan asks mise what is installed (`mise which`, `mise outdated`, `mise ls --prunable`) (#2760).

**Problem:** When that file does not parse, real mise refuses every command over it: `mise ls`, `outdated` and `ls --prunable` all exit 1 with "TOML parse error". So the plan failed before Apply could write, and the file the sync would have rewritten blocked its own rewrite. The only way out was deleting it by hand. Doctor, the converge and the dry run all reported an error rather than the repair.

**Solution:** The plan parses the deployed file before it asks mise anything. When the file does not parse, the plan returns the rewrite as its one change and asks mise nothing. Apply writes the rendered config, runs `mise install` against it and then verifies as usual, so the next plan reports nothing to do. The dry run of `dotf tools sync` and of the converge names the repair. Doctor fails, names the file, and says `dotf tools sync` rewrites it. The tests use a fake mise that refuses every command over an unparseable config, as the real one does (measured). The live dry run over a broken copy exits 0 and names the file, while `mise ls` over the same copy exits 1.

**Why:** A reconciler whose plan queries a consumer of the file it owns inherits that consumer's failure modes. When the consumer cannot read the file, the plan cannot run, and the repair never happens. Validate the owned file yourself, before the first query, and treat "unreadable" as a planned change rather than an error.
