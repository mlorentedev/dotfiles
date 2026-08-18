---
id: lesson-037-when-an-invariant-changes-dead-code-emerges-silent
type: lesson
status: active
created: "2026-05-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 037: When an invariant changes, dead code emerges silently downstream

**Context:** SDD-001 (PR #49) added an unconditional `[sdd]` reminder to `$ContextLines` / `CONTEXT_LINES` at the start of both `claude-session-start.{ps1,sh}`. The new invariant: the context buffer is NEVER empty. Two pre-existing branches gated on the OLD invariant (`if (-not $VaultRoot -and -not $ContextLines) { exit 0 }` and the bash equivalent) became unreachable code -- they fired the `exit 0` branch only when both were empty, which can no longer happen. Also: the claude-mem heal block's `$ContextLines = "..."` overwrite (instead of append) would have wiped the new reminder when heal output existed.

**Problem:** Dead code born from an invariant change is silently broken in a way that compiles and runs but does the wrong thing. Three failure modes: (1) the dead branch is never executed (silent waste), (2) the dead branch IS executed and produces incorrect behavior because its precondition is now impossible (rare, but happens when state is computed by side-effects), (3) downstream blocks that share state with the upstream invariant get clobbered (the claude-mem overwrite case). The bug surfaces only on the next refactor or in production when a corner case finally triggers the dead path.

**Solution:** When changing an invariant (especially one as fundamental as "this buffer is always non-empty"), do a Pre-Flight Audit per the AGENTS.md Socratic Guardrail: grep the entire file for references to the OLD invariant's preconditions (in this case, `(-not $ContextLines)` and `[ -z "$CONTEXT_LINES" ]`) and decide for each: (a) is this block now unreachable? remove it explicitly; (b) does this block read the buffer state and act on it? verify the new invariant doesn't break it; (c) does this block WRITE to the buffer? change overwrite to defensive append. Document the invariant change in a comment at the original assignment site so future readers see "this was made invariant by SDD-001 -- downstream blocks rely on it being non-empty".

**Rule:** Invariant changes are interface changes in disguise. Audit every reader AND writer of the affected state. A grep for the OLD invariant's predicate is the cheap mechanical step that catches the dead-code class. Skipping the audit = future-you debugs a 5-line silent regression for 30 minutes.

**Tags:** `#refactor` `#invariants` `#dead-code` `#pre-flight-audit` `#shell-state`
