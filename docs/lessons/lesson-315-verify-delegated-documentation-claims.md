---
id: lesson-315
type: lesson
status: active
created: "2026-09-24"
owner: manu
tags: [lesson, verification, subagents, claude-code, delegation]
---

# 315 — A subagent's documentation summary is a claim: check it against what is observable locally before acting on it

## What happened

While tuning Claude Code auto-compaction (2026-09-24), a `claude-code-guide` subagent was asked what `autoCompactWindow` means. It answered with citations that the setting is a percentage, default 80, and that the displayed 800k was 80% of the 1M window.

The claim contradicted local evidence already on screen. The user's `settings.json` held `"autoCompactWindow": 800000`, and `/context` printed "Auto-compact window: 800k tokens". Read as a percentage, 800000 would be meaningless. Acting on the summary would have put a percentage (e.g. 50) into the dotfiles template, where `50` is outside Claude Code's documented 100000–1000000 token range rather than configuring a 50% threshold. The citation URL was real; the content attributed to it was not.

## The rule

Before acting on a delegated lookup, check one load-bearing claim against the primary source, fetched directly (curl the doc page and grep the exact entry), and against any local observable (config value, command output). The settings reference says: "Type: number of tokens, from 100000 to 1000000 ... Default: unset". A cited URL shows where to look, not that the summary matches what is there. Relay the correction to the user explicitly rather than silently switching answers.
