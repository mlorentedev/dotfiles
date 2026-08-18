---
id: lesson-039-defensive-monitors-are-not-fixes-trigger-fix-and-m
type: lesson
status: active
created: "2026-05-19"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 039: Defensive monitors are not fixes — trigger fix and monitor are siblings, not substitutes

**Context:** dotfiles#33 was the "original" fix for the upstream `anthropics/claude-code#59870` truncation bug — every `claude plugin install` call rewrites `~/.claude/.claude.json` and silently drops subscription metadata (organizationType, organizationRateLimitTier, projects map, onboarding flags), shrinking the file from ~75 KB to ~1.5 KB and forcing re-authentication. The "fix" was an idempotence guard: don't call install if the plugin already appears in `claude plugin list` output. Six months later SDD-021 (2026-05-18) added a session-start canary (`Test-ClaudeJsonSize` in `claude-session-start.{sh,ps1}`) that flags the symptom if it ever recurs, with a 10 KB threshold. Today (2026-05-19) I noticed the canary firing on every session: file at 3444 bytes, re-login prompt in every project.

**Problem:** The idempotence guard had a false negative for `claude-mem@thedotmack` — it never appears in `claude plugin list` output (different marketplace, `@thedotmack` vs `@claude-plugins-official`), so the literal-match check `grep -qF "claude-mem@thedotmack"` returns false on every setup run, triggering one real install call and one real truncation. The SDD-021 canary CORRECTLY surfaced this — the warning text was in the session-start additionalContext I was reading at boot — but the canary is a detector, not a preventer. I had been blaming the recurrence on something I had "just broken" instead of recognising the canary was doing its job and the trigger fix from dotfiles#33 was incomplete.

**Solution:** Add a snapshot/restore wrapper around the install call (BUG-004 / PR pending) as a defense-in-depth layer beneath the existing idempotence guard. The wrapper snapshots `.claude.json` to a tempfile before the install, restores from snapshot in `finally` iff the post-call size dropped >50% from a baseline ≥10 KB (same threshold as SDD-021's canary — single SSOT for "anomalously small"). Now there are THREE layers that have to fail for the user to see re-login: (1) idempotence guard catches the common case, (2) wrapper catches the false-negative case, (3) canary alarms at next session start if both fail.

**Rule:** When you ship a monitor for a bug you "fixed", the monitor firing is evidence the fix was incomplete, not evidence the fix was undone. Before assuming "I broke it again", check whether the monitor was designed for the exact failure mode now appearing. Three-layer thinking: prevention (the trigger fix), detection (the monitor), recovery (the auto-restore). Each layer guards against the others failing. The presence of a monitor does NOT discharge the obligation to find the residual trigger — it just gives you a finite time window before the bug bites the user.

**Tags:** `#defense-in-depth` `#monitoring` `#claude-cli` `#setup-scripts` `#three-layer-thinking` `#upstream-bug`
